package twofactor

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"slices"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/hotp"
	"github.com/pquerna/otp/totp"
	"github.com/zerodha/logf"
)

const (
	maxAttempts       = 5
	lockDuration      = 5 * time.Minute
	recoveryCodeCount = 10
)

//go:embed queries.sql
var efs embed.FS

type Manager struct {
	db   *sqlx.DB
	q    queries
	key  string
	i18n *i18n.I18n
	lo   *logf.Logger
}

type queries struct {
	GetStatus        *sqlx.Stmt `query:"get-status"`
	BeginSetup       *sqlx.Stmt `query:"begin-setup"`
	LockEnrollment   *sqlx.Stmt `query:"lock-enrollment"`
	SaveEnrollment   *sqlx.Stmt `query:"save-enrollment"`
	DeleteEnrollment *sqlx.Stmt `query:"delete-enrollment"`
}

type Status struct {
	Enabled                bool `json:"enabled" db:"enabled"`
	RecoveryCodesRemaining int  `json:"recovery_codes_remaining" db:"recovery_codes_remaining"`
}

type Setup struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
}

type enrollment struct {
	Secret         string         `db:"secret"`
	Enabled        bool           `db:"enabled"`
	LastStep       int64          `db:"last_step"`
	RecoveryHashes pq.StringArray `db:"recovery_hashes"`
	FailedAttempts int            `db:"failed_attempts"`
	LockedUntil    sql.NullTime   `db:"locked_until"`
	SetupExpiresAt time.Time      `db:"setup_expires_at"`
}

func New(db *sqlx.DB, key string, translations *i18n.I18n, lo *logf.Logger) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}
	return &Manager{db: db, q: q, key: key, i18n: translations, lo: lo}, nil
}

func (m *Manager) Status(userID int) (Status, error) {
	var status Status
	if err := m.q.GetStatus.Get(&status, userID); err != nil {
		return status, m.dbError(err)
	}
	return status, nil
}

