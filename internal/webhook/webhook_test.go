package webhook

import (
	"errors"
	"io"
	"testing"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jakedolan443/fernmail/internal/webhook/models"
	"github.com/zerodha/logf"
)

func TestWebhookMissingResourceContracts(t *testing.T) {
	db := testutil.NewDB(t, "webhook_contracts")
	lo := logf.New(logf.Opts{Writer: io.Discard})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	assertMissing := func(err error) {
		t.Helper()
		var e envelope.Error
		if !errors.As(err, &e) || e.ErrorType != envelope.NotFoundError {
			t.Fatalf("got %v, want not found", err)
		}
	}
	draft := models.Webhook{Name: "hook", URL: "https://example.com", Events: []string{"message.created"}}
	_, err = m.Update(999999, draft)
	assertMissing(err)
	draft.Secret = stringutil.PasswordDummy
	_, err = m.Update(999999, draft)
	assertMissing(err)
	_, err = m.Toggle(999999)
	assertMissing(err)
	for i := 0; i < 2; i++ {
		if err = m.Delete(999999); err != nil {
			t.Fatal(err)
		}
	}
	_, err = m.Get(999999)
	assertMissing(err)
}
