package macro

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	db := testutil.NewDB(t, "macro")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating macro manager: %v", err)
	}
	return mgr
}

func TestGetAllCompact(t *testing.T) {
	mgr := newTestManager(t)

	withContent, err := mgr.Create("with content", "<p>hello</p>", nil, nil, "all", []string{"replying"}, json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("creating macro: %v", err)
	}
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

	compact, err := mgr.GetAllCompact()
	if err != nil {
		t.Fatalf("GetAllCompact: %v", err)
	}
	if len(compact) != 2 {
		t.Fatalf("GetAllCompact returned %d macros, want 2", len(compact))
	}
	for _, m := range compact {
		switch m.ID {
		case withContent.ID:
			if !m.HasMessageContent {
				t.Error("macro with content has HasMessageContent=false")
			}
			if m.Name != "with content" || len(m.Actions) == 0 {
				t.Error("compact row lost name or actions")
			}
		case withoutContent.ID:
			if m.HasMessageContent {
				t.Error("macro without content has HasMessageContent=true")
			}
		default:
			t.Errorf("unexpected macro id %d", m.ID)
		}
	}

	got, err := mgr.Get(withContent.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.MessageContent != "<p>hello</p>" {
		t.Errorf("Get returned message content %q", got.MessageContent)
	}
}
