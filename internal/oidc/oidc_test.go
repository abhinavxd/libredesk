package oidc

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func newTestManager(t *testing.T, cfgID, cfgSecret string) *Manager {
	m := &Manager{i18n: testutil.NewI18n(t)}
	m.setConfigCredentials(cfgID, cfgSecret)
	return m
}

func TestSecretFromConfig(t *testing.T) {
	tests := []struct {
		name, cfgID, cfgSecret, clientID string
		want                             bool
	}{
		{"matching id", "zitadel", "s", "zitadel", true},
		{"non-matching id", "zitadel", "s", "google", false},
		{"unset", "", "", "zitadel", false},
		{"id without secret", "zitadel", "", "zitadel", false},
		{"secret without id", "", "s", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := newTestManager(t, tc.cfgID, tc.cfgSecret).secretFromConfig(tc.clientID); got != tc.want {
				t.Errorf("secretFromConfig(%q) = %v, want %v", tc.clientID, got, tc.want)
			}
		})
	}
}

func TestClientSecret(t *testing.T) {
	o := newTestManager(t, "zitadel", "from-config")
	if got := o.ClientSecret(models.OIDC{ClientID: "zitadel", ClientSecret: "stored"}); got != "from-config" {
		t.Errorf("matching client id: got %q, want config secret", got)
	}
	if got := o.ClientSecret(models.OIDC{ClientID: "google", ClientSecret: "stored"}); got != "stored" {
		t.Errorf("other client id: got %q, want stored secret", got)
	}
	if got := newTestManager(t, "", "").ClientSecret(models.OIDC{ClientID: "zitadel", ClientSecret: "stored"}); got != "stored" {
		t.Errorf("unset config: got %q, want stored secret", got)
	}
}

func TestLockConfigCredentials(t *testing.T) {
	o := newTestManager(t, "zitadel", "from-config")
	current := models.OIDC{ClientID: "zitadel", ClientSecret: "stored"}

	t.Run("empty secret keeps stored", func(t *testing.T) {
		got, err := o.lockConfigCredentials(current, models.OIDC{Name: "renamed", ClientID: "zitadel"})
		if err != nil {
			t.Fatal(err)
		}
		if got.ClientSecret != "stored" || got.Name != "renamed" {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("client id change rejected", func(t *testing.T) {
		if _, err := o.lockConfigCredentials(current, models.OIDC{ClientID: "other"}); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("new secret rejected", func(t *testing.T) {
		if _, err := o.lockConfigCredentials(current, models.OIDC{ClientID: "zitadel", ClientSecret: "new"}); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("provider not config-managed is untouched", func(t *testing.T) {
		req := models.OIDC{ClientID: "other", ClientSecret: "new"}
		got, err := o.lockConfigCredentials(models.OIDC{ClientID: "google", ClientSecret: "stored"}, req)
		if err != nil || got != req {
			t.Errorf("got %+v, %v", got, err)
		}
	})
}

func TestValidateCredentialsFromConfig(t *testing.T) {
	o := newTestManager(t, "zitadel", "from-config")
	if err := o.validateCredentials(models.OIDC{ClientID: "zitadel"}); err != nil {
		t.Errorf("config-managed provider without secret: %v", err)
	}
	if err := o.validateCredentials(models.OIDC{ClientID: "google"}); err == nil {
		t.Error("expected error for missing secret on other provider")
	}
	if err := o.validateCredentials(models.OIDC{ClientSecret: "s"}); err == nil {
		t.Error("expected error for missing client id")
	}
}
