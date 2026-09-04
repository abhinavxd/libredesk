package notifier

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func newTestPushStore(t *testing.T) (*sqlPushStore, int, int) {
	t.Helper()
	db := testutil.NewDB(t, "notification_push_store")
	var q pushQueries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, queriesFS); err != nil {
		t.Fatal(err)
	}
	var firstUserID, secondUserID int
	if err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('agent', 'push-one@example.com', 'One') RETURNING id`).Scan(&firstUserID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('agent', 'push-two@example.com', 'Two') RETURNING id`).Scan(&secondUserID); err != nil {
		t.Fatal(err)
	}
	return &sqlPushStore{q: q}, firstUserID, secondUserID
}

func TestPushStoreMovesEndpointToLatestUser(t *testing.T) {
	store, firstUserID, secondUserID := newTestPushStore(t)
	endpoint := "https://push.example/shared"
	if err := store.Upsert(firstUserID, PushSubscriptionInput{Endpoint: endpoint, P256DH: "first-key", Auth: "first-auth"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(secondUserID, PushSubscriptionInput{Endpoint: endpoint, P256DH: "second-key", Auth: "second-auth"}); err != nil {
		t.Fatal(err)
	}

	firstSubscriptions, err := store.List(firstUserID)
	if err != nil {
		t.Fatal(err)
	}
	secondSubscriptions, err := store.List(secondUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstSubscriptions) != 0 {
		t.Fatalf("first user retained %d subscriptions", len(firstSubscriptions))
	}
	if len(secondSubscriptions) != 1 || secondSubscriptions[0].P256DH != "second-key" || secondSubscriptions[0].Auth != "second-auth" {
		t.Fatalf("second user subscriptions = %#v", secondSubscriptions)
	}
}

func TestPushStoreDeleteIsUserScoped(t *testing.T) {
	store, firstUserID, secondUserID := newTestPushStore(t)
	endpoint := "https://push.example/scoped"
	if err := store.Upsert(secondUserID, PushSubscriptionInput{Endpoint: endpoint, P256DH: "key", Auth: "auth"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(firstUserID, endpoint); err != nil {
		t.Fatal(err)
	}
	subscriptions, err := store.List(secondUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("other user delete left %d subscriptions, want 1", len(subscriptions))
	}
	if err := store.Delete(secondUserID, endpoint); err != nil {
		t.Fatal(err)
	}
	subscriptions, err = store.List(secondUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 0 {
		t.Fatalf("owner delete left %d subscriptions", len(subscriptions))
	}
}
