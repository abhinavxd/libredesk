package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
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

func TestV2_9_0CustomAttributeReadOnlyColumn(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0_read_only")

	// Simulate an installation created before the column existed.
	if _, err := db.Exec(`ALTER TABLE custom_attribute_definitions DROP COLUMN read_only`); err != nil {
		t.Fatalf("dropping column: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO custom_attribute_definitions (name, description, applies_to, key, data_type)
		VALUES ('Account tier', 'Tier of the customer account', 'contact', 'account_tier', 'text')
	`); err != nil {
		t.Fatalf("inserting definition: %v", err)
	}

	// Running the migration twice verifies that adding the column is idempotent.
	for range 2 {
		if err := V2_9_0(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	var column struct {
		IsNullable    string      `db:"is_nullable"`
		ColumnDefault null.String `db:"column_default"`
	}
	if err := db.Get(&column, `
		SELECT is_nullable, column_default FROM information_schema.columns
		WHERE table_name = 'custom_attribute_definitions' AND column_name = 'read_only'
	`); err != nil {
		t.Fatalf("reading column metadata: %v", err)
	}
	if column.IsNullable != "NO" {
		t.Errorf("read_only is_nullable = %q, want %q", column.IsNullable, "NO")
	}
	if column.ColumnDefault.String != "false" {
		t.Errorf("read_only column_default = %q, want %q", column.ColumnDefault.String, "false")
	}

	var readOnly bool
	if err := db.Get(&readOnly, `SELECT read_only FROM custom_attribute_definitions WHERE key = 'account_tier'`); err != nil {
		t.Fatalf("reading read_only: %v", err)
	}
	if readOnly {
		t.Error("existing definition backfilled with read_only = true, want false")
	}
}
