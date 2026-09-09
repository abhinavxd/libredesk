package migrations

import (
	"slices"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
)

func TestV2_9_0PrivateNotePermissionMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0")

	roles := []struct {
		name        string
		permissions pq.StringArray
		wantPrivate bool
	}{
		{"With old permission", pq.StringArray{"conversations:read", "messages:write"}, true},
		{"Without old permission", pq.StringArray{"conversations:read", "messages:read"}, false},
		{"Already migrated", pq.StringArray{"messages:write", "messages:write_private"}, true},
	}
	for _, role := range roles {
		if _, err := db.Exec(`INSERT INTO roles (name, description, permissions) VALUES ($1, '', $2)`, role.name, role.permissions); err != nil {
			t.Fatalf("inserting role %q: %v", role.name, err)
		}
	}

	// Running the migration twice verifies that it does not append duplicates.
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	for _, role := range roles {
		var got pq.StringArray
		if err := db.Get(&got, `SELECT permissions FROM roles WHERE name = $1`, role.name); err != nil {
			t.Fatalf("reading role %q: %v", role.name, err)
		}
		count := 0
		for _, permission := range got {
			if permission == "messages:write_private" {
				count++
			}
		}
		if slices.Contains(got, "messages:write_private") != role.wantPrivate {
			t.Errorf("role %q permissions = %v, want private permission = %v", role.name, got, role.wantPrivate)
		}
		if count > 1 {
			t.Errorf("role %q has duplicate private permissions: %v", role.name, got)
		}
	}
}

func TestV2_9_0ExternalSyncColumn(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0_external_sync")

	// schema.sql already ships the column, so drop it to stand in for a database created before it.
	db.MustExec(`ALTER TABLE users DROP COLUMN external_sync`)
	db.MustExec(`INSERT INTO users (type, first_name, email) VALUES ('contact', 'Ada', 'ada@example.com')`)

	// Running the migration twice verifies that adding the column is idempotent.
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
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
