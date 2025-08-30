// package authz provides Casbin-based authorization.
package authz

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

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
	// User permissions cache to avoid loading policies into Casbin every time.
	permsCache   map[int][]string
	permsCacheMu sync.RWMutex
	lo           *logf.Logger
	i18n         *i18n.I18n
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

	return &Enforcer{
		enforcer:   e,
		permsCache: make(map[int][]string),
		lo:         lo,
		i18n:       i18n,
	}, nil
}

// LoadPermissions syncs user permissions with Casbin enforcer by removing existing policies and adding current permissions as new policies.
func (e *Enforcer) LoadPermissions(user umodels.User) error {
	e.permsCacheMu.RLock()
	cached, exists := e.permsCache[user.ID]
	e.permsCacheMu.RUnlock()

	if exists && slices.Equal(cached, user.Permissions) {
		return nil
	}

	e.lo.Debug("loading user permissions in enforcer cache", "user_id", user.ID, "permissions", user.Permissions)

	// Build all policies and add them to the enforcer.
	var policies [][]string
	userID := strconv.Itoa(user.ID)
	for _, perm := range user.Permissions {
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid permission format: %s", perm)
		}
		policies = append(policies, []string{userID, parts[0], parts[1]})
	}

	_, err := e.enforcer.RemoveFilteredPolicy(0, userID)
	if err != nil {
		return fmt.Errorf("failed to remove policies: %v", err)
	}

	if len(policies) > 0 {
		_, err = e.enforcer.AddPolicies(policies)
		if err != nil {
			return fmt.Errorf("failed to add policies: %v", err)
		}
	}

	// Update permsCache with the latest permissions
	e.permsCacheMu.Lock()
	e.permsCache[user.ID] = slices.Clone(user.Permissions)
	e.permsCacheMu.Unlock()

	return nil
}

// InvalidateUserCache removes user from permsCache to be called when user permissions change.
func (e *Enforcer) InvalidateUserCache(userID int) {
	e.permsCacheMu.Lock()
	delete(e.permsCache, userID)
	e.permsCacheMu.Unlock()
}

// Enforce checks if a user has permission to perform an action on an object.
func (e *Enforcer) Enforce(user umodels.User, obj, act string) (bool, error) {
	// Load permissions before enforcing as user perjissions might have changed.
	err := e.LoadPermissions(user)
	if err != nil {
		e.lo.Error("error loading permissions", "user_id", user.ID, "object", obj, "action", act, "error", err)
		return false, err
	}
	// Check if the user has the required permission
	allowed, err := e.enforcer.Enforce(strconv.Itoa(user.ID), obj, act)
	if err != nil {
		e.lo.Error("error checking permission", "user_id", user.ID, "object", obj, "action", act, "error", err)
		return false, fmt.Errorf("error checking permission: %v", err)
	}
	return allowed, nil
}

// EnforceConversationAccess determines if a user has access to a specific conversation based on their permissions.
// Requires basic "read" permission AND one of the following conditions:
// 1. User has the "read_all" permission, allowing access to all conversations.
// 2. User has the "read_assigned" permission and is the assigned user.
// 3. User has the "read_team_inbox" permission and is part of the assigned team, with the conversation NOT assigned to any user.
// 4. User has the "read_unassigned" permission and the conversation is not assigned to any user or team.
// Returns true if access is granted, false otherwise. In case of an error while checking permissions returns false and the error.
func (e *Enforcer) EnforceConversationAccess(user umodels.User, conversation cmodels.Conversation) (bool, error) {
	checkPermission := func(action string) (bool, error) {
		allowed, err := e.Enforce(user, "conversations", action)
		if err != nil {
			e.lo.Error("error enforcing permission", "user_id", user.ID, "conversation_id", conversation.ID, "error", err)
			return false, envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorChecking", "name", "{globals.terms.permission}"), nil)
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
	// TODO: Pick this table / model name from the package/models/models.go
	case "messages":
		allowed, err := e.Enforce(user, model, "read")
		if err != nil {
			e.lo.Error("error enforcing permission", "user_id", user.ID, "model", model, "error", err)
			return false, envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorChecking", "name", "{globals.terms.permission}"), nil)
		}
		if !allowed {
			return false, envelope.NewError(envelope.UnauthorizedError, e.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil)
		}
	default:
		return true, nil
	}
	return true, nil
}
