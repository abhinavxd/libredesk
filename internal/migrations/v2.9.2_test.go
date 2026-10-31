package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
)

func TestV2_9_2DeleteConversationPermissionMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_2_conversations_delete")

	// schema.sql seeds Admin with the permission, which is the fresh-install case. Take it away
	// again to get the upgrade case the migration exists for: a desk installed before this release.
	if _, err := db.Exec(`UPDATE roles SET permissions = array_remove(permissions, 'conversations:delete') WHERE name = 'Admin'`); err != nil {
		t.Fatalf("removing the seeded permission: %v", err)
	}
	var seeded pq.StringArray
	if err := db.Get(&seeded, `SELECT permissions FROM roles WHERE name = 'Admin'`); err != nil {
		t.Fatalf("reading Admin: %v", err)
	}
	if slices.Contains(seeded, "conversations:delete") {
		t.Fatalf("Admin still has the permission before the migration: %v", seeded)
	}

	roles := []struct {
		name        string
		permissions pq.StringArray
	}{
		{"Supervisor", pq.StringArray{"conversations:read", "conversations:write"}},
		{"Already granted", pq.StringArray{"conversations:read", "conversations:delete"}},
	}
	for _, role := range roles {
		if _, err := db.Exec(`INSERT INTO roles (name, description, permissions) VALUES ($1, '', $2)`, role.name, role.permissions); err != nil {
			t.Fatalf("inserting role %q: %v", role.name, err)
		}
	}

	// Running the migration twice verifies that it does not append duplicates.
	for range 2 {
		if err := V2_9_2(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	for _, tc := range []struct {
		role      string
		wantCount int
	}{
		{"Admin", 1},
		{"Agent", 0},
		{"Supervisor", 0},
		{"Already granted", 1},
	} {
		var got pq.StringArray
		if err := db.Get(&got, `SELECT permissions FROM roles WHERE name = $1`, tc.role); err != nil {
			t.Fatalf("reading role %q: %v", tc.role, err)
		}
		count := 0
		for _, permission := range got {
			if permission == "conversations:delete" {
				count++
			}
		}
		if count != tc.wantCount {
			t.Errorf("role %q has %d %q permissions, want %d: %v", tc.role, count, "conversations:delete", tc.wantCount, got)
		}
	}
}
