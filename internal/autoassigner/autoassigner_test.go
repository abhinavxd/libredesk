package autoassigner

import (
	"io"
	"slices"
	"sort"
	"testing"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
	"github.com/mr-karan/balance"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type assignmentStore struct {
	count, countCalls, claims int
}

var canReply = pq.StringArray{authzmodels.PermMessagesWrite}

type fakeTeamStore struct {
	teams   []tmodels.Team
	members map[int][]tmodels.TeamMember
}

func (f *fakeTeamStore) GetAll() ([]tmodels.Team, error) { return f.teams, nil }

// GetMembers returns a copy, like the DB-backed store, because populateTeamBalancer
// shuffles the slice in place.
func (f *fakeTeamStore) GetMembers(teamID int) ([]tmodels.TeamMember, error) {
	return slices.Clone(f.members[teamID]), nil
}

func (s *assignmentStore) GetUnassignedConversations() ([]models.Conversation, error) {
	return []models.Conversation{{UUID: "ticket", AssignedTeamID: null.IntFrom(1)}}, nil
}

func (s *assignmentStore) ActiveUserConversationsCount(userID int) (int, error) {
	s.countCalls++
	return s.count, nil
}

func (s *assignmentStore) ClaimUnassignedConversation(uuid string, userID, teamID int, user umodels.User) error {
	s.claims++
	return nil
}

func TestPopulateTeamBalancerSkipOfflineAgents(t *testing.T) {
	members := []tmodels.TeamMember{
		{ID: 1, AvailabilityStatus: umodels.Online, Permissions: canReply},
		{ID: 2, AvailabilityStatus: umodels.Offline, Permissions: canReply},
		{ID: 3, AvailabilityStatus: umodels.Away, Permissions: canReply},
		{ID: 4, AvailabilityStatus: umodels.AwayManual, Permissions: canReply},
	}
	tests := []struct {
		name string
		skip bool
		want []string
	}{
		{"disabled keeps offline agents", false, []string{"1", "2", "3"}},
		{"enabled drops offline agents", true, []string{"1", "3"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeTeamStore{
				teams:   []tmodels.Team{{ID: 10, ConversationAssignmentType: AssignmentTypeRoundRobin, SkipOfflineAgents: tc.skip}},
				members: map[int][]tmodels.TeamMember{10: members},
			}
			lo := logf.New(logf.Opts{Writer: io.Discard})
			e, _ := New(store, nil, umodels.User{}, &lo)
			if err := e.populateTeamBalancer(); err != nil {
				t.Fatal(err)
			}
			got := e.roundRobinBalancer[10].ItemIDs()
			sort.Strings(got)
			if len(got) != len(tc.want) {
				t.Fatalf("pool = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("pool = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestPopulateTeamBalancerRemovesAgentWhoGoesOffline(t *testing.T) {
	store := &fakeTeamStore{
		teams:   []tmodels.Team{{ID: 10, ConversationAssignmentType: AssignmentTypeRoundRobin, SkipOfflineAgents: true}},
		members: map[int][]tmodels.TeamMember{10: {{ID: 1, AvailabilityStatus: umodels.Online, Permissions: canReply}}},
	}
	lo := logf.New(logf.Opts{Writer: io.Discard})
	e, _ := New(store, nil, umodels.User{}, &lo)
	if err := e.populateTeamBalancer(); err != nil {
		t.Fatal(err)
	}

	store.members[10][0].AvailabilityStatus = umodels.Offline
	if err := e.populateTeamBalancer(); err != nil {
		t.Fatal(err)
	}
	if got := e.roundRobinBalancer[10].ItemIDs(); len(got) != 0 {
		t.Fatalf("pool = %v, want empty", got)
	}
}

func TestPopulateTeamBalancerSkipsAgentsWhoCannotReply(t *testing.T) {
	store := &fakeTeamStore{
		teams: []tmodels.Team{{ID: 10, ConversationAssignmentType: AssignmentTypeRoundRobin}},
		members: map[int][]tmodels.TeamMember{10: {
			{ID: 1, AvailabilityStatus: umodels.Online, Permissions: canReply},
			{ID: 2, AvailabilityStatus: umodels.Online, Permissions: pq.StringArray{authzmodels.PermMessagesWritePrivate}},
			{ID: 3, AvailabilityStatus: umodels.Online},
		}},
	}
	lo := logf.New(logf.Opts{Writer: io.Discard})
	e, _ := New(store, nil, umodels.User{}, &lo)
	if err := e.populateTeamBalancer(); err != nil {
		t.Fatal(err)
	}
	if got := e.roundRobinBalancer[10].ItemIDs(); len(got) != 1 || got[0] != "1" {
		t.Fatalf("pool = %v, want [1]", got)
	}

	// An agent whose role loses the permission leaves the pool on the next refresh.
	store.members[10][0].Permissions = nil
	if err := e.populateTeamBalancer(); err != nil {
		t.Fatal(err)
	}
	if got := e.roundRobinBalancer[10].ItemIDs(); len(got) != 0 {
		t.Fatalf("pool = %v, want empty", got)
	}
}

func TestAssignmentOnlyCountsWhenTeamHasLimit(t *testing.T) {
	for _, tt := range []struct {
		name                                string
		limit, count, wantCalls, wantClaims int
	}{
		{"unlimited", 0, 10000, 0, 1},
		{"below limit", 5, 4, 1, 1},
		{"at limit", 5, 5, 1, 0},
		{"above limit", 5, 6, 1, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &assignmentStore{count: tt.count}
			pool := balance.NewBalance()
			if err := pool.Add("2", 1 /** weight **/); err != nil {
				t.Fatal(err)
			}
			lo := logf.New(logf.Opts{})
			engine := &Engine{
				roundRobinBalancer:     map[int]*balance.Balance{1: pool},
				teamMaxAutoAssignments: map[int]int{1: tt.limit},
				conversationStore:      store,
				lo:                     &lo,
			}
			if err := engine.assignConversations(); err != nil {
				t.Fatal(err)
			}
			if store.countCalls != tt.wantCalls || store.claims != tt.wantClaims {
				t.Fatalf("count calls=%d claims=%d, want calls=%d claims=%d", store.countCalls, store.claims, tt.wantCalls, tt.wantClaims)
			}
		})
	}
}
