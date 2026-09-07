package migrations

import (
	"github.com/abhinavxd/libredesk/internal/testutil"
	"testing"
)

func TestV2_11_0TwoFactor(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_11_0")
	db.MustExec(`DROP TABLE user_two_factor`)
	for range 2 {
		if err := V2_11_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	db.MustExec(`INSERT INTO users (type,email,first_name,last_name) VALUES ('agent','migration@example.com','Test','Agent')`)
	var enabled bool
	if err := db.Get(&enabled, `INSERT INTO user_two_factor (user_id,secret,setup_expires_at) SELECT id,'encrypted',NOW() FROM users LIMIT 1 RETURNING enabled`); err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("migration enables two-factor without verification")
	}
}
