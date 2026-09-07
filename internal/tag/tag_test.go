package tag

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	db := testutil.NewDB(t, "tag")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating tag manager: %v", err)
	}
	return mgr
}

func TestGetAllPagination(t *testing.T) {
	mgr := newTestManager(t)
	for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
		if _, err := mgr.Create(name); err != nil {
			t.Fatalf("creating tag %q: %v", name, err)
		}
	}

	all, err := mgr.GetAll("", 0, 0)
	if err != nil {
		t.Fatalf("GetAll without paging: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d tags, want all 5", len(all))
	}

	pageTwo, err := mgr.GetAll("", 2, 2)
	if err != nil {
		t.Fatalf("GetAll page 2: %v", err)
	}
	if len(pageTwo) != 2 || pageTwo[0].Name != "charlie" || pageTwo[1].Name != "delta" {
		t.Fatalf("page 2 of size 2 = %+v, want charlie and delta", pageTwo)
	}

	lastPage, err := mgr.GetAll("", 3, 2)
	if err != nil {
		t.Fatalf("GetAll page 3: %v", err)
	}
	if len(lastPage) != 1 || lastPage[0].Name != "echo" {
		t.Fatalf("page 3 of size 2 = %+v, want just echo", lastPage)
	}
}

func TestGetAllSearch(t *testing.T) {
	mgr := newTestManager(t)
	for _, name := range []string{"billing", "refund", "billing-dispute"} {
		if _, err := mgr.Create(name); err != nil {
			t.Fatalf("creating tag %q: %v", name, err)
		}
	}

	matches, err := mgr.GetAll("bill", 1, 30)
	if err != nil {
		t.Fatalf("GetAll with query: %v", err)
	}
	if len(matches) != 2 || matches[0].Name != "billing" || matches[1].Name != "billing-dispute" {
		t.Fatalf("query \"bill\" = %+v, want billing and billing-dispute", matches)
	}

	none, err := mgr.GetAll("nothing-matches-this", 1, 30)
	if err != nil {
		t.Fatalf("GetAll with unmatched query: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("unmatched query returned %d tags, want none", len(none))
	}
}

func TestGetByIDs(t *testing.T) {
	mgr := newTestManager(t)
	var ids []int
	for _, name := range []string{"alpha", "bravo", "charlie"} {
		tag, err := mgr.Create(name)
		if err != nil {
			t.Fatalf("creating tag %q: %v", name, err)
		}
		ids = append(ids, tag.ID)
	}

	got, err := mgr.GetByIDs([]int{ids[2], ids[0]})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(got) != 2 || got[0].Name != "alpha" || got[1].Name != "charlie" {
		t.Fatalf("GetByIDs = %+v, want alpha and charlie", got)
	}

	empty, err := mgr.GetByIDs([]int{})
	if err != nil {
		t.Fatalf("GetByIDs with no ids: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetByIDs with no ids returned %d tags, want none", len(empty))
	}
}
