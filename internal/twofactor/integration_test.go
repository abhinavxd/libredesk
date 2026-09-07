package twofactor

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/pquerna/otp/totp"
	"github.com/zerodha/logf"
)

func TestEnrollmentAndRecovery(t *testing.T) {
	m, db, id := newTestManager(t, "twofactor_lifecycle")
	setup, err := m.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	db.Get(&stored, `SELECT secret FROM user_two_factor WHERE user_id=$1`, id)
	if !strings.HasPrefix(stored, "enc:") || strings.Contains(stored, setup.Secret) {
		t.Fatal("unencrypted secret")
	}
	status, err := m.Status(id)
	if err != nil || status.Enabled {
		t.Fatalf("setup enabled before verification: %+v %v", status, err)
	}
	if err := m.Verify(id, "123456"); err == nil {
		t.Fatal("pending setup accepted for login")
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	codes, err := m.ConfirmSetup(id, code)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.BeginSetup(id, "agent@example.com"); err == nil {
		t.Fatal("overwrote active enrollment")
	}
	if err := m.Verify(id, code); err == nil {
		t.Fatal("reused setup code")
	}
	if err := m.Verify(id, codes[0]); err != nil {
		t.Fatal(err)
	}
	if err := m.Verify(id, codes[0]); err == nil {
		t.Fatal("reused recovery code")
	}
	status, _ = m.Status(id)
	if !status.Enabled || status.RecoveryCodesRemaining != 9 {
		t.Fatalf("unexpected status %+v", status)
	}
	regenerated, err := m.RegenerateRecoveryCodes(id, codes[1])
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Verify(id, codes[2]); err == nil {
		t.Fatal("accepted superseded recovery code")
	}
	if err := m.Disable(id, regenerated[0]); err != nil {
		t.Fatal(err)
	}
	status, _ = m.Status(id)
	if status.Enabled {
		t.Fatal("disable failed")
	}
}

func TestAuthenticatorLoginAndReplay(t *testing.T) {
	m, _, id := newTestManager(t, "twofactor_totp")
	setup, err := m.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	if _, err := m.ConfirmSetup(id, code); err != nil {
		t.Fatal(err)
	}
	current, _ := totp.GenerateCode(setup.Secret, time.Now().Add(30*time.Second))
	if err := m.Verify(id, current); err != nil {
		t.Fatal(err)
	}
	if err := m.Verify(id, current); err == nil {
		t.Fatal("reused authentication code")
	}
}

func TestConcurrentRecoveryUse(t *testing.T) {
	m, _, id := newTestManager(t, "twofactor_concurrent")
	codes := enroll(t, m, id)
	var wg sync.WaitGroup
	var successes atomic.Int32
	for range 4 {
		wg.Go(func() {
			if m.Verify(id, codes[0]) == nil {
				successes.Add(1)
			}
		})
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("recovery code succeeded %d times", successes.Load())
	}
}

func TestVerificationLockout(t *testing.T) {
	m, db, id := newTestManager(t, "twofactor_lockout")
	codes := enroll(t, m, id)
	for range 5 {
		if m.Verify(id, "invalid") == nil {
			t.Fatal("invalid code accepted")
		}
	}
	if m.Verify(id, codes[0]) == nil {
		t.Fatal("valid code bypassed lockout")
	}
	db.MustExec(`UPDATE user_two_factor SET locked_until=NOW()-INTERVAL '1 second' WHERE user_id=$1`, id)
	if err := m.Verify(id, codes[0]); err != nil {
		t.Fatalf("could not recover after lockout: %v", err)
	}
}

func TestSetupExpires(t *testing.T) {
	m, db, id := newTestManager(t, "twofactor_expiry")
	setup, err := m.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`UPDATE user_two_factor SET setup_expires_at=NOW()-INTERVAL '1 second' WHERE user_id=$1`, id)
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	if _, err := m.ConfirmSetup(id, code); err == nil {
		t.Fatal("expired enrollment accepted")
	}
}

func newTestManager(t *testing.T, name string) (*Manager, *sqlx.DB, int) {
	t.Helper()
	db := testutil.NewDB(t, name)
	logger := logf.New(logf.Opts{})
	m, err := New(db, "01234567890123456789012345678901", testutil.NewI18n(t), &logger)
	if err != nil {
		t.Fatal(err)
	}
	var id int
	if err := db.Get(&id, `INSERT INTO users (type,email,first_name,last_name) VALUES ('agent','agent@example.com','Test','Agent') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	return m, db, id
}

func enroll(t *testing.T, m *Manager, id int) []string {
	t.Helper()
	setup, err := m.BeginSetup(id, "agent@example.com")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	codes, err := m.ConfirmSetup(id, code)
	if err != nil {
		t.Fatal(err)
	}
	return codes
}
