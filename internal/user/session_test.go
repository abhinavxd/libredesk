package user

import "testing"

func TestPasswordChangesAdvanceSessionVersion(t *testing.T) {
	m, db := newTestManager(t)
	var id int
	if err := db.Get(&id, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'session@example.com', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	check := func(want int) {
		t.Helper()
		got, err := m.GetSessionVersion(id)
		if err != nil || got != want {
			t.Fatalf("session version = %d, err = %v, want %d", got, err, want)
		}
	}
	check(1)
	if err := m.UpdateAgent(id, "Agent", "", "session@example.com", nil, true, "", ""); err != nil {
		t.Fatal(err)
	}
	check(1)
	if err := m.UpdateAgent(id, "Agent", "", "session@example.com", nil, true, "", "Test-Password-123!"); err != nil {
		t.Fatal(err)
	}
	check(2)
	token, err := m.SetResetPasswordToken(id)
	if err != nil {
		t.Fatal(err)
	}
	resetID, err := m.ResetPassword(token, "Test-Password-456!")
	if err != nil || resetID != id {
		t.Fatalf("reset id = %d, err = %v", resetID, err)
	}
	check(3)
	if _, err := m.ResetPassword(token, "Test-Password-789!"); err == nil {
		t.Fatal("used reset token accepted")
	}
	check(3)
}
