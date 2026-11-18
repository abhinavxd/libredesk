package inbox

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/crypto"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	return &Manager{
		lo:            &lo,
		encryptionKey: "0123456789abcdef0123456789abcdef",
	}
}

func configMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestMergeLiveChatSecrets(t *testing.T) {
	m := testManager(t)
	dummy := strings.Repeat(stringutil.PasswordDummy, 10)
	current := json.RawMessage(`{"theme":"light","turnstile_site_key":"site","turnstile_secret_key":"stored-secret"}`)

	t.Run("keeps the stored secret when the update is masked", func(t *testing.T) {
		merged, err := m.MergeLiveChatSecrets(current, json.RawMessage(`{"theme":"dark","turnstile_site_key":"site","turnstile_secret_key":"`+dummy+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		cfg := configMap(t, merged)
		if cfg["turnstile_secret_key"] != "stored-secret" || cfg["theme"] != "dark" {
			t.Fatalf("merged = %#v", cfg)
		}
	})

	t.Run("keeps the stored secret when the update secret is blank", func(t *testing.T) {
		merged, err := m.MergeLiveChatSecrets(current, json.RawMessage(`{"turnstile_site_key":"site","turnstile_secret_key":""}`))
		if err != nil {
			t.Fatal(err)
		}
		if configMap(t, merged)["turnstile_secret_key"] != "stored-secret" {
			t.Fatalf("merged = %s", merged)
		}
	})

	t.Run("stores a new secret", func(t *testing.T) {
		merged, err := m.MergeLiveChatSecrets(current, json.RawMessage(`{"turnstile_site_key":"site","turnstile_secret_key":"new-secret"}`))
		if err != nil {
			t.Fatal(err)
		}
		if configMap(t, merged)["turnstile_secret_key"] != "new-secret" {
			t.Fatalf("merged = %s", merged)
		}
	})

	t.Run("clears the secret when the site key is removed", func(t *testing.T) {
		merged, err := m.MergeLiveChatSecrets(current, json.RawMessage(`{"turnstile_site_key":"  ","turnstile_secret_key":"`+dummy+`","theme":"light"}`))
		if err != nil {
			t.Fatal(err)
		}
		cfg := configMap(t, merged)
		if cfg["turnstile_site_key"] != "" || cfg["turnstile_secret_key"] != "" {
			t.Fatalf("merged = %#v", cfg)
		}
	})
}

func TestTurnstileSecretEncryptRoundTrip(t *testing.T) {
	m := testManager(t)
	raw := json.RawMessage(`{"turnstile_site_key":"site","turnstile_secret_key":"super-secret","theme":"light"}`)
	encrypted, err := m.encryptInboxConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	stored := configMap(t, encrypted)["turnstile_secret_key"].(string)
	if !crypto.IsEncrypted(stored) {
		t.Fatalf("stored secret = %q", stored)
	}
	if configMap(t, encrypted)["turnstile_site_key"] != "site" {
		t.Fatal("site key was encrypted")
	}
	decrypted, err := m.decryptInboxConfig(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if configMap(t, decrypted)["turnstile_secret_key"] != "super-secret" {
		t.Fatalf("decrypted = %s", decrypted)
	}
}

func TestClearPasswordsMasksTurnstileSecret(t *testing.T) {
	inbox := imodels.Inbox{
		Channel: ChannelLiveChat,
		Secret:  null.StringFrom("widget-secret"),
		Config:  json.RawMessage(`{"turnstile_site_key":"site","turnstile_secret_key":"super-secret"}`),
	}
	if err := inbox.ClearPasswords(); err != nil {
		t.Fatal(err)
	}
	cfg := configMap(t, inbox.Config)
	dummy := strings.Repeat(stringutil.PasswordDummy, 10)
	if cfg["turnstile_secret_key"] != dummy {
		t.Fatalf("secret = %#v", cfg["turnstile_secret_key"])
	}
	if cfg["turnstile_site_key"] != "site" {
		t.Fatalf("site key = %#v", cfg["turnstile_site_key"])
	}
	if inbox.Secret.String != dummy {
		t.Fatalf("inbox secret = %q", inbox.Secret.String)
	}
}
