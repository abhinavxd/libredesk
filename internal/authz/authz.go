// package authz provides Casbin-based authorization.
package authz

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

// Enforcer is a wrapper around Casbin enforcer.
type Enforcer struct {
	enforcer *casbin.SyncedEnforcer
	lo       *logf.Logger
	i18n     *i18n.I18n
}

const casbinModel = `
	[request_definition]
	r = sub, obj, act

	[policy_definition]
	p = sub, obj, act

	[policy_effect]
	e = some(where (p.eft == allow))

	[matchers]
	m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`

// NewEnforcer initializes a new Enforcer with the hardcoded model
func NewEnforcer(lo *logf.Logger, i18n *i18n.I18n) (*Enforcer, error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin model: %v", err)
	}
	e, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %v", err)
	}

	return &Enforcer{enforcer: e, lo: lo, i18n: i18n}, nil
}

// LoadPermissions syncs user permissions with Casbin enforcer by removing existing
// policies and adding current permissions as new policies
func (e *Enforcer) LoadPermissions(user umodels.User) error {
	// Remove existing policies for the user
	_, err := e.enforcer.RemoveFilteredPolicy(0, strconv.Itoa(user.ID))
	if err != nil {
		return fmt.Errorf("failed to remove policies: %v", err)
	}

	// Add each permission as a policy
	for _, perm := range user.Permissions {
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid permission format: %s", perm)
		}

		userID, permObj, permAct := strconv.Itoa(user.ID), parts[0], parts[1]
		if _, err := e.enforcer.AddPolicy(userID, permObj, permAct); err != nil {
			return fmt.Errorf("failed to add casbin policy: %v", err)
		}
	}
	return nil
}

// Enforce checks if a user has permission to perform an action on an object.
func (e *Enforcer) Enforce(user umodels.User, obj, act string) (bool, error) {
	// Load permissions before enforcing.
	err := e.LoadPermissions(user)
	if err != nil {
		return false, err
	}
	// Check if the user has the required permission
	allowed, err := e.enforcer.Enforce(strconv.Itoa(user.ID), obj, act)
	if err != nil {
		return false, fmt.Errorf("error checking permission: %v", err)
	}
	return allowed, nil
}

// EnforceConversationAccess determines if a user has access to a specific conversation based on their permissions.
// Access can be granted under the following conditions:
// 1. User has the "read_all" permission, allowing access to all conversations.
// 2. User has the "read_assigned" permission and is the assigned user.
// 3. User has the "read_team_inbox" permission and is part of the assigned team, with the conversation unassigned to any specific user.
// 4. User has the "read_unassigned" permission and the conversation is unassigned to any user or team.
// Returns true if access is granted, false otherwise. In case of an error while checking permissions, returns false and the error.
func (e *Enforcer) EnforceConversationAccess(user umodels.User, conversation cmodels.Conversation) (bool, error) {
	checkPermission := func(action string) (bool, error) {
		allowed, err := e.Enforce(user, "conversations", action)
		if err != nil {
			e.lo.Error("error enforcing permission", "user_id", user.ID, "conversation_id", conversation.ID, "error", err)
			return false, envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorChecking", "name", "{globals.entities.permission}"), nil)
		}
		if !allowed {
			e.lo.Debug("permission denied", "user_id", user.ID, "action", action, "conversation_id", conversation.ID)
		}
		return allowed, nil
	}

	// Check `read` permission
	if allowed, err := checkPermission("read"); err != nil || !allowed {
		return allowed, err
	}

	// Check `read_all` permission
	if allowed, err := checkPermission("read_all"); err != nil || allowed {
		return allowed, err
	}

	// Check `read_assigned` permission for user-assigned conversations
	if conversation.AssignedUserID.Int == user.ID {
		if allowed, err := checkPermission("read_assigned"); err != nil || allowed {
			return allowed, err
		}
	}

	// Check `read_team_inbox` permission for team-assigned conversations
	if conversation.AssignedTeamID.Int > 0 && slices.Contains(user.Teams.IDs(), conversation.AssignedTeamID.Int) && conversation.AssignedUserID.Int == 0 {
		if allowed, err := checkPermission("read_team_inbox"); err != nil || allowed {
			return allowed, err
		}
	}

	// Check `read_unassigned` permission for unassigned conversations
	if conversation.AssignedUserID.Int == 0 && conversation.AssignedTeamID.Int == 0 {
		if allowed, err := checkPermission("read_unassigned"); err != nil || allowed {
			return allowed, err
		}
	}
	return false, nil
}

// EnforceMediaAccess checks for read access on linked model to media.
func (e *Enforcer) EnforceMediaAccess(user umodels.User, model string) (bool, error) {
	switch model {
	case "messages":
		allowed, err := e.Enforce(user, model, "read")
		if err != nil {
			e.lo.Error("error enforcing permission", "user_id", user.ID, "model", model, "error", err)
			return false, envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorChecking", "name", "{globals.entities.permission}"), nil)
		}
		if !allowed {
			return false, envelope.NewError(envelope.UnauthorizedError, e.i18n.T("authz.permissionDenied"), nil)
		}
	default:
		return true, nil
	}
	return true, nil
}
