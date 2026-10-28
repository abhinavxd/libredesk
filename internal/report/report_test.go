package report

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestOverviewDatesUseWorkspaceTimezone(t *testing.T) {
	db := testutil.NewDB(t, "report_timezone")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var contactID, inboxID int
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, last_name) VALUES ('contact', 'Report', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Report', 'livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	instants := []time.Time{base.Add(30 * time.Minute), base.Add(23*time.Hour + 30*time.Minute)}
	for _, instant := range instants {
		if _, err := db.Exec(`INSERT INTO conversations (contact_id, inbox_id, status_id, created_at, resolved_at)
			VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3, $3)`, contactID, inboxID, instant); err != nil {
			t.Fatal(err)
		}
	}
	for _, timezone := range []string{"UTC", "Asia/Kathmandu", "America/Los_Angeles", "", "invalid/zone", "Local"} {
		t.Run(timezone, func(t *testing.T) {
			loc := time.UTC
			if timezone == "Asia/Kathmandu" || timezone == "America/Los_Angeles" {
				loc, err = time.LoadLocation(timezone)
				if err != nil {
					t.Fatal(err)
				}
			}
			want := map[string]int{}
			for _, instant := range instants {
				want[instant.In(loc).Format(time.DateOnly)]++
			}
			raw, err := m.GetOverviewChart(7 /** days **/, timezone)
			if err != nil {
				t.Fatal(err)
			}
			var charts map[string][]struct {
				Date  string `json:"date"`
				Count int    `json:"count"`
			}
			if err := json.Unmarshal(raw, &charts); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"new_conversations", "resolved_conversations"} {
				if len(charts[name]) != len(want) {
					t.Fatalf("%s dates = %v, want %v", name, charts[name], want)
				}
				for _, row := range charts[name] {
					if row.Count != want[row.Date] {
						t.Fatalf("%s row = %+v, want %v", name, row, want)
					}
				}
			}
		})
	}

	if _, err := db.Exec(`DELETE FROM conversations`); err != nil {
		t.Fatal(err)
	}
	timezone := "Asia/Kathmandu"
	var midnight time.Time
	if err := db.Get(&midnight, `SELECT DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1`, timezone); err != nil {
		t.Fatal(err)
	}
	for _, instant := range []time.Time{midnight.Add(-time.Minute), midnight.Add(time.Minute)} {
		var conversationID int
		if err := db.Get(&conversationID, `INSERT INTO conversations (contact_id, inbox_id, status_id, created_at)
			VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3) RETURNING id`, contactID, inboxID, instant); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO csat_responses (conversation_id, rating, created_at) VALUES ($1, 5, $2)`, conversationID, instant); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversation_messages (conversation_id, type, status, sender_id, sender_type, created_at)
			VALUES ($1, 'incoming', 'sent', $2, 'contact', $3)`, conversationID, contactID, instant); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name string
		get  func(int, string) (json.RawMessage, error)
		key  string
		want float64
	}{
		{"csat", m.GetOverviewCSAT, "total_responses", 1},
		{"messages", m.GetOverviewMessageVolume, "total_messages", 1},
		{"tags", m.GetOverviewTagDistribution, "untagged_conversations", 1},
		{"sla", m.GetOverviewSLA, "first_response_met_count", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tc.get(0 /** days **/, timezone)
			if err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			if result[tc.key] != tc.want {
				t.Fatalf("%s = %v, want %v", tc.key, result[tc.key], tc.want)
			}
		})
	}
}
