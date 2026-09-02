package inbox

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestInboxEmailAddressOwnership(t *testing.T) {
	db := testutil.NewDB(t, "inbox_email_ownership")
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
	_, err = mgr.Create(makeInbox("Duplicate primary", "SUPPORT@example.com"))
	require.Error(t, err)
	_, err = mgr.Create(makeInbox("Alias hits primary", "sales@example.com", "support@example.com"))
	require.Error(t, err)
	_, err = mgr.Create(makeInbox("Alias hits alias", "help@example.com", "BILLING@example.com"))
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
	_, err = mgr.Create(makeInbox("Released alias", "billing@example.com"))
	require.NoError(t, err)
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

	require.NoError(t, mgr.SoftDelete(ctx, first.ID))
	_, err = mgr.Create(makeInbox("Released primary", "support@example.com", "accounts@example.com"))
	require.NoError(t, err)
}

func TestConcurrentInboxAddressClaim(t *testing.T) {
	db := testutil.NewDB(t, "concurrent_inbox_email_claim")
	lo := logf.New(logf.Opts{})
	mgr, err := New(&lo, db, testutil.NewI18n(t), "01234567890123456789012345678901")
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := mgr.Create(imodels.Inbox{Name: "Concurrent", Channel: ChannelEmail, From: "shared@example.com", Enabled: true, Config: json.RawMessage(`{}`)})
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var successes int
	for err := range results {
		if err == nil {
			successes++
		}
	}
	require.Equal(t, 1, successes)
}
