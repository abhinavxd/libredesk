package user

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/zerodha/logf"
)

func isInputError(err error) bool {
	var envErr envelope.Error
	return errors.As(err, &envErr) && envErr.ErrorType == envelope.InputError
}

func TestUsersScreenSafeguardsKeepAnAdminInCharge(t *testing.T) {
	db := testutil.NewDB(t, "users_admin")
	lo := logf.New(logf.Opts{})
	m, err := New(testutil.NewI18n(t), Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSystemUser(context.Background(), "", db); err != nil {
		t.Fatal(err)
	}
	var inboxID, support, billing, systemID int
	db.MustExec(`INSERT INTO inboxes(name,channel,"from") VALUES('Mail','email','support@example.test')`)
	db.Get(&inboxID, `SELECT id FROM inboxes`)
	db.Get(&support, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'support@example.test','mailbox') RETURNING id`, inboxID)
	db.Get(&billing, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'billing@example.test','alias') RETURNING id`, inboxID)
	db.Get(&systemID, `SELECT id FROM users WHERE email='System'`)

	owner, err := m.CreateManagedUser("Owner", "", "owner@example.test", "Admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := m.CreateManagedUser("Writer", "Person", "writer@example.test", "Contributor", []int{support})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(writer.Roles, []string{"Contributor"}) || len(writer.AddressIDs) != 1 || writer.AddressIDs[0] != int64(support) {
		t.Fatalf("created user = %+v", writer)
	}
	if _, err := m.CreateManagedUser("Dup", "", "WRITER@example.test", "Agent", nil); err == nil {
		t.Fatal("duplicate email accepted")
	}
	if _, err := m.CreateManagedUser("Bad", "", "bad@example.test", "Superuser", nil); !isInputError(err) {
		t.Fatalf("unknown role error = %v", err)
	}

	// The System user is neither listed nor editable.
	listed, err := m.GetManagedUsers()
	if err != nil || len(listed) != 2 {
		t.Fatalf("listed users = %d err=%v", len(listed), err)
	}
	if err := m.UpdateAccess(owner.ID, systemID, AccessUpdate{Role: "Agent", Enabled: true}); !isInputError(err) {
		t.Fatalf("System edit error = %v", err)
	}

	// Nobody changes their own role or disables themselves.
	if err := m.UpdateAccess(owner.ID, owner.ID, AccessUpdate{Role: "Agent", Enabled: true}); !isInputError(err) {
		t.Fatalf("self demotion error = %v", err)
	}
	if err := m.UpdateAccess(owner.ID, owner.ID, AccessUpdate{Role: "Admin", Enabled: false}); !isInputError(err) {
		t.Fatalf("self disable error = %v", err)
	}
	// Own addresses can still be edited.
	if err := m.UpdateAccess(owner.ID, owner.ID, AccessUpdate{Role: "Admin", Enabled: true, AddressIDs: []int{billing}}); err != nil {
		t.Fatal(err)
	}

	// The last enabled Admin cannot be demoted, disabled or deleted by anyone.
	if err := m.UpdateAccess(writer.ID, owner.ID, AccessUpdate{Role: "Agent", Enabled: true}); !isInputError(err) {
		t.Fatalf("last admin demotion error = %v", err)
	}
	if err := m.UpdateAccess(writer.ID, owner.ID, AccessUpdate{Role: "Admin", Enabled: false}); !isInputError(err) {
		t.Fatalf("last admin disable error = %v", err)
	}
	if err := m.DeleteManagedUser(writer.ID, owner.ID); !isInputError(err) {
		t.Fatalf("last admin delete error = %v", err)
	}
	if err := m.CheckAccessChange(writer.ID, owner.ID, []string{"Agent"}, true); !isInputError(err) {
		t.Fatalf("legacy API last admin error = %v", err)
	}

	// Promote the writer, after which the owner may step down.
	if err := m.UpdateAccess(owner.ID, writer.ID, AccessUpdate{Role: "Admin", Enabled: true, AddressIDs: []int{support, billing}}); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateAccess(writer.ID, owner.ID, AccessUpdate{Role: "Agent", Enabled: true, AddressIDs: []int{support}}); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetManagedUser(owner.ID)
	if err != nil || !slices.Equal(got.Roles, []string{"Agent"}) || len(got.AddressIDs) != 1 || got.AddressIDs[0] != int64(support) {
		t.Fatalf("demoted owner = %+v err=%v", got, err)
	}
	agent, err := m.GetAgentCachedOrLoad(owner.ID)
	if err != nil || slices.Contains(agent.Permissions, "users:manage") {
		t.Fatalf("stale cached permissions after demotion: %v %v", agent.Permissions, err)
	}

	// A disabled Admin does not count towards keeping someone in charge.
	if err := m.UpdateAccess(writer.ID, owner.ID, AccessUpdate{Role: "Admin", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateAccess(owner.ID, writer.ID, AccessUpdate{Role: "Agent", Enabled: true}); !isInputError(err) {
		t.Fatalf("demoting the only enabled admin error = %v", err)
	}

	// An empty address list clears direct grants; deletion removes them.
	if err := m.UpdateAccess(writer.ID, owner.ID, AccessUpdate{Role: "Agent", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	var grants int
	db.Get(&grants, `SELECT count(*) FROM email_address_users WHERE user_id=$1`, owner.ID)
	if grants != 0 {
		t.Fatalf("grants after clearing = %d", grants)
	}
	if err := m.DeleteManagedUser(writer.ID, writer.ID); !isInputError(err) {
		t.Fatalf("self delete error = %v", err)
	}
	if err := m.DeleteManagedUser(writer.ID, owner.ID); err != nil {
		t.Fatal(err)
	}
	if listed, _ := m.GetManagedUsers(); len(listed) != 1 {
		t.Fatalf("deleted user still listed: %+v", listed)
	}
}
