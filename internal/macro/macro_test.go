package macro

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/macro/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "macro")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating macro manager: %v", err)
	}
	return mgr, db
}

func TestGetAllCompact(t *testing.T) {
	mgr, _ := newTestManager(t)

	withContent := mustCreate(t, mgr, "with content", "<p>hello</p>", "all", nil, nil, []string{"replying"})
	withoutContent, err := mgr.Create("without content", "", nil, nil, "all", []string{"replying"}, json.RawMessage(`[{"type":"add_tags","value":["x"]}]`))
	if err != nil {
		t.Fatalf("creating macro: %v", err)
	}

	full, err := mgr.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(full) != 2 {
		t.Fatalf("GetAll returned %d macros, want 2", len(full))
	}

	compact, err := mgr.GetAllCompact("", 1, 30)
	if err != nil {
		t.Fatalf("GetAllCompact: %v", err)
	}
	if len(compact) != 2 {
		t.Fatalf("GetAllCompact returned %d macros, want 2", len(compact))
	}
	for _, m := range compact {
		if m.Total != 2 {
			t.Errorf("macro %d has total %d, want 2", m.ID, m.Total)
		}
		switch m.ID {
		case withContent.ID:
			if !m.HasMessageContent {
				t.Error("macro with content has HasMessageContent=false")
			}
		case withoutContent.ID:
			if m.HasMessageContent {
				t.Error("macro without content has HasMessageContent=true")
			}
			if len(m.Actions) == 0 {
				t.Error("compact row lost actions")
			}
		default:
			t.Errorf("unexpected macro id %d", m.ID)
		}
	}

	filtered, err := mgr.GetAllCompact("without", 1, 30)
	if err != nil {
		t.Fatalf("GetAllCompact with query: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != withoutContent.ID {
		t.Fatalf("name filter returned %d macros, want the one named 'without content'", len(filtered))
	}

	paged, err := mgr.GetAllCompact("", 2, 1)
	if err != nil {
		t.Fatalf("GetAllCompact page 2: %v", err)
	}
	if len(paged) != 1 {
		t.Fatalf("page 2 of size 1 returned %d macros, want 1", len(paged))
	}
	if paged[0].Total != 2 {
		t.Fatalf("page 2 of size 1 has total %d, want 2", paged[0].Total)
	}

	got, err := mgr.Get(withContent.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.MessageContent != "<p>hello</p>" {
		t.Errorf("Get returned message content %q", got.MessageContent)
	}
}

func TestSearchCompact(t *testing.T) {
	mgr, db := newTestManager(t)

	var agentID, otherID, teamID, otherTeamID int
	if err := db.Get(&agentID, `INSERT INTO users (type, email, first_name) VALUES ('agent', 'a@example.com', 'A') RETURNING id`); err != nil {
		t.Fatalf("inserting agent: %v", err)
	}
	if err := db.Get(&otherID, `INSERT INTO users (type, email, first_name) VALUES ('agent', 'b@example.com', 'B') RETURNING id`); err != nil {
		t.Fatalf("inserting agent: %v", err)
	}
	if err := db.Get(&teamID, `INSERT INTO teams (name, timezone, conversation_assignment_type) VALUES ('mine', 'UTC', 'Manual') RETURNING id`); err != nil {
		t.Fatalf("inserting team: %v", err)
	}
	if err := db.Get(&otherTeamID, `INSERT INTO teams (name, timezone, conversation_assignment_type) VALUES ('theirs', 'UTC', 'Manual') RETURNING id`); err != nil {
		t.Fatalf("inserting team: %v", err)
	}

	everyone := mustCreate(t, mgr, "everyone", "<p>x</p>", "all", nil, nil, []string{"replying"})
	mine := mustCreate(t, mgr, "mine only", "<p>x</p>", "user", &agentID, nil, []string{"replying"})
	mustCreate(t, mgr, "theirs only", "<p>x</p>", "user", &otherID, nil, []string{"replying"})
	myTeam := mustCreate(t, mgr, "my team", "<p>x</p>", "team", nil, &teamID, []string{"replying"})
	mustCreate(t, mgr, "other team", "<p>x</p>", "team", nil, &otherTeamID, []string{"replying"})
	noteOnly := mustCreate(t, mgr, "note only", "<p>x</p>", "all", nil, nil, []string{"adding_private_note"})

	visible, err := mgr.SearchCompact("", "", agentID, []int{teamID})
	if err != nil {
		t.Fatalf("SearchCompact: %v", err)
	}
	want := map[int]bool{everyone.ID: true, mine.ID: true, myTeam.ID: true, noteOnly.ID: true}
	if len(visible) != len(want) {
		t.Fatalf("got %d macros, want %d", len(visible), len(want))
	}
	for _, m := range visible {
		if !want[m.ID] {
			t.Errorf("macro %q leaked past visibility scoping", m.Name)
		}
	}

	replying, err := mgr.SearchCompact("", "replying", agentID, []int{teamID})
	if err != nil {
		t.Fatalf("SearchCompact with view: %v", err)
	}
	for _, m := range replying {
		if m.ID == noteOnly.ID {
			t.Error("view filter returned a macro not visible while replying")
		}
	}
	if len(replying) != 3 {
		t.Fatalf("got %d macros for the replying view, want 3", len(replying))
	}

	byName, err := mgr.SearchCompact("TEAM", "", agentID, []int{teamID})
	if err != nil {
		t.Fatalf("SearchCompact with query: %v", err)
	}
	if len(byName) != 1 || byName[0].ID != myTeam.ID {
		t.Fatalf("case-insensitive name search returned %d macros, want just 'my team'", len(byName))
	}

	noTeams, err := mgr.SearchCompact("", "", agentID, []int{})
	if err != nil {
		t.Fatalf("SearchCompact with no teams: %v", err)
	}
	for _, m := range noTeams {
		if m.ID == myTeam.ID {
			t.Error("team macro returned for an agent with no teams")
		}
	}
}

func mustCreate(t *testing.T, mgr *Manager, name, content, visibility string, userID, teamID *int, visibleWhen []string) models.Macro {
	t.Helper()
	m, err := mgr.Create(name, content, userID, teamID, visibility, visibleWhen, json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("creating macro %q: %v", name, err)
	}
	return m
}
