package autoassigner

import (
	"io"
	"sort"
	"testing"

	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

type fakeTeamStore struct {
	teams   []tmodels.Team
	members map[int][]tmodels.TeamMember
}

func (f *fakeTeamStore) GetAll() ([]tmodels.Team, error) { return f.teams, nil }
func (f *fakeTeamStore) GetMembers(teamID int) ([]tmodels.TeamMember, error) {
	return f.members[teamID], nil
}

func TestPopulateTeamBalancerSkipOfflineAgents(t *testing.T) {
	members := []tmodels.TeamMember{
		{ID: 1, AvailabilityStatus: umodels.Online},
		{ID: 2, AvailabilityStatus: umodels.Offline},
		{ID: 3, AvailabilityStatus: umodels.Away},
		{ID: 4, AvailabilityStatus: umodels.AwayManual},
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
		members: map[int][]tmodels.TeamMember{10: {{ID: 1, AvailabilityStatus: umodels.Online}}},
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
