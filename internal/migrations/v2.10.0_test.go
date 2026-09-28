package migrations

import (
	"github.com/abhinavxd/libredesk/internal/testutil"
	"testing"
)

func TestTelegramMigrationIsIdempotent(t *testing.T) {
	db := testutil.NewDB(t, "telegram_migration")
	for range 2 {
		if err := V2_10_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid WHERE t.typname='channels' AND e.enumlabel='telegram'`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("telegram enum entries=%d", count)
	}
	var id int
	if err := db.Get(&id, `INSERT INTO users (type,first_name,last_name) VALUES ('contact','Telegram','Contact') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contact_channel_identities (contact_id,channel,identifier) VALUES ($1,'telegram','4500000000000')`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contact_channel_identities (contact_id,channel,identifier) VALUES ($1,'telegram','4500000000000')`, id); err == nil {
		t.Fatal("duplicate telegram identity accepted")
	}
	if _, err := db.Exec(`INSERT INTO contact_channel_identities (contact_id,channel,identifier) VALUES ($1,'whatsapp','4500000000000')`, id); err != nil {
		t.Fatal("different channels must not collide:", err)
	}
	db.Close()
	if err := V2_10_0(db, nil, nil); err == nil {
		t.Fatal("database error ignored")
	}
}
