// Package autoassigner continuously assigns conversations at regular intervals to users.
package autoassigner

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/mr-karan/balance"
	"github.com/zerodha/logf"
)

var (
	ErrTeamNotFound = errors.New("team not found")
)

const (
	AssignmentTypeRoundRobin = "Round robin"
)

type conversationStore interface {
	GetUnassignedConversations() ([]models.Conversation, error)
	UpdateConversationUserAssignee(conversationUUID string, userID int, user umodels.User) error
	ActiveUserConversationsCount(userID int) (int, error)
}

type teamStore interface {
	GetAll() ([]tmodels.Team, error)
	GetMembers(teamID int) ([]tmodels.TeamMember, error)
}

// Engine represents a manager for assigning unassigned conversations
// to team agents in a round-robin pattern.
type Engine struct {
	roundRobinBalancer map[int]*balance.Balance
	// Mutex to protect the balancer map
	balanceMu              sync.Mutex
	teamMaxAutoAssignments map[int]int

	systemUser        umodels.User
	conversationStore conversationStore
	teamStore         teamStore
	lo                *logf.Logger
	closed            bool
	closedMu          sync.Mutex
	wg                sync.WaitGroup
}

// New initializes a new Engine instance, set up with the provided team manager,
// conversation manager, and logger.
func New(teamStore teamStore, conversationStore conversationStore, systemUser umodels.User, lo *logf.Logger) (*Engine, error) {
	var e = Engine{
		conversationStore:      conversationStore,
		teamStore:              teamStore,
		systemUser:             systemUser,
		lo:                     lo,
		teamMaxAutoAssignments: make(map[int]int),
		roundRobinBalancer:     make(map[int]*balance.Balance),
	}
	return &e, nil
}

// Run initiates the conversation assignment process and is to be invoked as a goroutine.
// This function continuously assigns unassigned conversations to agents at regular intervals.
func (e *Engine) Run(ctx context.Context, autoAssignInterval time.Duration) {
	ticker := time.NewTicker(autoAssignInterval)
	defer ticker.Stop()

	e.wg.Add(1)
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.closedMu.Lock()
			closed := e.closed
			e.closedMu.Unlock()
			if closed {
				return
			}
			// Reload the balancer with latest team and user data.
			if err := e.reloadBalancer(); err != nil {
				e.lo.Error("error reloading balancer", "error", err)
			}
			// Start assigning conversations.
			if err := e.assignConversations(); err != nil {
				e.lo.Error("error assigning conversations", "error", err)
			}
		}
	}
}

// Close signals the Engine to stop its auto-assignment process.
// It sets the closed flag, which will cause the Run loop to exit.
func (e *Engine) Close() {
	e.closedMu.Lock()
	defer e.closedMu.Unlock()
	if e.closed {
		return
	}
	e.closed = true
	e.wg.Wait()
}

// reloadBalancer updates the round-robin balancer with the latest user and team data.
func (e *Engine) reloadBalancer() error {
	e.balanceMu.Lock()
	defer e.balanceMu.Unlock()

	err := e.populateTeamBalancer()
	if err != nil {
		e.lo.Error("error updating team balancer pool", "error", err)
		return err
	}
	return nil
}

