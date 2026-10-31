package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
)

func TestV2_9_3QueryIndexesMatchSchemaAfterRepeatedUpgrade(t *testing.T) {
	db := testutil.NewDB(t, "migration_query_indexes")
	names := []string{"index_conversations_on_last_message_at_desc", "index_conversation_messages_on_sender_id_and_created_at"}
	expected := map[string]string{}
	for _, name := range names {
		var definition string
		if err := db.Get(&definition, `SELECT pg_get_indexdef(indexrelid) FROM pg_index WHERE indexrelid = to_regclass($1) AND indisvalid AND indisready`, name); err != nil {
			t.Fatal(err)
		}
		expected[name] = definition
		if _, err := db.Exec(`DROP INDEX ` + pq.QuoteIdentifier(name)); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := V2_9_3(db, nil /** fs **/, nil /** ko **/); err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			var actual string
			if err := db.Get(&actual, `SELECT pg_get_indexdef(indexrelid) FROM pg_index WHERE indexrelid = to_regclass($1) AND indisvalid AND indisready`, name); err != nil {
				t.Fatal(err)
			}
			if actual != expected[name] {
				t.Fatalf("index %s = %s, want %s", name, actual, expected[name])
			}
		}
	}
}