func (m *Manager) BeginSetup(userID int, account string) (Setup, error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Libredesk", AccountName: account, SecretSize: 20, Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	if err != nil {
		return Setup{}, m.dbError(err)
	}
	encrypted, err := crypto.Encrypt(key.Secret(), m.key)
	if err != nil {
		return Setup{}, m.dbError(err)
	}
	img, err := key.Image(256, 256)
	if err != nil {
		return Setup{}, m.dbError(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return Setup{}, m.dbError(err)
	}
	result, err := m.q.BeginSetup.Exec(userID, encrypted)
	if err != nil {
		return Setup{}, m.dbError(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Setup{}, m.dbError(err)
	}
	if count == 0 {
		return Setup{}, m.inputError("twoFactor.alreadyEnabled")
	}
	return Setup{Secret: key.Secret(), QRCode: "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())}, nil
}

func (m *Manager) ConfirmSetup(userID int, code string) ([]string, error) {
	return m.verifyAndUpdate(userID, code, "confirm", true)
}

func (m *Manager) Verify(userID int, code string) error {
	_, err := m.verifyAndUpdate(userID, code, "verify", true)
	return err
}

func (m *Manager) Disable(userID int, code string) error {
	_, err := m.verifyAndUpdate(userID, code, "disable", true)
	return err
}

func (m *Manager) RegenerateRecoveryCodes(userID int, code string) ([]string, error) {
	return m.verifyAndUpdate(userID, code, "regenerate", true)
}

func (m *Manager) DisableAfterVerification(userID int) error {
	_, err := m.verifyAndUpdate(userID, "", "disable", false)
	return err
}

func (m *Manager) RegenerateRecoveryCodesAfterVerification(userID int) ([]string, error) {
	return m.verifyAndUpdate(userID, "", "regenerate", false)
}

func (m *Manager) verifyAndUpdate(userID int, code, action string, verify bool) ([]string, error) {
	tx, err := m.db.Beginx()
	if err != nil {
		return nil, m.dbError(err)
	}
	defer tx.Rollback()
	// Verification and code consumption must hold the same row lock.
	var state enrollment
	if err := tx.Stmtx(m.q.LockEnrollment).Get(&state, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, m.inputError("twoFactor.setupExpired")
		}
		return nil, m.dbError(err)
	}
	now := time.Now()
	if state.LockedUntil.Valid && now.Before(state.LockedUntil.Time) {
		return nil, envelope.NewError(envelope.RateLimitError, m.i18n.T("twoFactor.tooManyAttempts"), nil)
	}
	if state.LockedUntil.Valid {
		state.FailedAttempts = 0
		state.LockedUntil = sql.NullTime{}
	}
	if action == "confirm" {
		if state.Enabled {
			return nil, m.inputError("twoFactor.alreadyEnabled")
		}
		if !now.Before(state.SetupExpiresAt) {
			return nil, m.inputError("twoFactor.setupExpired")
		}
	} else if !state.Enabled {
		return nil, m.inputError("twoFactor.notEnabled")
	}
	if verify {
		// Decrypt accepts legacy plaintext, which is not valid for authenticator secrets.
		if !crypto.IsEncrypted(state.Secret) {
			return nil, m.dbError(errors.New("unencrypted two-factor secret"))
		}
		secret, err := crypto.Decrypt(state.Secret, m.key)
		if err != nil {
			return nil, m.dbError(err)
		}
		code = strings.TrimSpace(code)
		step, valid := matchStep(secret, code, now, state.LastStep)
		if valid {
			state.LastStep = step
		} else if state.Enabled {
			if index := slices.Index(state.RecoveryHashes, recoveryHash(code)); index >= 0 {
				state.RecoveryHashes = slices.Delete(state.RecoveryHashes, index, index+1)
				valid = true
			}
		}
		if !valid {
			state.FailedAttempts++
			if state.FailedAttempts >= maxAttempts {
				state.LockedUntil = sql.NullTime{Time: now.Add(lockDuration), Valid: true}
			}
			if _, err := tx.Stmtx(m.q.SaveEnrollment).Exec(userID, state.Enabled, state.LastStep, state.RecoveryHashes, state.FailedAttempts, state.LockedUntil); err != nil {
				return nil, m.dbError(err)
			}
			// Rejected codes must still persist the attempt count.
			if err := tx.Commit(); err != nil {
				return nil, m.dbError(err)
			}
			if state.FailedAttempts >= maxAttempts {
				return nil, envelope.NewError(envelope.RateLimitError, m.i18n.T("twoFactor.tooManyAttempts"), nil)
			}
			return nil, m.inputError("twoFactor.invalidCode")
		}
	}
	var codes []string
	if action == "confirm" || action == "regenerate" {
		codes, state.RecoveryHashes, err = newRecoveryCodes()
		if err != nil {
			return nil, m.dbError(err)
		}
		state.Enabled = true
	}
	if action == "disable" {
		_, err = tx.Stmtx(m.q.DeleteEnrollment).Exec(userID)
	} else {
		_, err = tx.Stmtx(m.q.SaveEnrollment).Exec(userID, state.Enabled, state.LastStep, state.RecoveryHashes, 0, sql.NullTime{})
	}
	if err != nil {
		return nil, m.dbError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, m.dbError(err)
	}
	return codes, nil
}

func (m *Manager) inputError(key string) error {
	return envelope.NewError(envelope.InputError, m.i18n.T(key), nil)
}

func (m *Manager) dbError(err error) error {
	m.lo.Error("two-factor operation failed", "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}

func matchStep(secret, code string, now time.Time, lastStep int64) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	step := now.Unix() / 30
	for _, candidate := range []int64{step, step - 1, step + 1} {
		// Accepting a future step also invalidates all earlier steps.
		if candidate <= lastStep || candidate < 0 {
			continue
		}
		valid, err := hotp.ValidateCustom(code, uint64(candidate), secret, hotp.ValidateOpts{Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err == nil && valid {
			return candidate, true
		}
	}
	return 0, false
}

func newRecoveryCodes() ([]string, pq.StringArray, error) {
	codes := make([]string, recoveryCodeCount)
	hashes := make(pq.StringArray, recoveryCodeCount)
	for i := range recoveryCodeCount {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		text := hex.EncodeToString(raw)
		codes[i] = fmt.Sprintf("%s-%s-%s-%s", text[:8], text[8:16], text[16:24], text[24:])
		hashes[i] = recoveryHash(codes[i])
	}
	return codes, hashes, nil
}

func recoveryHash(code string) string {
	code = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
