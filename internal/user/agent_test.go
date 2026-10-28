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

	all, err := mgr.GetAgentsCompact("", "", false, 0, 0)
	if err != nil {
		t.Fatalf("GetAgentsCompact without paging: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d agents, want all 5", len(all))
	}

	var seen []string
	for page := 1; ; page++ {
		batch, err := mgr.GetAgentsCompact("", "", false, page, 2)
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

func TestGetAgentsCompactSearch(t *testing.T) {
	mgr, db := newTestManager(t)
	db.MustExec(`INSERT INTO users (type, email, first_name, last_name) VALUES
		('agent', 'ada@example.com', 'Ada', 'Lovelace'),
		('agent', 'grace@example.com', 'Grace', 'Hopper'),
		('agent', 'alan@example.com', 'Alan', 'Turing')`)

	byLastName, err := mgr.GetAgentsCompact("hopper", "", false, 1, 30)
	if err != nil {
		t.Fatalf("GetAgentsCompact with query: %v", err)
	}
	if len(byLastName) != 1 || byLastName[0].Email.String != "grace@example.com" {
		t.Fatalf("query \"hopper\" = %+v, want grace", byLastName)
	}

	byEmail, err := mgr.GetAgentsCompact("alan@", "", false, 1, 30)
	if err != nil {
		t.Fatalf("GetAgentsCompact by email: %v", err)
	}
	if len(byEmail) != 1 || byEmail[0].Email.String != "alan@example.com" {
		t.Fatalf("query \"alan@\" = %+v, want alan", byEmail)
	}

	byFullName, err := mgr.GetAgentsCompact("ada lov", "", false, 1, 30)
	if err != nil {
		t.Fatalf("GetAgentsCompact by full name: %v", err)
	}
	if len(byFullName) != 1 || byFullName[0].Email.String != "ada@example.com" {
		t.Fatalf("query \"ada lov\" = %+v, want ada", byFullName)
	}
}

func TestGetAgentsCompactByIDs(t *testing.T) {
	mgr, db := newTestManager(t)
	db.MustExec(`INSERT INTO users (type, email, first_name, last_name) VALUES
		('agent', 'ada@example.com', 'Ada', 'Lovelace'),
		('agent', 'grace@example.com', 'Grace', 'Hopper'),
		('contact', 'customer@example.com', 'Customer', 'A')`)

	var adaID, contactID int
	if err := db.Get(&adaID, `SELECT id FROM users WHERE email = 'ada@example.com'`); err != nil {
		t.Fatalf("loading Ada ID: %v", err)
	}
	if err := db.Get(&contactID, `SELECT id FROM users WHERE email = 'customer@example.com'`); err != nil {
		t.Fatalf("loading contact ID: %v", err)
	}

	got, err := mgr.GetAgentsCompactByIDs([]int{adaID, contactID})
	if err != nil {
		t.Fatalf("GetAgentsCompactByIDs: %v", err)
	}
	if len(got) != 1 || got[0].Email.String != "ada@example.com" {
		t.Fatalf("GetAgentsCompactByIDs = %+v, want only ada", got)
	}
}

func TestGetAgentsCompactFilters(t *testing.T) {
	mgr, db := newTestManager(t)
	db.MustExec(`INSERT INTO users (type, email, first_name, last_name, enabled) VALUES
		('agent', 'active@example.com', 'Active', 'Agent', true),
		('agent', 'gone@example.com', 'Disabled', 'Agent', false),
		('ai_assistant', 'bot@example.com', 'Bot', '', true)`)

	enabledAgents, err := mgr.GetAgentsCompact("", "agent", true, 1, 30)
	if err != nil {
		t.Fatalf("GetAgentsCompact with filters: %v", err)
	}
	if len(enabledAgents) != 1 || enabledAgents[0].Email.String != "active@example.com" {
		t.Fatalf("enabled agent filter = %+v, want only active@example.com", enabledAgents)
	}

	unfiltered, err := mgr.GetAgentsCompact("", "", false, 1, 30)
	if err != nil {
		t.Fatalf("GetAgentsCompact without filters: %v", err)
	}
	if len(unfiltered) != 3 {
		t.Fatalf("got %d users without filters, want 3", len(unfiltered))
	}
}

func TestPasswordChangesAndResetToken(t *testing.T) {
	m, db := newTestManager(t)
	var id int
	if err := db.Get(&id, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'session@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	check := func(password string) {
		t.Helper()
		user, err := m.VerifyPassword("session@example.com", []byte(password))
		if err != nil || user.ID != id {
			t.Fatalf("password rejected: user=%d err=%v", user.ID, err)
		}
	}
	if err := m.UpdateAgent(id, "Agent", "", "session@example.com", nil /** roles **/, true /** enabled **/, "" /** availabilityStatus **/, "Test-Password-123!"); err != nil {
		t.Fatal(err)
	}
	check("Test-Password-123!")
	if err := m.UpdateAgent(id, "Agent", "", "session@example.com", nil /** roles **/, true /** enabled **/, "" /** availabilityStatus **/, "" /** newPassword **/); err != nil {
		t.Fatal(err)
	}
	check("Test-Password-123!")
	token, err := m.SetResetPasswordToken(id)
	if err != nil {
		t.Fatal(err)
	}
	resetID, err := m.ResetPassword(token, "Test-Password-456!")
	if err != nil || resetID != id {
		t.Fatalf("reset id = %d, err = %v", resetID, err)
	}
	check("Test-Password-456!")
	if _, err := m.ResetPassword(token, "Test-Password-789!"); err == nil {
		t.Fatal("used reset token accepted")
	}
	check("Test-Password-456!")
}
