package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
)

func TestV2_10_0_RC1QueryIndexesMatchSchemaAfterRepeatedUpgrade(t *testing.T) {
	db := testutil.NewDB(t, "migration_query_indexes")
	names := []string{"index_conversations_on_last_message_at_desc", "index_conversation_messages_on_sender_id_and_created_at", "index_conversation_tags_on_tag_id_and_conversation_id"}
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
		if err := V2_10_0_RC1(db, nil /** fs **/, nil /** ko **/); err != nil {
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

func TestV2_10_0_RC1OIDCVerificationDefaultAndPreservedChoice(t *testing.T) {
	db := testutil.NewDB(t, "migration_oidc_verification")
	if _, err := db.Exec(`
		ALTER TABLE oidc DROP COLUMN verify_email;
		INSERT INTO oidc (provider_url, client_id, client_secret)
		VALUES ('https://oidc.test', 'migration-test', 'secret');
	`); err != nil {
		t.Fatal(err)
	}
	for attempt := range 2 {
		if err := V2_10_0_RC1(db, nil /** fs **/, nil /** ko **/); err != nil {
			t.Fatal(err)
		}
		var verifyEmail bool
		if err := db.Get(&verifyEmail, `SELECT verify_email FROM oidc WHERE client_id = 'migration-test'`); err != nil {
			t.Fatal(err)
		}
		if expected := attempt == 0; verifyEmail != expected {
			t.Fatalf("upgrade %d: verify_email = %t, want %t", attempt+1, verifyEmail, expected)
		}
		if _, err := db.Exec(`UPDATE oidc SET verify_email = false WHERE client_id = 'migration-test'`); err != nil {
			t.Fatal(err)
		}
	}
}
