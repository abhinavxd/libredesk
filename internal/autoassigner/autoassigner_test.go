package autoassigner

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/mr-karan/balance"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type assignmentStore struct {
	count, countCalls, claims int
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
