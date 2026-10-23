package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0ResponseIntervalMigration(t *testing.T) {
	db := testutil.NewDB(t, "migration_v2_9_0_response_interval")
	query := `SELECT data_type || ':' || is_nullable || ':' || COALESCE(column_default, '') FROM information_schema.columns WHERE table_name = 'ai_assistants' AND column_name = 'response_interval'`
	var expected string
	if err := db.Get(&expected, query); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`ALTER TABLE ai_assistants DROP COLUMN response_interval`)
	var userID, assistantID int
	if err := db.Get(&userID, `INSERT INTO users (type, first_name, last_name, enabled) VALUES ('ai_assistant', 'Support', '', true) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&assistantID, `INSERT INTO ai_assistants (user_id) VALUES ($1) RETURNING id`, userID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := V2_9_0_RC11(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var actual string
	if err := db.Get(&actual, query); err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("migration column = %q, schema column = %q", actual, expected)
	}
	var interval int
	if err := db.Get(&interval, `SELECT response_interval FROM ai_assistants WHERE id = $1`, assistantID); err != nil {
		t.Fatal(err)
	}
	if interval != 0 {
		t.Fatalf("existing assistant interval = %d, want 0", interval)
	}
	for _, value := range []int{-1, 0, 5, 300, 301} {
		_, err := db.Exec(`UPDATE ai_assistants SET response_interval = $1 WHERE id = $2`, value, assistantID)
		wantError := value < 0 || value > 300
		if (err != nil) != wantError {
			t.Fatalf("interval %d: got error %v, want error %v", value, err, wantError)
		}
	}
}
