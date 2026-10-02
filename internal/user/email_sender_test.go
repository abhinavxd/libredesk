package user

import (
	"github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/volatiletech/null/v9"
	"sync"
	"testing"
)

func TestEmailSenderIdentityIsReusedWithoutCRMUpdates(t *testing.T) {
	manager, db := newTestManager(t)
	first := models.User{Email: null.StringFrom(" Sender@Example.com "), FirstName: "Sender"}
	if err := manager.ResolveEmailSender(&first); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			next := models.User{Email: null.StringFrom("sender@example.com"), FirstName: "New name", ExternalUserID: null.StringFrom("ignored"), CustomAttributes: []byte(`{"plan":"ignored"}`)}
			if err := manager.ResolveEmailSender(&next); err != nil {
				t.Error(err)
				return
			}
			if next.ID != first.ID {
				t.Errorf("sender identity changed: %d != %d", next.ID, first.ID)
			}
		})
	}
	wg.Wait()
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM users WHERE email = 'sender@example.com' AND first_name = 'Sender' AND external_user_id IS NULL AND custom_attributes = '{}'::jsonb`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one minimal immutable sender identity, got %d", count)
	}
}
