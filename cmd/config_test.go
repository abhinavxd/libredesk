package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	oidcmodels "github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/zerodha/logf"
)

func TestIsLocalLoginEnabled(t *testing.T) {
	tests := []struct {
		name       string
		configured *bool
		providers  []oidcmodels.OIDC
		want       bool
	}{
		{name: "defaults to enabled", want: true},
		{name: "remains enabled when configured", configured: boolPointer(true), want: true},
		{name: "remains enabled without an enabled OIDC provider", configured: boolPointer(false), providers: []oidcmodels.OIDC{{Enabled: false}}, want: true},
		{name: "is disabled with an enabled OIDC provider", configured: boolPointer(false), providers: []oidcmodels.OIDC{{Enabled: true}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko.Delete(localLoginEnabledKey)
			if tt.configured != nil {
				if err := ko.Set(localLoginEnabledKey, *tt.configured); err != nil {
					t.Fatalf("set config: %v", err)
				}
			}
			t.Cleanup(func() { ko.Delete(localLoginEnabledKey) })

			if got := isLocalLoginEnabled(tt.providers); got != tt.want {
				t.Fatalf("isLocalLoginEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstallPreservesExistingDatabase(t *testing.T) {
	db := testutil.NewDB(t, "setup_preserve")
	db.MustExec(`CREATE TABLE install_sentinel (value text)`)
	db.MustExec(`INSERT INTO install_sentinel VALUES ('keep')`)
	for _, idempotent := range []bool{false, true} {
		if err := install(t.Context(), db, nil, idempotent, false /* prompt */); err != nil {
			t.Fatal(err)
		}
	}
	var value string
	if err := db.Get(&value, `SELECT value FROM install_sentinel`); err != nil || value != "keep" {
		t.Fatalf("existing data changed: value=%q err=%v", value, err)
	}
	db.MustExec(`DROP TABLE settings`)
	if err := install(t.Context(), db, nil, true /* idempotentInstall */, false /* prompt */); err == nil {
		t.Fatal("installation accepted an occupied, partially installed database")
	}
	if err := db.Get(&value, `SELECT value FROM install_sentinel`); err != nil || value != "keep" {
		t.Fatalf("partial database changed: value=%q err=%v", value, err)
	}
}

func TestFreshInstallIsAtomicAndRerunnable(t *testing.T) {
	db := testutil.NewDB(t, "setup_install")
	db.MustExec(`DROP SCHEMA public CASCADE`)
	db.MustExec(`CREATE SCHEMA public`)
	fs, err := stuffbin.NewLocalFS("/", "../schema.sql:schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIBREDESK_SYSTEM_USER_PASSWORD", "")
	if err := install(t.Context(), db, fs, true /* idempotentInstall */, false /* prompt */); err != nil {
		t.Fatal(err)
	}
	required, err := setupRequired(db)
	if err != nil || !required {
		t.Fatalf("fresh installation must require browser setup: %v", err)
	}
	if err := install(t.Context(), db, fs, false /* idempotentInstall */, false /* prompt */); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM users WHERE email = 'System'`)
	if count != 1 {
		t.Fatalf("system user count = %d", count)
	}
}

func TestSetupAccountRejectsInvalidAndExpiredCodes(t *testing.T) {
	db := testutil.NewDB(t, "setup_invalid")
	req := setupAccount{Token: strings.Repeat("a", 40), FirstName: "Admin", Email: "admin@example.com", Password: "SetupPass123!"}
	if err := createSetupAccount(t.Context(), db, req); err == nil {
		t.Fatal("existing installation accepted a setup account")
	}
	seedSetupCode(t, db, req.Token, time.Now().Add(time.Hour))
	invalid := req
	invalid.Token = strings.Repeat("b", 40)
	if err := createSetupAccount(t.Context(), db, invalid); err == nil {
		t.Fatal("wrong code accepted")
	}
	invalid = req
	invalid.Password = "short"
	if err := createSetupAccount(t.Context(), db, invalid); err == nil {
		t.Fatal("weak password accepted")
	}
	seedSetupCode(t, db, req.Token, time.Now().Add(-time.Second))
	if err := createSetupAccount(t.Context(), db, req); err == nil {
		t.Fatal("expired code accepted")
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM users WHERE type = 'agent'`); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid requests created %d agents", count)
	}
}

func TestSetupCodeCreatesExactlyOneAdmin(t *testing.T) {
	db := testutil.NewDB(t, "setup_once")
	token := strings.Repeat("a", 40)
	seedSetupCode(t, db, token, time.Now().Add(time.Hour))
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Go(func() {
			results <- createSetupAccount(t.Context(), db, setupAccount{Token: token, FirstName: "Admin", Email: fmt.Sprintf("admin%d@example.com", i), Password: "SetupPass123!"})
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("created %d accounts using one setup code", successes)
	}
	logger := logf.New(logf.Opts{Level: logf.FatalLevel})
	manager, err := user.New(testutil.NewI18n(t), user.Opts{DB: db, Lo: &logger})
	if err != nil {
		t.Fatal(err)
	}
	var email string
	if err := db.Get(&email, `SELECT email FROM users WHERE type = 'agent'`); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.VerifyPassword(email, []byte("SetupPass123!")); err != nil {
		t.Fatalf("new admin cannot sign in: %v", err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM user_roles JOIN roles ON roles.id = user_roles.role_id WHERE roles.name = 'Admin'`)
	if count != 1 {
		t.Fatalf("admin role count = %d", count)
	}
	required, err := setupRequired(db)
	if err != nil || required {
		t.Fatalf("setup stayed open: required=%v err=%v", required, err)
	}
}

func TestSetupChecklistRequiresDeliveredPublicReply(t *testing.T) {
	db := testutil.NewDB(t, "setup_checklist")
	db.MustExec(`INSERT INTO inboxes (name, channel) VALUES ('Test', 'livechat')`)
	db.MustExec(`INSERT INTO users (type, first_name) VALUES ('contact', 'Visitor'), ('agent', 'Agent')`)
	db.MustExec(`INSERT INTO conversations (inbox_id, contact_id, status_id) SELECT 1, 1, id FROM conversation_statuses LIMIT 1`)
	assertComplete := func(want bool) {
		t.Helper()
		status, err := getSetupChecklist(db)
		if err != nil {
			t.Fatal(err)
		}
		if got := status.Conversation != ""; got != want {
			t.Fatalf("conversation complete = %v, want %v", got, want)
		}
	}
	assertComplete(false)
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type) VALUES ('incoming', 'received', 1, 1, 'contact')`)
	assertComplete(false)
	db.MustExec(`INSERT INTO conversation_messages (type, status, private, conversation_id, sender_id, sender_type) VALUES ('outgoing', 'sent', true, 1, 2, 'agent')`)
	assertComplete(false)
	db.MustExec(`UPDATE conversation_messages SET private = false, status = 'failed' WHERE type = 'outgoing'`)
	assertComplete(false)
	db.MustExec(`UPDATE conversation_messages SET status = 'pending' WHERE type = 'outgoing'`)
	assertComplete(false)
	db.MustExec(`UPDATE conversation_messages SET status = 'sent' WHERE type = 'outgoing'`)
	assertComplete(true)
}

func TestInstallationChecksUploadAndCookieSettings(t *testing.T) {
	config := koanf.New(".")
	config.Set("upload.provider", "fs")
	config.Set("upload.fs.upload_path", filepath.Join(t.TempDir(), "missing"))
	if err := checkUploadDirectory(config); err == nil {
		t.Fatal("missing upload directory passed")
	}
	config.Set("upload.fs.upload_path", t.TempDir())
	if err := checkUploadDirectory(config); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		url           string
		disableSecure bool
		wantError     bool
	}{
		{"http://localhost:9000", false, false},
		{"http://support.example.com", false, true},
		{"https://support.example.com", false, false},
		{"https://support.example.com", true, true},
	} {
		config.Set("app.root_url", tt.url)
		config.Set("app.server.disable_secure_cookies", tt.disableSecure)
		if err := checkCookieConfiguration(config); (err != nil) != tt.wantError {
			t.Fatalf("cookie check for %s: %v", tt.url, err)
		}
	}
}

func seedSetupCode(t *testing.T, db *sqlx.DB, token string, expiry time.Time) {
	t.Helper()
	hash := sha256.Sum256([]byte(token))
	db.MustExec(`INSERT INTO settings (key, value) VALUES ('setup.bootstrap', jsonb_build_object('required', true, 'token_hash', $1::text, 'expires', $2::bigint)) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, hex.EncodeToString(hash[:]), expiry.Unix())
}

func boolPointer(value bool) *bool {
	return &value
}
