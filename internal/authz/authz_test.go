package authz

import (
	"fmt"
	"slices"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func newTestEnforcer(t *testing.T) *Enforcer {
	t.Helper()
	tr, err := i18n.New([]byte(`{"_.code":"en","_.name":"English","status.deniedPermission":"denied"}`))
	if err != nil {
		t.Fatalf("i18n init: %v", err)
	}
	lo := logf.New(logf.Opts{})
	e, err := NewEnforcer(&lo, tr)
	if err != nil {
		t.Fatalf("enforcer init: %v", err)
	}
	return e
}

func user(id int, perms []string, teamIDs ...int) umodels.User {
	teams := make(tmodels.TeamsCompact, 0, len(teamIDs))
	for _, tid := range teamIDs {
		teams = append(teams, tmodels.TeamCompact{ID: tid})
	}
	return umodels.User{
		ID:          id,
		Permissions: pq.StringArray(perms),
		Teams:       teams,
	}
}

func conv(assignedUser, assignedTeam int) cmodels.Conversation {
	c := cmodels.Conversation{}
	if assignedUser > 0 {
		c.AssignedUserID = null.IntFrom(assignedUser)
	}
	if assignedTeam > 0 {
		c.AssignedTeamID = null.IntFrom(assignedTeam)
	}
	return c
}

type convAccessCase struct {
	name string
	user umodels.User
	conv cmodels.Conversation
	want bool
}

type mediaAccessCase struct {
	name      string
	user      umodels.User
	model     string
	wantAllow bool
	wantErr   bool
}

func convAccessCases() []convAccessCase {
	const (
		me      = 1
		other   = 2
		myTeam  = 10
		notMine = 20
	)
	return []convAccessCase{
		{"no perms at all", user(me, nil), conv(0, 0), false},
		{"read missing even with read_all", user(me, []string{"conversations:read_all"}), conv(0, 0), false},
		{"read missing even with read_assigned + assignment", user(me, []string{"conversations:read_assigned"}), conv(me, 0), false},
		{"read alone, no scope perm", user(me, []string{"conversations:read"}), conv(0, 0), false},
		{"read alone, conversation assigned to user", user(me, []string{"conversations:read"}), conv(me, 0), false},

		{"read_all allows any conversation: unassigned", user(me, []string{"conversations:read", "conversations:read_all"}), conv(0, 0), true},
		{"read_all allows any conversation: assigned to other", user(me, []string{"conversations:read", "conversations:read_all"}), conv(other, notMine), true},
		{"read_all allows any conversation: assigned to me", user(me, []string{"conversations:read", "conversations:read_all"}), conv(me, myTeam), true},

		{"read_assigned: assigned to user", user(me, []string{"conversations:read", "conversations:read_assigned"}), conv(me, 0), true},
		{"read_assigned: assigned to other", user(me, []string{"conversations:read", "conversations:read_assigned"}), conv(other, 0), false},
		{"read_assigned: unassigned", user(me, []string{"conversations:read", "conversations:read_assigned"}), conv(0, 0), false},
		{"read_assigned: assigned to me + team I'm not in", user(me, []string{"conversations:read", "conversations:read_assigned"}, myTeam), conv(me, notMine), true},

		{"read_team_all: user in team, no user assigned", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(0, myTeam), true},
		{"read_team_all: user in team, user assigned", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(other, myTeam), true},
		{"read_team_all: user in team, self assigned", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(me, myTeam), true},
		{"read_team_all: user not in team", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(0, notMine), false},
		{"read_team_all: no team assigned", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(0, 0), false},
		{"read_team_all: user has no teams", user(me, []string{"conversations:read", "conversations:read_team_all"}), conv(0, myTeam), false},

		{"read_team_inbox: user in team, no user assigned", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(0, myTeam), true},
		{"read_team_inbox: user in team but user already assigned", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(other, myTeam), false},
		{"read_team_inbox: user in team and self assigned", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(me, myTeam), false},
		{"read_team_inbox: user not in team", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(0, notMine), false},
		{"read_team_inbox: no team on conversation", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(0, 0), false},

		{"read_unassigned: no user no team", user(me, []string{"conversations:read", "conversations:read_unassigned"}), conv(0, 0), true},
		{"read_unassigned: user assigned only", user(me, []string{"conversations:read", "conversations:read_unassigned"}), conv(other, 0), false},
		{"read_unassigned: team assigned only", user(me, []string{"conversations:read", "conversations:read_unassigned"}), conv(0, myTeam), false},
		{"read_unassigned: both assigned", user(me, []string{"conversations:read", "conversations:read_unassigned"}), conv(other, myTeam), false},

		{"combined: read_assigned + read_team_inbox, matches assigned", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_inbox"}, myTeam), conv(me, myTeam), true},
		{"combined: read_assigned + read_team_inbox, matches team_inbox", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_inbox"}, myTeam), conv(0, myTeam), true},
		{"combined: read_assigned + read_team_inbox, neither matches", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_inbox"}, myTeam), conv(other, notMine), false},
		{"combined: all scope perms, fully unassigned", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_all", "conversations:read_team_inbox", "conversations:read_unassigned"}, myTeam), conv(0, 0), true},

		// Multi-team user
		{"multi-team: user in 3 teams, conv on 1st team, read_team_all", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam, 11, 12), conv(0, myTeam), true},
		{"multi-team: user in 3 teams, conv on 2nd team, read_team_all", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam, 11, 12), conv(0, 11), true},
		{"multi-team: user in 3 teams, conv on 3rd team, read_team_all", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam, 11, 12), conv(0, 12), true},
		{"multi-team: user in 3 teams, conv on unrelated team, read_team_all", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam, 11, 12), conv(0, notMine), false},
		{"multi-team: user in 3 teams, conv on 2nd team, read_team_inbox no user", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam, 11, 12), conv(0, 11), true},
		{"multi-team: user in 3 teams, conv on 2nd team, read_team_inbox user assigned", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam, 11, 12), conv(other, 11), false},

		// read_all priority/short-circuit
		{"read_all overrides denying scopes: only read_assigned would deny", user(me, []string{"conversations:read", "conversations:read_all", "conversations:read_assigned"}), conv(other, notMine), true},
		{"read_all overrides denying scopes: nothing else matches", user(me, []string{"conversations:read", "conversations:read_all", "conversations:read_unassigned"}, myTeam), conv(other, notMine), true},
		{"read_all with every other scope: still allow", user(me, []string{"conversations:read", "conversations:read_all", "conversations:read_assigned", "conversations:read_team_all", "conversations:read_team_inbox", "conversations:read_unassigned"}, myTeam), conv(other, notMine), true},

		// OR fallthrough across 3+ scopes
		{"read_assigned + read_team_all + read_unassigned: only unassigned matches", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_all", "conversations:read_unassigned"}, myTeam), conv(0, 0), true},
		{"read_assigned + read_team_all + read_unassigned: only team_all matches", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_all", "conversations:read_unassigned"}, myTeam), conv(other, myTeam), true},
		{"read_assigned + read_team_all + read_unassigned: only assigned matches", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_all", "conversations:read_unassigned"}, myTeam), conv(me, notMine), true},
		{"read_assigned + read_team_all + read_unassigned: none match", user(me, []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_all", "conversations:read_unassigned"}, myTeam), conv(other, notMine), false},
		{"read_team_inbox + read_unassigned: team_inbox path", user(me, []string{"conversations:read", "conversations:read_team_inbox", "conversations:read_unassigned"}, myTeam), conv(0, myTeam), true},
		{"read_team_inbox + read_unassigned: unassigned path", user(me, []string{"conversations:read", "conversations:read_team_inbox", "conversations:read_unassigned"}, myTeam), conv(0, 0), true},
		{"read_team_inbox + read_unassigned: assigned-to-other deny", user(me, []string{"conversations:read", "conversations:read_team_inbox", "conversations:read_unassigned"}, myTeam), conv(other, 0), false},

		// User + team both assigned, multiple paths
		{"self assigned + in team: only read_team_all", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(me, myTeam), true},
		{"self assigned + in team: only read_assigned", user(me, []string{"conversations:read", "conversations:read_assigned"}, myTeam), conv(me, myTeam), true},
		{"self assigned + in team: only read_team_inbox (user assigned so team_inbox path skipped)", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(me, myTeam), false},
		{"other assigned + team in user teams: read_team_all allows", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(other, myTeam), true},
		{"other assigned + team in user teams: read_assigned denies (not me)", user(me, []string{"conversations:read", "conversations:read_assigned"}, myTeam), conv(other, myTeam), false},
		{"other assigned + team in user teams: read_team_inbox denies (user set)", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(other, myTeam), false},

		// Self assigned but team not in user's teams
		{"self assigned + team not mine: read_team_all denies", user(me, []string{"conversations:read", "conversations:read_team_all"}, myTeam), conv(me, notMine), false},
		{"self assigned + team not mine: read_assigned allows", user(me, []string{"conversations:read", "conversations:read_assigned"}, myTeam), conv(me, notMine), true},
		{"self assigned + team not mine: read_team_inbox denies", user(me, []string{"conversations:read", "conversations:read_team_inbox"}, myTeam), conv(me, notMine), false},

		// Boundary: user has zero teams
		{"no teams + read_team_all + team-assigned conv: deny", user(me, []string{"conversations:read", "conversations:read_team_all"}), conv(0, myTeam), false},
		{"no teams + read_team_inbox + team-assigned conv: deny", user(me, []string{"conversations:read", "conversations:read_team_inbox"}), conv(0, myTeam), false},
		{"no teams + read_assigned + self-assigned conv: allow", user(me, []string{"conversations:read", "conversations:read_assigned"}), conv(me, 0), true},

		// Duplicate / irrelevant perms in list (shouldn't change semantics)
		{"duplicate read entries", user(me, []string{"conversations:read", "conversations:read", "conversations:read_all"}), conv(0, 0), true},
		{"unrelated perms mixed in", user(me, []string{"messages:write", "tags:manage", "conversations:read", "conversations:read_assigned"}), conv(me, 0), true},
		{"typo perm doesn't match", user(me, []string{"conversations:read", "conversations:readd_all"}), conv(other, 0), false},
		{"obj typo doesn't match", user(me, []string{"conversations:read", "conversation:read_all"}), conv(other, 0), false},
	}
}

func mediaAccessCases() []mediaAccessCase {
	return []mediaAccessCase{
		{"messages with read", user(1, []string{"messages:read"}), "messages", true, false},
		{"messages without read", user(1, nil), "messages", false, true},
		{"messages with unrelated perms", user(1, []string{"conversations:read"}), "messages", false, true},
		{"messages with read among many", user(1, []string{"conversations:read_all", "messages:read", "tags:manage"}), "messages", true, false},
		{"non-messages model with no perms", user(1, nil), "conversations", true, false},
		{"non-messages model with perms", user(1, []string{"messages:read"}), "contacts", true, false},
		{"empty model treated as non-messages", user(1, nil), "", true, false},
		{"case-sensitive: Messages not equal to messages", user(1, nil), "Messages", true, false},
	}
}

func TestEnforceConversationAccess(t *testing.T) {
	e := newTestEnforcer(t)
	for _, tt := range convAccessCases() {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.EnforceConversationAccess(tt.user, tt.conv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestCanReadAssignment(t *testing.T) {
	for _, tt := range convAccessCases() {
		t.Run(tt.name, func(t *testing.T) {
			got := CanReadAssignment(tt.user, tt.conv.AssignedUserID, tt.conv.AssignedTeamID)
			if got != tt.want {
				t.Errorf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestCanReadAssignmentMatchesEnforceConversationAccess(t *testing.T) {
	e := newTestEnforcer(t)
	for _, tt := range convAccessCases() {
		t.Run(tt.name, func(t *testing.T) {
			helper := CanReadAssignment(tt.user, tt.conv.AssignedUserID, tt.conv.AssignedTeamID)
			method, err := e.EnforceConversationAccess(tt.user, tt.conv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if helper != method {
				t.Errorf("helper=%v method=%v (must stay in sync)", helper, method)
			}
		})
	}
}

func TestEnforceMediaAccess(t *testing.T) {
	e := newTestEnforcer(t)
	for _, tt := range mediaAccessCases() {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.EnforceMediaAccess(tt.user, tt.model)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tt.wantAllow {
				t.Errorf("got %v want %v", got, tt.wantAllow)
			}
		})
	}
}

func referenceAllow(perms []string, userID, assignedUser, assignedTeam int, userTeams []int) bool {
	has := func(p string) bool { return slices.Contains(perms, "conversations:"+p) }
	inTeam := assignedTeam > 0 && slices.Contains(userTeams, assignedTeam)

	if !has("read") {
		return false
	}
	if has("read_all") {
		return true
	}
	if has("read_assigned") && assignedUser == userID && userID > 0 {
		return true
	}
	if inTeam && has("read_team_all") {
		return true
	}
	if inTeam && assignedUser == 0 && has("read_team_inbox") {
		return true
	}
	if assignedUser == 0 && assignedTeam == 0 && has("read_unassigned") {
		return true
	}
	return false
}

func powerSet(items []string) [][]string {
	n := len(items)
	result := make([][]string, 0, 1<<n)
	for mask := 0; mask < (1 << n); mask++ {
		subset := make([]string, 0, n)
		for i, item := range items {
			if mask&(1<<i) != 0 {
				subset = append(subset, item)
			}
		}
		result = append(result, subset)
	}
	return result
}

func TestExhaustivePermutation(t *testing.T) {
	e := newTestEnforcer(t)

	scopes := []string{
		"conversations:read",
		"conversations:read_all",
		"conversations:read_assigned",
		"conversations:read_team_all",
		"conversations:read_team_inbox",
		"conversations:read_unassigned",
	}
	permSubsets := powerSet(scopes)

	const me = 1
	const other = 2

	teamConfigs := [][]int{
		nil,
		{10},
		{10, 11},
		{10, 11, 12},
	}
	assignedUsers := []int{0, me, other}
	assignedTeams := []int{0, 10, 11, 99}

	checked, failures := 0, 0
	for _, perms := range permSubsets {
		for _, teams := range teamConfigs {
			for _, au := range assignedUsers {
				for _, at := range assignedTeams {
					u := user(me, perms, teams...)
					c := conv(au, at)

					got, err := e.EnforceConversationAccess(u, c)
					if err != nil {
						t.Fatalf("error: %v", err)
					}
					want := referenceAllow(perms, me, au, at, teams)
					checked++
					if got != want {
						failures++
						if failures <= 10 {
							t.Errorf("perms=%v teams=%v assignedUser=%d assignedTeam=%d: got=%v want=%v",
								perms, teams, au, at, got, want)
						}
					}
				}
			}
		}
	}
	if failures > 10 {
		t.Errorf("... and %d more failures", failures-10)
	}
	t.Logf("checked %d combinations, %d failures", checked, failures)
}

func TestExhaustiveMediaAccess(t *testing.T) {
	e := newTestEnforcer(t)

	allPerms := []string{
		"messages:read", "messages:write",
		"conversations:read", "conversations:read_all",
		"tags:manage", "users:manage",
	}
	models := []string{"messages", "conversations", "tags", "users", "contacts", "", "MESSAGES", "Messages"}

	checked := 0
	for _, perms := range powerSet(allPerms) {
		for _, m := range models {
			u := user(1, perms)
			got, err := e.EnforceMediaAccess(u, m)

			want, wantErr := true, false
			if m == "messages" && !slices.Contains(perms, "messages:read") {
				want, wantErr = false, true
			}
			checked++
			if got != want {
				t.Errorf("perms=%v model=%q: got allow=%v want=%v", perms, m, got, want)
			}
			if wantErr && err == nil {
				t.Errorf("perms=%v model=%q: expected err", perms, m)
			}
			if !wantErr && err != nil {
				t.Errorf("perms=%v model=%q: unexpected err: %v", perms, m, err)
			}
		}
	}
	t.Logf("checked %d combinations", checked)
}

func TestPowerSet(t *testing.T) {
	got := powerSet([]string{"a", "b", "c"})
	if len(got) != 8 {
		t.Fatalf("expected 8 subsets for 3 items, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, s := range got {
		seen[fmt.Sprint(s)] = true
	}
	if len(seen) != 8 {
		t.Errorf("duplicate subsets returned")
	}
}
