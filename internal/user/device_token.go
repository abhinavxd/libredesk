package user

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
)

const (
	deviceTokenPrefix    = "ld"
	deviceTokenParts     = 3
	deviceTokenSelectorN = 16
	deviceTokenVerifierN = 32
	deviceTokenTouchGap  = 24 * time.Hour
)

var errMalformedDeviceToken = errors.New("malformed device token")

// MintDeviceToken issues a token for a device. The plaintext is returned once and never stored.
func (u *Manager) MintDeviceToken(userID int, name string) (string, models.DeviceToken, error) {
	var token models.DeviceToken

	selector, err := randomHex(deviceTokenSelectorN)
	if err != nil {
		u.lo.Error("error generating device token selector", "error", err)
		return "", token, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	verifier, err := randomHex(deviceTokenVerifierN)
	if err != nil {
		u.lo.Error("error generating device token verifier", "error", err)
		return "", token, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	hash := sha256.Sum256([]byte(verifier))
	if err := u.q.InsertDeviceToken.Get(&token, userID, strings.TrimSpace(name), selector, hash[:]); err != nil {
		u.lo.Error("error inserting device token", "error", err, "user_id", userID)
		return "", token, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return deviceTokenPrefix + "_" + selector + "_" + verifier, token, nil
}

// ValidateDeviceToken resolves a device token to its agent.
func (u *Manager) ValidateDeviceToken(token string) (models.User, error) {
	var user models.User

	selector, verifier, err := parseDeviceToken(token)
	if err != nil {
		return user, envelope.NewError(envelope.UnauthorizedError, u.i18n.T("validation.invalidCredential"), nil)
	}

	var stored models.DeviceToken
	if err := u.q.GetDeviceTokenBySelector.Get(&stored, selector); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, envelope.NewError(envelope.UnauthorizedError, u.i18n.T("validation.invalidCredential"), nil)
		}
		u.lo.Error("error fetching device token", "error", err)
		return user, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	hash := sha256.Sum256([]byte(verifier))
	if subtle.ConstantTimeCompare(hash[:], stored.VerifierHash) != 1 {
		return user, envelope.NewError(envelope.UnauthorizedError, u.i18n.T("validation.invalidCredential"), nil)
	}
	if stored.RevokedAt.Valid || time.Now().After(stored.ExpiresAt) {
		return user, envelope.NewError(envelope.UnauthorizedError, u.i18n.T("validation.invalidCredential"), nil)
	}

	// Sliding expiry is only written once it is a day stale, so validation is not a write per request.
	if !stored.LastUsedAt.Valid || time.Since(stored.LastUsedAt.Time) > deviceTokenTouchGap {
		if _, err := u.q.TouchDeviceToken.Exec(stored.ID); err != nil {
			u.lo.Error("error touching device token", "error", err, "id", stored.ID)
		}
	}

	return u.GetAgentCachedOrLoad(stored.UserID)
}

// GetDeviceTokens returns an agent's live device tokens.
func (u *Manager) GetDeviceTokens(userID int) ([]models.DeviceToken, error) {
	var tokens = make([]models.DeviceToken, 0)
	if err := u.q.GetDeviceTokens.Select(&tokens, userID); err != nil {
		u.lo.Error("error fetching device tokens", "error", err, "user_id", userID)
		return tokens, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tokens, nil
}

// RevokeDeviceToken revokes one of the agent's own tokens.
func (u *Manager) RevokeDeviceToken(userID, tokenID int) error {
	result, err := u.q.RevokeDeviceToken.Exec(tokenID, userID)
	if err != nil {
		u.lo.Error("error revoking device token", "error", err, "id", tokenID)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return envelope.NewError(envelope.NotFoundError, u.i18n.T("globals.messages.notFound"), nil)
	}
	return nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func parseDeviceToken(token string) (selector, verifier string, err error) {
	parts := strings.Split(token, "_")
	if len(parts) != deviceTokenParts || parts[0] != deviceTokenPrefix {
		return "", "", errMalformedDeviceToken
	}
	if parts[1] == "" || parts[2] == "" {
		return "", "", errMalformedDeviceToken
	}
	return parts[1], parts[2], nil
}
