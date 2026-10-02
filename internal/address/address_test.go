package address

import (
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestAddressLifecyclePreservesHistoricalRouting(t *testing.T) {
	db := testutil.NewDB(t, "address_lifecycle")
	manager, err := New(db)
	if err != nil {
		t.Fatal(err)
	}

	var inboxID, contactID, agentID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes(name, channel, "from") VALUES ('Mail transport', 'email', 'director@example.test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users(type, email, first_name) VALUES ('contact', 'customer@example.test', 'Customer') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&agentID, `INSERT INTO users(type, email, first_name) VALUES ('agent', 'agent@example.test', 'Agent') RETURNING id`); err != nil {
		t.Fatal(err)
	}

	primary, err := manager.EnsureMailboxAddress(inboxID, "director@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(Address{InboxID: inboxID, Address: "duplicate@example.test", Kind: KindMailbox, Enabled: true}); err == nil {
		t.Fatal("expected direct primary-address creation to be rejected")
	}

	alias, err := manager.Create(Address{InboxID: inboxID, Address: "support@example.test", Kind: KindAlias, Enabled: true, UserIDs: []int{agentID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Get(new(int), `INSERT INTO conversations(contact_id, inbox_id, address_id, status_id)
		VALUES ($1, $2, $3, (SELECT id FROM conversation_statuses WHERE name='Open')) RETURNING id`, contactID, inboxID, alias.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Update(alias.ID, Address{InboxID: inboxID, Address: "help@example.test", Kind: KindAlias, Enabled: true}); err == nil || !strings.Contains(err.Error(), "cannot be moved or renamed") {
		t.Fatalf("renaming historical alias error = %v", err)
	}
	if _, err := manager.Delete(alias.ID); err == nil || !strings.Contains(err.Error(), "must be disabled") {
		t.Fatalf("deleting historical alias error = %v", err)
	}
	if _, err := manager.Update(alias.ID, Address{InboxID: inboxID, Address: alias.Address, Kind: KindAlias, Enabled: false, UserIDs: []int{agentID}}); err != nil {
		t.Fatalf("disabling historical alias: %v", err)
	}
	// Address grants are canonical. A stale transport restriction must not
	// silently undo the explicit policy configured on this endpoint.
	db.MustExec(`INSERT INTO inbox_access(inbox_id, restricted) VALUES ($1, TRUE)`, inboxID)
	if allowed, err := manager.CanAccess(agentID, alias.ID); err != nil || !allowed {
		t.Fatalf("disabled historical alias with an explicit grant should stay readable: allowed=%v err=%v", allowed, err)
	}
	if _, err := manager.Update(primary.ID, Address{InboxID: inboxID, Address: "renamed@example.test", Kind: KindMailbox, Enabled: true}); err == nil || !strings.Contains(err.Error(), "owned by its transport") {
		t.Fatalf("renaming primary error = %v", err)
	}
}

func TestAccessibleAddressesListMailboxesBeforeAliases(t *testing.T) {
	db := testutil.NewDB(t, "address_ordering")
	manager, err := New(db)
	if err != nil {
		t.Fatal(err)
	}

	var firstInboxID, secondInboxID, agentID int
	if err := db.Get(&firstInboxID, `INSERT INTO inboxes(name, channel, "from") VALUES ('Director transport', 'email', 'director@example.test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&secondInboxID, `INSERT INTO inboxes(name, channel, "from") VALUES ('Support transport', 'email', 'support@example.test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&agentID, `INSERT INTO users(type, email, first_name) VALUES ('agent', 'agent@example.test', 'Agent') RETURNING id`); err != nil {
		t.Fatal(err)
	}

	director, err := manager.EnsureMailboxAddress(firstInboxID, "director@example.test")
	if err != nil {
		t.Fatal(err)
	}
	support, err := manager.EnsureMailboxAddress(secondInboxID, "support@example.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []Address{
		{ID: director.ID, InboxID: firstInboxID, Address: director.Address, Kind: KindMailbox, Enabled: true, UserIDs: []int{agentID}},
		{ID: support.ID, InboxID: secondInboxID, Address: support.Address, Kind: KindMailbox, Enabled: true, UserIDs: []int{agentID}},
	} {
		if _, err := manager.Update(address.ID, address); err != nil {
			t.Fatal(err)
		}
	}
	for _, address := range []Address{
		{InboxID: firstInboxID, Address: "billing@example.test", Kind: KindAlias, Enabled: true, UserIDs: []int{agentID}},
		{InboxID: secondInboxID, Address: "contact@example.test", Kind: KindAlias, Enabled: true, UserIDs: []int{agentID}},
	} {
		if _, err := manager.Create(address); err != nil {
			t.Fatal(err)
		}
	}

	addresses, err := manager.GetAccessible(agentID)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(addresses))
	for i, address := range addresses {
		got[i] = address.Address
	}
	want := []string{
		"director@example.test",
		"support@example.test",
		"billing@example.test",
		"contact@example.test",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("accessible address order = %v, want %v", got, want)
	}
}
