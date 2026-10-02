package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestContributorUpgradeTurnsOpenAddressesIntoExplicitGrants(t *testing.T) {
	db := testutil.NewDB(t, "contributor_upgrade")
	restoreOpenAddressFlag(t, db)
	// Recreate the v3.6 shape: no Contributor role, no review permissions or tables.
	db.MustExec(`
		DROP TABLE outbound_review_media, outbound_reviews;
		DELETE FROM roles WHERE name='Contributor';
		UPDATE roles SET permissions = array_remove(array_remove(permissions, 'reviews:manage'), 'conversations:create');
		INSERT INTO users(type,email,first_name,enabled) VALUES
			('agent','agent@example.test','Agent',true),
			('agent','away@example.test','Away',false),
			('agent','admin@example.test','Admin',true);
		INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u JOIN roles r ON r.name='Agent' WHERE u.email IN ('agent@example.test','away@example.test');
		INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u JOIN roles r ON r.name='Admin' WHERE u.email='admin@example.test';
		INSERT INTO inboxes(name,channel,"from") VALUES('Mail','email','open@example.test');
		INSERT INTO email_addresses(inbox_id,address,kind,restricted) SELECT id,'open@example.test','mailbox',false FROM inboxes;
		INSERT INTO email_addresses(inbox_id,address,kind,restricted) SELECT id,'closed@example.test','alias',true FROM inboxes;
	`)
	for range 2 {
		if err := V3_7_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	access := func(email, address string) bool {
		t.Helper()
		var allowed bool
		if err := db.Get(&allowed, `SELECT can_access_email_address(a.id,u.id) FROM email_addresses a CROSS JOIN users u WHERE a.address=$1 AND u.email=$2`, address, email); err != nil {
			t.Fatal(err)
		}
		return allowed
	}
	if !access("agent@example.test", "open@example.test") {
		t.Fatal("agent lost access to a previously open address")
	}
	if access("agent@example.test", "closed@example.test") {
		t.Fatal("agent gained access to a restricted address")
	}
	if !access("admin@example.test", "open@example.test") || !access("admin@example.test", "closed@example.test") {
		t.Fatal("admin must see every address")
	}

	var ok bool
	for _, query := range []string{
		`SELECT count(*) = 1 FROM email_address_users`,
		`SELECT NOT EXISTS(SELECT 1 FROM email_address_users eau JOIN users u ON u.id=eau.user_id WHERE u.email IN ('away@example.test','admin@example.test'))`,
		`SELECT NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='email_addresses' AND column_name='restricted')`,
		`SELECT permissions @> '{reviews:submit,conversations:create,messages:read}' AND NOT 'messages:write' = ANY(permissions) FROM roles WHERE name='Contributor'`,
		`SELECT bool_and(cardinality(array_positions(permissions, 'reviews:manage')) = 1 AND cardinality(array_positions(permissions, 'conversations:create')) = 1) FROM roles WHERE name IN ('Admin','Agent')`,
		`SELECT to_regclass('outbound_reviews') IS NOT NULL AND to_regclass('outbound_review_media') IS NOT NULL`,
		`SELECT can_access_inbox(i.id,u.id) FROM inboxes i CROSS JOIN users u WHERE u.email='agent@example.test'`,
	} {
		if err := db.Get(&ok, query); err != nil || !ok {
			t.Fatalf("%s: %v %v", query, ok, err)
		}
	}

	// Nothing is open any more: a user added after the upgrade sees nothing until granted.
	db.MustExec(`INSERT INTO users(type,email,first_name) VALUES('agent','new@example.test','New')`)
	if access("new@example.test", "open@example.test") {
		t.Fatal("new user can read an address without a grant")
	}
	if err := db.Get(&ok, `SELECT can_access_inbox(i.id,u.id) FROM inboxes i CROSS JOIN users u WHERE u.email='new@example.test'`); err != nil || ok {
		t.Fatalf("new user can read a transport without a grant: %v %v", ok, err)
	}
}
