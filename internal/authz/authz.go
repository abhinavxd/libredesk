package authz

import (
	"fmt"
	"slices"

	authzmodels "github.com/jakedolan443/fernmail/internal/authz/models"
	cmodels "github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type Enforcer struct {
	lo            *logf.Logger
	i18n          *i18n.I18n
	inboxAccess   func(userID, inboxID int) (bool, error)
	addressAccess func(userID, addressID int) (bool, error)
}

func NewEnforcer(lo *logf.Logger, i18n *i18n.I18n, checkInbox func(int, int) (bool, error), checkAddress ...func(int, int) (bool, error)) (*Enforcer, error) {
	if checkInbox == nil {
		return nil, fmt.Errorf("inbox access checker is required")
	}
	addressChecker := checkInbox
	if len(checkAddress) > 0 && checkAddress[0] != nil {
		addressChecker = checkAddress[0]
	}
	return &Enforcer{lo: lo, i18n: i18n, inboxAccess: checkInbox, addressAccess: addressChecker}, nil
}

// Enforce returns true if the user's permission list contains "obj:act".
func (e *Enforcer) Enforce(user umodels.User, obj, act string) (bool, error) {
	return slices.Contains(user.Permissions, obj+":"+act), nil
}

// EnforceConversationAccess determines if a user has access to a specific conversation based on their permissions.
// Requires inbox access and basic "read" permission AND one of the following conditions:
// 1. User has the "read_all" permission, allowing access to all conversations.
// 2. User has the "read_assigned" permission and is the assigned user.
// 3. User has the "read_team_inbox" permission and is part of the assigned team, with the conversation NOT assigned to any user.
// 4. User has the "read_unassigned" permission and the conversation is not assigned to any user or team.
// Returns true if access is granted, false otherwise. In case of an error while checking permissions returns false and the error.
func (e *Enforcer) EnforceConversationAccess(user umodels.User, conversation cmodels.Conversation) (bool, error) {
	if !CanReadAssignment(user, conversation.AssignedUserID, conversation.AssignedTeamID) {
		return false, nil
	}
	if conversation.AddressID.Valid {
		if e.addressAccess == nil {
			return false, fmt.Errorf("address access checker is required")
		}
		return e.addressAccess(user.ID, conversation.AddressID.Int)
	}
	if e.inboxAccess == nil {
		return false, fmt.Errorf("inbox access checker is required")
	}
	return e.inboxAccess(user.ID, conversation.InboxID)
}

func CanReadAssignment(user umodels.User, assignedUserID, assignedTeamID null.Int) bool {
	if !slices.Contains(user.Permissions, authzmodels.PermConversationsRead) {
		return false
	}
	if slices.Contains(user.Permissions, authzmodels.PermConversationsReadAll) {
		return true
	}
	if assignedUserID.Valid && assignedUserID.Int == user.ID &&
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadAssigned) {
		return true
	}
	if assignedTeamID.Valid && slices.Contains(user.Teams.IDs(), assignedTeamID.Int) {
		if slices.Contains(user.Permissions, authzmodels.PermConversationsReadTeamAll) {
			return true
		}
		if !assignedUserID.Valid && slices.Contains(user.Permissions, authzmodels.PermConversationsReadTeamInbox) {
			return true
		}
	}
	if !assignedUserID.Valid && !assignedTeamID.Valid &&
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadUnassigned) {
		return true
	}
	return false
}

// EnforceMediaAccess checks read access on the model linked to a media item.
func (e *Enforcer) EnforceMediaAccess(user umodels.User, model string) (bool, error) {
	if model != "messages" {
		return true, nil
	}
	if !slices.Contains(user.Permissions, "messages:read") {
		return false, envelope.NewError(envelope.UnauthorizedError, e.i18n.T("status.deniedPermission"), nil)
	}
	return true, nil
}
