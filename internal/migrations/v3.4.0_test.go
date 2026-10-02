package migrations

import (
	"github.com/jakedolan443/fernmail/internal/testutil"
	"testing"
)

func TestInboxAccessUpgradePreservesExistingMailboxesAndGrants(t *testing.T) {
	db := testutil.NewDB(t, "inbox_access_upgrade")
	db.MustExec(`DROP FUNCTION can_access_inbox(integer,bigint); DROP TABLE inbox_access,inbox_users,inbox_roles;
 INSERT INTO inboxes(name,channel) VALUES('Existing inbox','email');
 INSERT INTO users(type,email,first_name) VALUES('agent','reader@example.test','Reader');`)
	if err := V3_4_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	var allowed bool
	if err := db.Get(&allowed, `SELECT can_access_inbox((SELECT id FROM inboxes LIMIT 1),(SELECT id FROM users WHERE email='reader@example.test'))`); err != nil || !allowed {
		t.Fatalf("legacy inbox restricted: %v, %v", allowed, err)
	}
	db.MustExec(`INSERT INTO inbox_access SELECT id,true FROM inboxes;
 INSERT INTO inbox_users SELECT i.id,u.id FROM inboxes i CROSS JOIN users u WHERE u.email='reader@example.test';`)
	if err := V3_4_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	var preserved bool
	if err := db.Get(&preserved, `SELECT restricted AND EXISTS(SELECT 1 FROM inbox_users) FROM inbox_access LIMIT 1`); err != nil || !preserved {
		t.Fatalf("grant lost on retry: %v, %v", preserved, err)
	}
}
