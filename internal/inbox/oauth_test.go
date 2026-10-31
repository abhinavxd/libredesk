package inbox

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/crypto"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestUpdateOAuthConfig(t *testing.T) {
	db := testutil.NewDB(t, "inbox_oauth_updates")
	lo := logf.New(logf.Opts{})
	mgr, err := New(&lo, db, testutil.NewI18n(t), "01234567890123456789012345678901")
	require.NoError(t, err)
	base := imodels.Config{
		AuthType: imodels.AuthTypeOAuth2,
		OAuth: &imodels.OAuthConfig{
			Provider: "microsoft", TenantID: "tenant", ClientID: "client", ClientSecret: "delegated-secret",
			AccessToken: "old-token", RefreshToken: "old-refresh", ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
		IMAP: []imodels.IMAPConfig{{Host: "imap.example.com"}},
		SMTP: []imodels.SMTPConfig{{Host: "smtp.example.com"}},
	}
	create := func(t *testing.T) imodels.Inbox {
		t.Helper()
		raw, err := json.Marshal(base)
		require.NoError(t, err)
		record, err := mgr.Create(imodels.Inbox{Name: "Shared", Channel: ChannelEmail, From: "shared@example.com", Config: raw, Enabled: true})
		require.NoError(t, err)
		return record
	}
	read := func(t *testing.T, id int) imodels.Config {
		t.Helper()
		record, err := mgr.GetDBRecord(id)
		require.NoError(t, err)
		var cfg imodels.Config
		require.NoError(t, json.Unmarshal(record.Config, &cfg))
		return cfg
	}
	refreshed := *base.OAuth
	refreshed.AccessToken, refreshed.RefreshToken = "new-token", "new-refresh"

	t.Run("refresh preserves live non-OAuth fields and encrypts tokens", func(t *testing.T) {
		record := create(t)
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(record.Config, &fields))
		fields["reply_to"] = json.RawMessage(`"changed@example.com"`)
		fields["future_setting"] = json.RawMessage(`{"enabled":true}`)
		raw, err := json.Marshal(fields)
		require.NoError(t, err)
		require.NoError(t, mgr.UpdateConfig(record.ID, raw))
		require.NoError(t, mgr.UpdateOAuthConfig(record.ID, base.AuthType, base.OAuth, &refreshed))
		cfg := read(t, record.ID)
		require.Equal(t, "changed@example.com", cfg.ReplyTo)
		require.Equal(t, &refreshed, cfg.OAuth)
		require.Equal(t, imodels.AuthTypeOAuth2, cfg.AuthType)
		record, err = mgr.GetDBRecord(record.ID)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(record.Config, &fields))
		require.JSONEq(t, `{"enabled":true}`, string(fields["future_setting"]))
		var encrypted json.RawMessage
		require.NoError(t, db.Get(&encrypted, "SELECT config FROM inboxes WHERE id = $1", record.ID))
		var stored imodels.Config
		require.NoError(t, json.Unmarshal(encrypted, &stored))
		require.True(t, crypto.IsEncrypted(stored.OAuth.ClientSecret))
		require.True(t, crypto.IsEncrypted(stored.OAuth.AccessToken))
		require.True(t, crypto.IsEncrypted(stored.OAuth.RefreshToken))
		require.ErrorIs(t, mgr.UpdateOAuthConfig(record.ID, base.AuthType, base.OAuth, &refreshed), ErrOAuthConfigChanged)
	})

	for _, writer := range []string{"refresh", "reconnect"} {
		t.Run("stale "+writer+" cannot overwrite app-only conversion", func(t *testing.T) {
			record := create(t)
			appOnly := base
			appOnly.AuthType = imodels.AuthTypeOAuth2App
			appOnly.OAuth = &imodels.OAuthConfig{
				Provider: "microsoft", TenantID: "new-tenant", ClientID: "new-client", ClientSecret: "app-secret",
			}
			raw, err := json.Marshal(appOnly)
			require.NoError(t, err)
			encrypted, err := mgr.encryptInboxConfig(raw)
			require.NoError(t, err)
			tx, err := db.Beginx()
			require.NoError(t, err)
			defer func() { _ = tx.Rollback() }()
			var id int
			require.NoError(t, tx.Stmtx(mgr.queries.LockInbox).Get(&id, record.ID))
			_, err = tx.Stmtx(mgr.queries.UpdateConfig).Exec(record.ID, encrypted)
			require.NoError(t, err)

			incoming := refreshed
			if writer == "reconnect" {
				incoming.ClientSecret = "reconnected-secret"
			}
			done := make(chan error, 1)
			go func() {
				done <- mgr.UpdateOAuthConfig(record.ID, base.AuthType, base.OAuth, &incoming)
			}()
			require.Eventually(t, func() bool {
				var waiting bool
				err := db.Get(&waiting, `SELECT EXISTS (
					SELECT 1 FROM pg_stat_activity WHERE datname = current_database()
					AND wait_event_type = 'Lock' AND query LIKE 'SELECT id FROM inboxes%'
				)`)
				require.NoError(t, err)
				return waiting
			}, 5*time.Second, 10*time.Millisecond, "stale writer must wait on the conversion's row lock")
			require.NoError(t, tx.Commit())
			select {
			case err := <-done:
				require.ErrorIs(t, err, ErrOAuthConfigChanged)
			case <-time.After(5 * time.Second):
				t.Fatal("stale writer did not finish after the conversion committed")
			}
			cfg := read(t, record.ID)
			require.True(t, cfg.IsAppOnly())
			require.Equal(t, appOnly.OAuth, cfg.OAuth)
			require.Empty(t, cfg.OAuth.AccessToken)
			require.Empty(t, cfg.OAuth.RefreshToken)
			require.True(t, cfg.OAuth.ExpiresAt.IsZero())
			require.ErrorIs(t, mgr.UpdateOAuthConfig(record.ID, appOnly.AuthType, appOnly.OAuth, &incoming), ErrOAuthConfigChanged)
		})
	}

	t.Run("secret rotation rejects the old writer", func(t *testing.T) {
		record := create(t)
		rotated := refreshed
		rotated.ClientSecret = "rotated-secret"
		require.NoError(t, mgr.UpdateOAuthConfig(record.ID, base.AuthType, base.OAuth, &rotated))
		require.ErrorIs(t, mgr.UpdateOAuthConfig(record.ID, base.AuthType, base.OAuth, &refreshed), ErrOAuthConfigChanged)
		require.Equal(t, &rotated, read(t, record.ID).OAuth)
	})
}