// populateTeamBalancer populates the team balancer pool with the team members.
func (e *Engine) populateTeamBalancer() error {
	teams, err := e.teamStore.GetAll()
	if err != nil {
		return err
	}

	for _, team := range teams {
		if team.ConversationAssignmentType != AssignmentTypeRoundRobin {
			continue
		}

		users, err := e.teamStore.GetMembers(team.ID)
		if err != nil {
			e.lo.Error("error fetching team members", "team_id", team.ID, "error", err)
			continue
		}

		// Shuffle users to prevent ordering bias, as every app restart will pick the same first user.
		rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(users), func(i, j int) {
			users[i], users[j] = users[j], users[i]
		})

		// Initialize team balancer if missing
		if _, exists := e.roundRobinBalancer[team.ID]; !exists {
			e.lo.Debug("creating new balancer for team", "team_id", team.ID)
			e.roundRobinBalancer[team.ID] = balance.NewBalance()
		}

		balancer := e.roundRobinBalancer[team.ID]
		existingUsers := make(map[string]struct{})
		for _, user := range users {
			// Skip user if availability status is `away_manual` or `away_and_reassigning`
			if user.AvailabilityStatus == umodels.AwayManual || user.AvailabilityStatus == umodels.AwayAndReassigning {
				e.lo.Debug("user is away, skipping autoasssignment ", "team_id", team.ID, "user_id", user.ID, "availability_status", user.AvailabilityStatus)
				continue
			}

			// Add user to the balancer pool
			uid := strconv.Itoa(user.ID)
			existingUsers[uid] = struct{}{}
			if err := balancer.Add(uid, 1); err != nil {
				if err != balance.ErrDuplicateID {
					e.lo.Error("error adding user to balancer pool", "team_id", team.ID, "user_id", user.ID, "error", err)
				}
				continue
			}
			e.lo.Debug("added user to balancer pool", "team_id", team.ID, "user_id", user.ID)
		}

		// Remove users no longer in the team
		for _, id := range balancer.ItemIDs() {
			if _, exists := existingUsers[id]; !exists {
				if err := balancer.Remove(id); err != nil {
					e.lo.Error("error removing user from balancer pool", "team_id", team.ID, "user_id", id, "error", err)
				} else {
					e.lo.Debug("removed user from balancer pool", "team_id", team.ID, "user_id", id)
				}
			}
		}

		// Set max auto assigned conversations for the team
		e.teamMaxAutoAssignments[team.ID] = team.MaxAutoAssignedConversations
	}
	return nil
}

// assignConversations function fetches conversations that have been assigned to teams but not to any individual user,
// and then proceeds to assign them to team members based on a round-robin strategy.
func (e *Engine) assignConversations() error {
	unassignedConversations, err := e.conversationStore.GetUnassignedConversations()
	if err != nil {
		return fmt.Errorf("fetching unassigned conversations: %w", err)
	}

	if len(unassignedConversations) > 0 {
		e.lo.Debug("found unassigned conversations", "count", len(unassignedConversations))
	}

	for _, conversation := range unassignedConversations {
		// Get user from the pool.
		userIDStr, err := e.getUserFromPool(conversation.AssignedTeamID.Int)
		if err != nil {
			if err != ErrTeamNotFound {
				e.lo.Error("error fetching user from balancer pool", "conversation_uuid", conversation.UUID, "error", err)
			}
			continue
		}

		// Convert to int.
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			e.lo.Error("error converting user id from string to int", "user_id", userIDStr, "error", err)
			continue
		}

		// Get active conversations count for the user.
		activeConversationsCount, err := e.conversationStore.ActiveUserConversationsCount(userID)
		if err != nil {
			e.lo.Error("error fetching active conversations count for user", "user_id", userID, "error", err)
			continue
		}

		teamMaxAutoAssignments := e.teamMaxAutoAssignments[conversation.AssignedTeamID.Int]
		// Check if user has reached the max auto assigned conversations limit,
		// 0 is unlimited.
		if teamMaxAutoAssignments != 0 {
			if activeConversationsCount >= teamMaxAutoAssignments {
				e.lo.Debug("user has reached max auto assigned conversations limit, skipping auto assignment", "user_id", userID,
					"user_active_conversations_count", activeConversationsCount, "max_auto_assigned_conversations", teamMaxAutoAssignments)
				continue
			}
		}

		// Assign conversation to user.
		if err := e.conversationStore.UpdateConversationUserAssignee(conversation.UUID, userID, e.systemUser); err != nil {
			e.lo.Error("error assigning conversation", "conversation_uuid", conversation.UUID, "error", err)
			continue
		}
	}
	return nil
}

// getUserFromPool returns user ID from the team balancer pool.
func (e *Engine) getUserFromPool(assignedTeamID int) (string, error) {
	e.balanceMu.Lock()
	defer e.balanceMu.Unlock()

	pool, ok := e.roundRobinBalancer[assignedTeamID]
	if !ok {
		return "", ErrTeamNotFound
	}
	return pool.Get(), nil
}
