package user

import (
	"context"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"golang.org/x/crypto/bcrypt"
)

// MonitorAgentAvailability continuously checks for user activity and sets them offline if inactive for more than 5 minutes.
func (u *Manager) MonitorAgentAvailability(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			u.markInactiveAgentsOffline()
		case <-ctx.Done():
			return
		}
	}
}

// GetAgent retrieves an agent by ID and also caches it for future requests.
func (u *Manager) GetAgent(id int, email string) (models.User, error) {
	agent, err := u.Get(id, email, models.UserTypeAgent)
	if err != nil {
		return models.User{}, err
	}

	u.agentCacheMu.Lock()
	u.agentCache[id] = agent
	u.agentCacheMu.Unlock()

	return agent, nil
}

// GetAgentFromCache retrieves an agent from the cache by ID.
func (u *Manager) GetAgentFromCache(id int) (models.User, bool) {
	u.agentCacheMu.RLock()
	defer u.agentCacheMu.RUnlock()
	agent, exists := u.agentCache[id]
	if !exists {
		return models.User{}, false
	}
	return agent, true
}

// GetAgentCachedOrLoad retrieves an agent from cache, falling back to DB if not cached.
func (u *Manager) GetAgentCachedOrLoad(id int) (models.User, error) {
	if agent, exists := u.GetAgentFromCache(id); exists {
		return agent, nil
	}
	return u.GetAgent(id, "")
}

// InvalidateAgentCache invalidates the agent cache for a specific agent ID.
func (u *Manager) InvalidateAgentCache(id int) {
	u.agentCacheMu.Lock()
	defer u.agentCacheMu.Unlock()
	delete(u.agentCache, id)
}

// GetAgentsCompact returns a compact list of agents with limited fields.
func (u *Manager) GetAgentsCompact() ([]models.UserCompact, error) {
	var users = make([]models.UserCompact, 0)
	if err := u.db.Select(&users, u.q.GetUsersCompact, pq.Array([]string{models.UserTypeAgent})); err != nil {
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", u.i18n.P("globals.terms.user")), nil)
	}
	return users, nil
}

// CreateAgent creates a new agent user.
func (u *Manager) CreateAgent(firstName, lastName, email string, roles []string) (models.User, error) {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return models.User{}, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil)
	}

	var id = 0
	avatarURL := null.String{}
	email = strings.TrimSpace(strings.ToLower(email))
	if err := u.q.InsertAgent.QueryRow(email, firstName, lastName, password, avatarURL, pq.Array(roles)).Scan(&id); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return models.User{}, envelope.NewError(envelope.GeneralError, u.i18n.T("user.sameEmailAlreadyExists"), nil)
		}
		u.lo.Error("error creating user", "error", err)
		return models.User{}, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil)
	}
	return u.Get(id, "", models.UserTypeAgent)
}

// UpdateAgent updates an agent with individual field parameters
func (u *Manager) UpdateAgent(id int, firstName, lastName, email string, roles []string, enabled bool, availabilityStatus, newPassword string) error {
	var (
		hashedPassword any
		err            error
	)

	// Set password?
	if newPassword != "" {
		if !IsStrongPassword(newPassword) {
			return envelope.NewError(envelope.InputError, PasswordHint, nil)
		}
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			u.lo.Error("error generating bcrypt password", "error", err)
			return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
		}
		u.lo.Info("setting new password for user", "user_id", id)
	}

	// Update user in the database and clear cache.
	if _, err := u.q.UpdateAgent.Exec(id, firstName, lastName, email, pq.Array(roles), null.String{}, hashedPassword, enabled, availabilityStatus); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
	}
	u.InvalidateAgentCache(id)
	return nil
}

// SoftDeleteAgent soft deletes an agent by ID.
func (u *Manager) SoftDeleteAgent(id int) error {
	// Disallow if user is system user.
	systemUser, err := u.GetSystemUser()
	if err != nil {
		return err
	}
	if id == systemUser.ID {
		return envelope.NewError(envelope.InputError, u.i18n.T("user.cannotDeleteSystemUser"), nil)
	}
	if _, err := u.q.SoftDeleteAgent.Exec(id); err != nil {
		u.lo.Error("error deleting user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// markInactiveAgentsOffline sets agents offline if they have been inactive for more than 5 minutes.
func (u *Manager) markInactiveAgentsOffline() {
	if res, err := u.q.UpdateInactiveOffline.Exec(); err != nil {
		u.lo.Error("error setting users offline", "error", err)
	} else {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			u.lo.Info("set inactive users offline", "count", rows)
		}
	}
}

// GetAllAgents returns a list of all agents.
func (u *Manager) GetAgents() ([]models.UserCompact, error) {
	// Some dirty hack.
	return u.GetAllUsers(1, 999999999, models.UserTypeAgent, "desc", "users.updated_at", "")
}
