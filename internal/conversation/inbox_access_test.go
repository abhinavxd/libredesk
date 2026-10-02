package conversation

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/address"
	"github.com/jakedolan443/fernmail/internal/authz"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/inbox"
	"github.com/jakedolan443/fernmail/internal/search"
	smodels "github.com/jakedolan443/fernmail/internal/search/models"
	"github.com/jakedolan443/fernmail/internal/testutil"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// Every read surface must use the same address policy. This covers list,
// unread badges, drafts, live-subscription filtering, search, and direct URL
// access rather than trusting a sidebar-only visibility check.
func TestAddressAccessAcrossMailSurfaces(t *testing.T) {
	db := testutil.NewDB(t, "address_access_surfaces")
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	inboxes, err := inbox.New(&lo, db, i18n, "")
	if err != nil {
		t.Fatal(err)
	}
	addresses, err := address.New(db)
	if err != nil {
		t.Fatal(err)
	}

	var viewer, contact, billingInbox, supportInbox int
	for _, row := range []struct {
		dest *int
		q    string
	}{
		{&viewer, `INSERT INTO users(type,email,first_name) VALUES('agent','reader@example.test','Reader') RETURNING id`},
		{&contact, `INSERT INTO users(type,email,first_name) VALUES('contact','sender@example.test','Sender') RETURNING id`},
		{&billingInbox, `INSERT INTO inboxes(name,channel) VALUES('Billing transport','email') RETURNING id`},
		{&supportInbox, `INSERT INTO inboxes(name,channel) VALUES('Support transport','email') RETURNING id`},
	} {
		if err := db.Get(row.dest, row.q); err != nil {
			t.Fatal(err)
		}
	}
	billing, err := addresses.EnsureMailboxAddress(billingInbox, "billing@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := addresses.Update(billing.ID, address.Address{
		InboxID: billingInbox, Address: billing.Address, Kind: address.KindMailbox,
		Enabled: true, UserIDs: []int{viewer},
	}); err != nil {
		t.Fatal(err)
	}
	support, err := addresses.EnsureMailboxAddress(supportInbox, "support@example.test")
	if err != nil {
		t.Fatal(err)
	}

	var billingConversation, supportConversation int
	var billingUUID, supportUUID string
	if err := db.QueryRowx(`INSERT INTO conversations(contact_id,inbox_id,address_id,status_id,subject,last_message_at)
		VALUES($1,$2,$3,(SELECT id FROM conversation_statuses WHERE name='Open'),'Billing',NOW()) RETURNING id,uuid::text`, contact, billingInbox, billing.ID).Scan(&billingConversation, &billingUUID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowx(`INSERT INTO conversations(contact_id,inbox_id,address_id,status_id,subject,last_message_at)
		VALUES($1,$2,$3,(SELECT id FROM conversation_statuses WHERE name='Open'),'Support',NOW()) RETURNING id,uuid::text`, contact, supportInbox, support.ID).Scan(&supportConversation, &supportUUID); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`
		INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,text_content) VALUES
			($1,$2,'contact','incoming','received','billing private mailbox example'),
			($3,$2,'contact','incoming','received','support private mailbox example')
	`, billingConversation, contact, supportConversation)
	db.MustExec(`
		INSERT INTO conversation_drafts(conversation_id,user_id,type,content) VALUES
			($1,$3,'reply','Billing draft'), ($2,$3,'reply','Support draft')
	`, billingConversation, supportConversation, viewer)

	manager := newTestManager()
	manager.db, manager.lo, manager.i18n = db, &lo, i18n
	if err := dbutil.ScanSQLFile("queries.sql", &manager.q, db, efs); err != nil {
		t.Fatal(err)
	}
	searcher, err := search.New(search.Opts{DB: db, Lo: &lo, I18n: i18n, FilterLocation: func() string { return "UTC" }})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := authz.NewEnforcer(&lo, i18n, inboxes.CanAccess, addresses.CanAccess)
	if err != nil {
		t.Fatal(err)
	}
	reader := umodels.User{ID: viewer, Enabled: true, Permissions: []string{"conversations:read", "conversations:read_all"}}
	scope := smodels.ReadScope{UserID: viewer, Read: true, ReadAll: true}

	check := func(want int) {
		t.Helper()
		rows, err := manager.GetConversations(viewer, viewer, nil, []string{models.AllConversations}, "DESC", "", "[]", 1, 20)
		if err != nil || len(rows) != want {
			t.Fatalf("address list rows=%d err=%v want=%d", len(rows), err, want)
		}
		counts, err := manager.GetSidebarCounts(viewer, reader.Permissions, nil)
		if err != nil || counts.Unread != want || len(counts.Addresses) != want {
			t.Fatalf("address badges=%+v err=%v want=%d", counts, err, want)
		}
		drafts, err := manager.GetAllUserDrafts(viewer)
		if err != nil || len(drafts) != want {
			t.Fatalf("drafts=%d err=%v want=%d", len(drafts), err, want)
		}
		var authorized []string
		err = manager.q.FilterAuthorizedListUUIDs.Select(&authorized, pq.Array([]string{billingUUID, supportUUID}), viewer, pq.Array([]int{}), true, true, false, false, false, false)
		if err != nil || len(authorized) != want {
			t.Fatalf("live subscriptions=%v err=%v want=%d", authorized, err, want)
		}
		conversations, _, _, err := searcher.Conversations(smodels.Query{Term: "sender@example.test", PageSize: 20}, scope)
		if err != nil || len(conversations) != want {
			t.Fatalf("search conversations=%d err=%v want=%d", len(conversations), err, want)
		}
		messages, _, _, err := searcher.Messages(smodels.Query{Term: "private mailbox example", PageSize: 20}, scope)
		if err != nil || len(messages) != want {
			t.Fatalf("search messages=%d err=%v want=%d", len(messages), err, want)
		}
	}

	check(1)
	allowed, err := enforcer.EnforceConversationAccess(reader, models.Conversation{InboxID: supportInbox, AddressID: null.IntFrom(support.ID)})
	if err != nil || allowed {
		t.Fatalf("restricted direct support access=%t err=%v", allowed, err)
	}

	if _, err := addresses.Update(support.ID, address.Address{
		InboxID: supportInbox, Address: support.Address, Kind: address.KindMailbox,
		Enabled: true, UserIDs: []int{viewer},
	}); err != nil {
		t.Fatal(err)
	}
	check(2)
	allowed, err = enforcer.EnforceConversationAccess(reader, models.Conversation{InboxID: supportInbox, AddressID: null.IntFrom(support.ID)})
	if err != nil || !allowed {
		t.Fatalf("direct support grant=%t err=%v", allowed, err)
	}

	if _, err := addresses.Update(billing.ID, address.Address{
		InboxID: billingInbox, Address: billing.Address, Kind: address.KindMailbox,
		Enabled: true, UserIDs: []int{},
	}); err != nil {
		t.Fatal(err)
	}
	check(1)
}
