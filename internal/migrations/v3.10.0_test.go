package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestKeyDistributionMigrationIsIdempotent(t *testing.T) {
	db := testutil.NewDB(t, "key_distribution")
	// Recreate the v3.9 shape: no key tables, setting or permission.
	db.MustExec(`
		DROP TABLE activation_key_preferences, activation_keys, activation_key_apps;
		DROP FUNCTION guard_activation_key_status();
		DELETE FROM settings WHERE key = 'key_distribution';
		UPDATE roles SET permissions = array_remove(permissions, 'activation_keys:manage');
	`)
	for range 2 {
		if err := V3_10_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var ok bool
	for _, query := range []string{
		`SELECT to_regclass('activation_keys') IS NOT NULL AND to_regclass('activation_key_apps') IS NOT NULL AND to_regclass('activation_key_preferences') IS NOT NULL`,
		`SELECT value = '{"enabled":false,"max_keys_per_email":1,"low_stock_threshold":10}'::jsonb FROM settings WHERE key = 'key_distribution'`,
		`SELECT array_length(array_positions(permissions, 'activation_keys:manage'), 1) = 1 FROM roles WHERE name = 'Admin'`,
		`SELECT NOT EXISTS (SELECT 1 FROM roles WHERE name <> 'Admin' AND 'activation_keys:manage' = ANY(permissions))`,
		`SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'guard_activation_key_status')`,
	} {
		if err := db.Get(&ok, query); err != nil || !ok {
			t.Fatalf("%s: %v %v", query, ok, err)
		}
	}
}
