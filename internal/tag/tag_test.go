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

	all, err := mgr.GetAll(0, 0)
	if err != nil {
		t.Fatalf("GetAll without paging: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d tags, want all 5", len(all))
	}

	pageTwo, err := mgr.GetAll(2, 2)
	if err != nil {
		t.Fatalf("GetAll page 2: %v", err)
	}
	if len(pageTwo) != 2 || pageTwo[0].Name != "charlie" || pageTwo[1].Name != "delta" {
		t.Fatalf("page 2 of size 2 = %+v, want charlie and delta", pageTwo)
	}

	lastPage, err := mgr.GetAll(3, 2)
	if err != nil {
		t.Fatalf("GetAll page 3: %v", err)
	}
	if len(lastPage) != 1 || lastPage[0].Name != "echo" {
		t.Fatalf("page 3 of size 2 = %+v, want just echo", lastPage)
	}
}
