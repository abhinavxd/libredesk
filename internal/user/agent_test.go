package user

import (
	"fmt"
	"testing"
)

func TestGetAgentsCompactPagination(t *testing.T) {
	mgr, db := newTestManager(t)
	for i := 1; i <= 5; i++ {
		db.MustExec(`INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', $1, $2, '')`,
			fmt.Sprintf("agent%d@example.com", i), fmt.Sprintf("Agent%d", i))
	}

	all, err := mgr.GetAgentsCompact(0, 0)
	if err != nil {
		t.Fatalf("GetAgentsCompact without paging: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d agents, want all 5", len(all))
	}

	var seen []string
	for page := 1; ; page++ {
		batch, err := mgr.GetAgentsCompact(page, 2)
		if err != nil {
			t.Fatalf("GetAgentsCompact page %d: %v", page, err)
		}
		for _, a := range batch {
			seen = append(seen, a.Email.String)
		}
		if len(batch) < 2 {
			break
		}
	}
	if len(seen) != 5 {
		t.Fatalf("paging collected %d agents, want 5: %v", len(seen), seen)
	}
	for i, email := range seen {
		if want := fmt.Sprintf("agent%d@example.com", i+1); email != want {
			t.Fatalf("paging order broken at %d: got %q, want %q", i, email, want)
		}
	}
}
