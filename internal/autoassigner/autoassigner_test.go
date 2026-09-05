package autoassigner

import (
	"testing"

	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/mr-karan/balance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zerodha/logf"
)

type mockTeamStore struct {
	mock.Mock
}

func (m *mockTeamStore) GetAll() ([]tmodels.Team, error) {
	args := m.Called()
	return args.Get(0).([]tmodels.Team), args.Error(1)
}

func (m *mockTeamStore) GetMembers(teamID int) ([]tmodels.TeamMember, error) {
	args := m.Called(teamID)
	return args.Get(0).([]tmodels.TeamMember), args.Error(1)
}

func createTestEngine(store *mockTeamStore) *Engine {
	logger := logf.New(logf.Opts{Level: logf.DebugLevel})
	return &Engine{
		teamStore:              store,
		lo:                     &logger,
		teamMaxAutoAssignments: make(map[int]int),
		roundRobinBalancer:     make(map[int]*balance.Balance),
	}
}

func TestPopulateTeamBalancer_OnlyOnlineUsersAddedToPool(t *testing.T) {
	store := new(mockTeamStore)
	engine := createTestEngine(store)

	teams := []tmodels.Team{
		{ID: 1, ConversationAssignmentType: AssignmentTypeRoundRobin},
	}
	members := []tmodels.TeamMember{
		{ID: 1, TeamID: 1, AvailabilityStatus: umodels.Online},
		{ID: 2, TeamID: 1, AvailabilityStatus: umodels.Away},
		{ID: 3, TeamID: 1, AvailabilityStatus: umodels.AwayManual},
		{ID: 4, TeamID: 1, AvailabilityStatus: umodels.AwayAndReassigning},
		{ID: 5, TeamID: 1, AvailabilityStatus: umodels.Offline},
	}
	store.On("GetAll").Return(teams, nil)
	store.On("GetMembers", 1).Return(members, nil)

	err := engine.populateTeamBalancer()

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"1"}, engine.roundRobinBalancer[1].ItemIDs())
}
