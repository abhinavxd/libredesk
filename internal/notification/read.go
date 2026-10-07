package notifier

import (
	"database/sql"
	"errors"
	"slices"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

type NotificationAgentStore interface {
	GetAgentCachedOrLoad(int) (umodels.User, error)
}

type DeliveryChecker interface {
	ShouldDeliver(int, models.NotificationReference, models.NotificationChannel) (bool, error)
}

type ReplyDeliveryChecker struct {
	notifications *UserNotificationManager
	agents        NotificationAgentStore
	prefs         NotificationPreferenceStore
}

type replyDeliveryState struct {
	AssignedUserID null.Int `db:"assigned_user_id"`
	AssignedTeamID null.Int `db:"assigned_team_id"`
	Participating  bool     `db:"participating"`
	Seen           bool     `db:"seen"`
}

func NewReplyDeliveryChecker(notifications *UserNotificationManager, agents NotificationAgentStore, prefs NotificationPreferenceStore) *ReplyDeliveryChecker {
	return &ReplyDeliveryChecker{notifications: notifications, agents: agents, prefs: prefs}
}

func (c *ReplyDeliveryChecker) ShouldDeliver(userID int, ref models.NotificationReference, channel models.NotificationChannel) (bool, error) {
	if !models.IsReply(ref.Type) {
		return true, nil
	}
	agent, err := c.agents.GetAgentCachedOrLoad(userID)
	if err != nil {
		return false, err
	}
	enabled, err := c.prefs.EnabledChannels([]int{userID}, ref.Type)
	if err != nil || !slices.Contains(enabled[userID], channel) {
		return false, err
	}
	var state replyDeliveryState
	if err := c.notifications.q.ReplyDeliveryState.Get(&state, userID, ref.ConversationID, ref.MessageID, ref.NotificationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return !state.Seen && CanReceiveReply(agent, ref.Type, state.AssignedUserID, state.AssignedTeamID, state.Participating), nil
}

// CanReceiveReply reports whether agent should get a reply notification of type nType.
func CanReceiveReply(agent umodels.User, nType models.NotificationType, assignedUserID, assignedTeamID null.Int, participating bool) bool {
	if !agent.Enabled || !authz.CanReadAssignment(agent, assignedUserID, assignedTeamID) {
		return false
	}
	if nType == models.NotificationTypeNewReplyParticipating {
		return participating && assignedUserID.Int != agent.ID
	}
	return assignedUserID.Valid && assignedUserID.Int == agent.ID
}
