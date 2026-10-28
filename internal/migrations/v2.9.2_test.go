package migrations

import (
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_2ExternalSyncColumn(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_2_external_sync")

	// schema.sql already ships the column, so drop it to stand in for a database created before it.
	db.MustExec(`ALTER TABLE users DROP COLUMN external_sync`)
	db.MustExec(`INSERT INTO users (type, first_name, email) VALUES ('contact', 'Ada', 'ada@example.com')`)
	db.MustExec(`INSERT INTO users (type, first_name, last_name, email, external_user_id, phone_number) VALUES ('contact', 'Grace', '', 'grace@example.com', 'ext-grace', '5550100')`)

	// Running the migration twice verifies that adding the column is idempotent.
	for range 2 {
		if err := V2_9_2(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	// A contact an integration created is recorded as holding what the integration supplied, since
	// that is what the sync wrote before the record existed; empty fields are left out.
	var seeded string
	if err := db.Get(&seeded, `SELECT external_sync::text FROM users WHERE external_user_id = 'ext-grace'`); err != nil {
		t.Fatalf("reading seeded external_sync: %v", err)
	}
	if want := `{"email": "grace@example.com", "first_name": "Grace", "phone_number": "5550100"}`; seeded != want {
		t.Errorf("external contact external_sync = %s, want %s", seeded, want)
	}

	var column struct {
		DataType   string `db:"data_type"`
		IsNullable string `db:"is_nullable"`
		Default    string `db:"column_default"`
	}
	if err := db.Get(&column, `
		SELECT data_type, is_nullable, COALESCE(column_default, '') AS column_default
		FROM information_schema.columns
		WHERE table_name = 'users' AND column_name = 'external_sync'`); err != nil {
		t.Fatalf("reading external_sync column: %v", err)
	}
	if column.DataType != "jsonb" {
		t.Errorf("external_sync data type = %q, want jsonb", column.DataType)
	}
	if column.IsNullable != "NO" {
		t.Errorf("external_sync is nullable = %q, want NO", column.IsNullable)
	}
	if !strings.Contains(column.Default, "{}") {
		t.Errorf("external_sync default = %q, want an empty JSON object", column.Default)
	}

	var value string
	if err := db.Get(&value, `SELECT external_sync::text FROM users WHERE email = 'ada@example.com'`); err != nil {
		t.Fatalf("reading external_sync value: %v", err)
	}
	if value != "{}" {
		t.Errorf("existing contact external_sync = %q, want {}", value)
	}
}
