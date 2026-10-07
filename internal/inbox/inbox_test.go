package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/crypto"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type aliasVerificationTestInbox struct {
	EmailInbox
	send func(string, string) error
}

func (inb *aliasVerificationTestInbox) StartAliasVerification(alias, token string) error {
	return inb.send(alias, token)
}

func TestInboxEmailAddresses(t *testing.T) {
	db := testutil.NewDB(t, "inbox_email_addresses")
	ctx := context.Background()
	lo := logf.New(logf.Opts{})
	mgr, err := New(&lo, db, testutil.NewI18n(t), "01234567890123456789012345678901")
	require.NoError(t, err)

	config, err := json.Marshal(imodels.Config{
		IMAP: []imodels.IMAPConfig{{Host: "imap.example.com"}},
		SMTP: []imodels.SMTPConfig{{Host: "smtp.example.com"}},
	})
	require.NoError(t, err)
	makeInbox := func(name, from string, aliases ...string) imodels.Inbox {
		aliasValues := make(imodels.EmailAliases, len(aliases))
		for i, alias := range aliases {
			aliasValues[i] = imodels.EmailAlias{Email: alias}
		}
		return imodels.Inbox{Name: name, Channel: ChannelEmail, From: from, Aliases: aliasValues, Enabled: true, Config: config}
	}

	first, err := mgr.Create(makeInbox("Support", "support@example.com", "billing@example.com"))
	require.NoError(t, err)
	_, err = mgr.Create(makeInbox("Shared primary", "SUPPORT@example.com", "billing@example.com"))
	require.NoError(t, err)
	_, err = mgr.Create(makeInbox("Alias repeats own primary", "sales@example.com", "SALES@example.com"))
	require.Error(t, err)
	_, err = mgr.Create(makeInbox("Alias listed twice", "help@example.com", "desk@example.com", "DESK@example.com"))
	require.Error(t, err)

	updated, err := mgr.Update(first.ID, makeInbox("Support", "support@example.com", "accounts@example.com"))
	require.NoError(t, err)
	require.Equal(t, "accounts@example.com", updated.Aliases[0].Email)
	var status string
	_, err = db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = 'verified' WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID)
	require.NoError(t, err)
	_, err = mgr.Update(first.ID, makeInbox("Support updated", "support@example.com", "accounts@example.com"))
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT verification_status FROM inbox_email_addresses WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID).Scan(&status))
	require.Equal(t, "verified", status)
	_, err = db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = 'pending', verification_token = 'pending-token', verification_started_at = '2025-01-01 12:00:00+00' WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID)
	require.NoError(t, err)
	_, err = mgr.Update(first.ID, makeInbox("Support pending", "support@example.com", "accounts@example.com"))
	require.NoError(t, err)
	var verificationToken string
	var verificationStartedAt string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT verification_token, verification_started_at::text FROM inbox_email_addresses WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID).Scan(&verificationToken, &verificationStartedAt))
	require.Equal(t, "pending-token", verificationToken)
	require.Contains(t, verificationStartedAt, "2025-01-01 12:00:00")
	_, err = db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = 'pending', verification_token = 'old-token' WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID)
	require.NoError(t, err)
	require.NoError(t, mgr.CompleteAliasVerification(context.Background(), first.ID, "old-token", "accounts@example.com"))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT verification_status FROM inbox_email_addresses WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID).Scan(&status))
	require.Equal(t, "verified", status)
	_, err = db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = 'pending', verification_token = 'actual-token' WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID)
	require.NoError(t, err)
	require.NoError(t, mgr.CompleteAliasVerification(context.Background(), first.ID, "bad-token", "accounts@example.com"))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT verification_status FROM inbox_email_addresses WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID).Scan(&status))
	require.Equal(t, "pending", status)

	var addressID int
	require.NoError(t, db.Get(&addressID, `SELECT id FROM inbox_email_addresses WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID))
	for range 5 {
		_, err = mgr.queries.StartAliasVerification.Exec(first.ID, "accounts@example.com", imodels.AliasVerificationPending, "concurrent-token")
		require.NoError(t, err)
		start := make(chan struct{})
		results := make(chan error, 3)
		for range 2 {
			go func() {
				<-start
				_, err := mgr.Update(first.ID, makeInbox("Concurrent save", "support@example.com", "accounts@example.com"))
				results <- err
			}()
		}
		go func() {
			<-start
			results <- mgr.CompleteAliasVerification(ctx, first.ID, "concurrent-token", "accounts@example.com")
		}()
		close(start)
		for range 3 {
			require.NoError(t, <-results)
		}
		record, err := mgr.GetDBRecord(first.ID)
		require.NoError(t, err)
		require.Equal(t, imodels.AliasVerificationVerified, record.Aliases[0].VerificationStatus)
		var savedID int
		require.NoError(t, db.Get(&savedID, `SELECT id FROM inbox_email_addresses WHERE inbox_id = $1 AND email = 'accounts@example.com'`, first.ID))
		require.Equal(t, addressID, savedID)
	}

	omitted := makeInbox("Omitted aliases", "support@example.com")
	omitted.Aliases = nil
	updated, err = mgr.Update(first.ID, omitted)
	require.NoError(t, err)
	require.Len(t, updated.Aliases, 1)
	require.Equal(t, imodels.AliasVerificationVerified, updated.Aliases[0].VerificationStatus)

	mgr.inboxes[first.ID] = &aliasVerificationTestInbox{send: func(alias, token string) error {
		return errors.New("SMTP rejected sender")
	}}
	require.Error(t, mgr.StartAliasVerification(ctx, first.ID, "accounts@example.com"))
	record, err := mgr.GetDBRecord(first.ID)
	require.NoError(t, err)
	require.Equal(t, imodels.AliasVerificationFailed, record.Aliases[0].VerificationStatus)

	mgr.inboxes[first.ID] = &aliasVerificationTestInbox{send: func(alias, token string) error {
		return mgr.CompleteAliasVerification(ctx, first.ID, token, alias)
	}}
	require.NoError(t, mgr.StartAliasVerification(ctx, first.ID, "accounts@example.com"))
	record, err = mgr.GetDBRecord(first.ID)
	require.NoError(t, err)
	require.Equal(t, imodels.AliasVerificationVerified, record.Aliases[0].VerificationStatus)

	_, err = mgr.queries.StartAliasVerification.Exec(first.ID, "accounts@example.com", imodels.AliasVerificationPending, "latest-token")
	require.NoError(t, err)
	_, err = mgr.queries.FailAliasVerification.Exec(first.ID, "accounts@example.com", imodels.AliasVerificationFailed, "obsolete-token")
	require.NoError(t, err)
	require.NoError(t, mgr.CompleteAliasVerification(ctx, first.ID, "latest-token", "accounts@example.com"))
	record, err = mgr.GetDBRecord(first.ID)
	require.NoError(t, err)
	require.Equal(t, imodels.AliasVerificationVerified, record.Aliases[0].VerificationStatus)

	_, err = mgr.queries.StartAliasVerification.Exec(first.ID, "accounts@example.com", imodels.AliasVerificationPending, "stale-token")
	require.NoError(t, err)
	require.NoError(t, mgr.ExpireAliasVerifications(ctx, first.ID, time.Now().Add(-time.Minute)))
	record, err = mgr.GetDBRecord(first.ID)
	require.NoError(t, err)
	require.Equal(t, imodels.AliasVerificationPending, record.Aliases[0].VerificationStatus)
	require.NoError(t, mgr.ExpireAliasVerifications(ctx, first.ID, time.Now().Add(time.Minute)))
	record, err = mgr.GetDBRecord(first.ID)
	require.NoError(t, err)
	require.Equal(t, imodels.AliasVerificationFailed, record.Aliases[0].VerificationStatus)
	require.Nil(t, record.Aliases[0].VerifiedAt)

	var newConfig imodels.Config
	require.NoError(t, json.Unmarshal(record.Config, &newConfig))
	newConfig.SMTP[0].Host = "replacement.example.com"
	record.Config, err = json.Marshal(newConfig)
	require.NoError(t, err)
	updated, err = mgr.Update(first.ID, record)
	require.NoError(t, err)
	require.Equal(t, imodels.AliasVerificationNotVerified, updated.Aliases[0].VerificationStatus)
	require.Nil(t, updated.Aliases[0].VerifiedAt)

	updated.Aliases = imodels.EmailAliases{}
	updated, err = mgr.Update(first.ID, updated)
	require.NoError(t, err)
	require.Empty(t, updated.Aliases)

}

func TestLivechatSecretStaysEncryptedOnUpdate(t *testing.T) {
	db := testutil.NewDB(t, "inbox_livechat_secret")
	lo := logf.New(logf.Opts{})
	mgr, err := New(&lo, db, testutil.NewI18n(t), "01234567890123456789012345678901")
	require.NoError(t, err)

	created, err := mgr.Create(imodels.Inbox{Name: "Chat", Channel: ChannelLiveChat, Enabled: true, Config: json.RawMessage(`{}`), Secret: null.StringFrom("widget-secret")})
	require.NoError(t, err)
	created.Secret = null.StringFrom(strings.Repeat(stringutil.PasswordDummy, 10))
	_, err = mgr.Update(created.ID, created)
	require.NoError(t, err)

	var stored string
	require.NoError(t, db.Get(&stored, `SELECT secret FROM inboxes WHERE id = $1`, created.ID))
	require.True(t, crypto.IsEncrypted(stored))
	record, err := mgr.GetDBRecord(created.ID)
	require.NoError(t, err)
	require.Equal(t, "widget-secret", record.Secret.String)
}

func TestEmailSendingConfigChanged(t *testing.T) {
	base := imodels.Config{
		AuthType: imodels.AuthTypeOAuth2,
		SMTP:     []imodels.SMTPConfig{{Host: "smtp.example.com", Port: 587, Username: "support@example.com", Password: "old", AuthProtocol: "plain"}},
		OAuth:    &imodels.OAuthConfig{Provider: "google", ClientID: "client", RefreshToken: "refresh"},
	}
	for _, tt := range []struct {
		name   string
		change func(*imodels.Config)
		want   bool
	}{
		{"unchanged", func(c *imodels.Config) {}, false},
		{"host", func(c *imodels.Config) { c.SMTP[0].Host = "other.example.com" }, true},
		{"port", func(c *imodels.Config) { c.SMTP[0].Port = 465 }, true},
		{"username", func(c *imodels.Config) { c.SMTP[0].Username = "other@example.com" }, true},
		{"password", func(c *imodels.Config) { c.SMTP[0].Password = "new" }, true},
		{"auth protocol", func(c *imodels.Config) { c.SMTP[0].AuthProtocol = "login" }, true},
		{"auth type", func(c *imodels.Config) { c.AuthType = imodels.AuthTypePassword }, true},
		{"additional server", func(c *imodels.Config) { c.SMTP = append(c.SMTP, imodels.SMTPConfig{Host: "extra.example.com"}) }, true},
		{"OAuth account", func(c *imodels.Config) { c.OAuth.RefreshToken = "other-refresh" }, true},
		{"OAuth client", func(c *imodels.Config) { c.OAuth.ClientID = "other-client" }, true},
		{"OAuth token refresh", func(c *imodels.Config) { c.OAuth.AccessToken = "new-access" }, false},
		{"pool timeout", func(c *imodels.Config) { c.SMTP[0].PoolWaitTimeout = "30s" }, false},
		{"IMAP change", func(c *imodels.Config) { c.IMAP = []imodels.IMAPConfig{{Host: "other.example.com"}} }, false},
		{"display name", func(c *imodels.Config) { c.FromNameTemplate = "New name" }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			updated := base
			updated.SMTP = slices.Clone(base.SMTP)
			oauth := *base.OAuth
			updated.OAuth = &oauth
			tt.change(&updated)
			require.Equal(t, tt.want, emailSendingConfigChanged(base, updated))
		})
	}
}
