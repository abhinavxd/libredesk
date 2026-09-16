package team

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "team")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating team manager: %v", err)
	}
	return mgr, db
}

func seedTeams(t *testing.T, db *sqlx.DB, names ...string) map[string]int {
	t.Helper()
	ids := make(map[string]int, len(names))
	for _, name := range names {
		var id int
		if err := db.Get(&id, `INSERT INTO teams (name, conversation_assignment_type) VALUES ($1, 'Manual') RETURNING id`, name); err != nil {
			t.Fatalf("seeding team %q: %v", name, err)
		}
		ids[name] = id
	}
	return ids
}

func TestGetAllCompactPagination(t *testing.T) {
	mgr, db := newTestManager(t)
	seedTeams(t, db, "alpha", "bravo", "charlie", "delta", "echo")

	all, err := mgr.GetAllCompact("", 0, 0)
	if err != nil {
		t.Fatalf("GetAllCompact without paging: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d teams, want all 5", len(all))
	}
	if all[0].Name != "alpha" || all[4].Name != "echo" {
		t.Fatalf("teams are not name ordered: %+v", all)
	}

	pageTwo, err := mgr.GetAllCompact("", 2, 2)
	if err != nil {
		t.Fatalf("GetAllCompact page 2: %v", err)
	}
	if len(pageTwo) != 2 || pageTwo[0].Name != "charlie" || pageTwo[1].Name != "delta" {
		t.Fatalf("page 2 of size 2 = %+v, want charlie and delta", pageTwo)
	}

	lastPage, err := mgr.GetAllCompact("", 3, 2)
	if err != nil {
		t.Fatalf("GetAllCompact page 3: %v", err)
	}
	if len(lastPage) != 1 || lastPage[0].Name != "echo" {
		t.Fatalf("page 3 of size 2 = %+v, want just echo", lastPage)
	}

	pastEnd, err := mgr.GetAllCompact("", 9, 2)
	if err != nil {
		t.Fatalf("GetAllCompact past the end: %v", err)
	}
	if len(pastEnd) != 0 {
		t.Fatalf("page past the end returned %d teams, want none", len(pastEnd))
	}
}

func TestGetAllCompactSearch(t *testing.T) {
	mgr, db := newTestManager(t)
	seedTeams(t, db, "Billing", "billing-escalations", "Refunds")

	matches, err := mgr.GetAllCompact("bill", 1, 30)
	if err != nil {
		t.Fatalf("GetAllCompact with query: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("query \"bill\" matched %d teams, want 2 (case insensitive): %+v", len(matches), matches)
	}

	middle, err := mgr.GetAllCompact("escalat", 1, 30)
	if err != nil {
		t.Fatalf("GetAllCompact matching mid-name: %v", err)
	}
	if len(middle) != 1 || middle[0].Name != "billing-escalations" {
		t.Fatalf("query \"escalat\" = %+v, want billing-escalations", middle)
	}

	none, err := mgr.GetAllCompact("nothing-matches-this", 1, 30)
	if err != nil {
		t.Fatalf("GetAllCompact with unmatched query: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("unmatched query returned %d teams, want none", len(none))
	}

	paged, err := mgr.GetAllCompact("bill", 2, 1)
	if err != nil {
		t.Fatalf("GetAllCompact page 2 of a query: %v", err)
	}
	if len(paged) != 1 || paged[0].Name != "billing-escalations" {
		t.Fatalf("page 2 of query \"bill\" = %+v, want billing-escalations", paged)
	}
}

func TestGetAllCompactByIDs(t *testing.T) {
	mgr, db := newTestManager(t)
	ids := seedTeams(t, db, "alpha", "bravo", "charlie")

	got, err := mgr.GetAllCompactByIDs([]int{ids["charlie"], ids["alpha"]})
	if err != nil {
		t.Fatalf("GetAllCompactByIDs: %v", err)
	}
	if len(got) != 2 || got[0].Name != "alpha" || got[1].Name != "charlie" {
		t.Fatalf("GetAllCompactByIDs = %+v, want alpha and charlie in name order", got)
	}

	withUnknown, err := mgr.GetAllCompactByIDs([]int{ids["bravo"], 999999})
	if err != nil {
		t.Fatalf("GetAllCompactByIDs with an unknown id: %v", err)
	}
	if len(withUnknown) != 1 || withUnknown[0].Name != "bravo" {
		t.Fatalf("unknown id should be skipped, got %+v", withUnknown)
	}

	empty, err := mgr.GetAllCompactByIDs([]int{})
	if err != nil {
		t.Fatalf("GetAllCompactByIDs with no ids: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetAllCompactByIDs with no ids returned %d teams, want none", len(empty))
	}
}
