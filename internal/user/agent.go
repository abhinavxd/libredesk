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

// MonitorUserAvailability sweeps inactive users to offline; cadence stays well below the 5-min threshold.
func (u *Manager) MonitorUserAvailability(ctx context.Context, onUsersOffline func([]models.OfflineUser)) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if users := u.MarkInactiveUsersOffline(); len(users) > 0 && onUsersOffline != nil {
				onUsersOffline(users)
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetAgent retrieves an agent by ID and caches it.
func (u *Manager) GetAgent(id int, email string) (models.User, error) {
	agent, err := u.Get(id, email, []string{models.UserTypeAgent})
	if err != nil {
		return models.User{}, err
	}

	u.agentCacheMu.Lock()
	u.agentCache[agent.ID] = cachedAgent{user: agent, expiresAt: time.Now().Add(agentCacheTTL)}
	u.agentCacheMu.Unlock()

	return agent, nil
}

// GetAgentFromCache returns a cached agent; expired entries are treated as misses.
func (u *Manager) GetAgentFromCache(id int) (models.User, bool) {
	u.agentCacheMu.RLock()
	defer u.agentCacheMu.RUnlock()
	c, exists := u.agentCache[id]
	if !exists || time.Now().After(c.expiresAt) {
		return models.User{}, false
	}
	return c.user, true
}

// GetAgentCachedOrLoad returns the cached agent or loads from DB.
func (u *Manager) GetAgentCachedOrLoad(id int) (models.User, error) {
	if agent, exists := u.GetAgentFromCache(id); exists {
		return agent, nil
	}
	return u.GetAgent(id, "")
}

// InvalidateAgentCache drops a single agent from the cache.
func (u *Manager) InvalidateAgentCache(id int) {
	u.lo.Debug("invalidating agent cache", "agent_id", id)
	u.agentCacheMu.Lock()
	defer u.agentCacheMu.Unlock()
	delete(u.agentCache, id)
}

// InvalidateAllAgentCache clears the entire agent cache.
func (u *Manager) InvalidateAllAgentCache() {
	u.agentCacheMu.Lock()
	defer u.agentCacheMu.Unlock()
	u.agentCache = make(map[int]cachedAgent)
}

// GetAgentsCompact returns a compact list of agents with limited fields.
func (u *Manager) GetAgentsCompact() ([]models.UserCompact, error) {
	var users = make([]models.UserCompact, 0)
	if err := u.db.Select(&users, u.q.GetUsersCompact, pq.Array([]string{models.UserTypeAgent})); err != nil {
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return users, nil
}

// CreateAgent creates a new agent user.
func (u *Manager) CreateAgent(firstName, lastName, email string, roles []string) (models.User, error) {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return models.User{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var id = 0
	avatarURL := null.String{}
	email = strings.TrimSpace(strings.ToLower(email))
	if err := u.q.InsertAgent.QueryRow(email, firstName, lastName, password, avatarURL, pq.Array(roles)).Scan(&id); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return models.User{}, envelope.NewError(envelope.GeneralError, u.i18n.T("user.sameEmailAlreadyExists"), nil)
		}
		u.lo.Error("error creating user", "error", err)
		return models.User{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return u.Get(id, "", []string{models.UserTypeAgent})
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
			return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		u.lo.Info("setting new password for user", "user_id", id)
	}

	// Update user in the database.
	if _, err := u.q.UpdateAgent.Exec(id, firstName, lastName, email, pq.Array(roles), null.String{}, hashedPassword, enabled, availabilityStatus); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.GeneralError, u.i18n.T("user.sameEmailAlreadyExists"), nil)
		}
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
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
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// MarkInactiveUsersOffline sets users offline if they have been inactive for more than 5 minutes.
func (u *Manager) MarkInactiveUsersOffline() []models.OfflineUser {
	var users []models.OfflineUser
	if err := u.q.UpdateInactiveOffline.Select(&users); err != nil {
		u.lo.Error("error setting users offline", "error", err)
		return nil
	}
	for _, user := range users {
		u.InvalidateAgentCache(user.ID)
	}
	if len(users) > 0 {
		u.lo.Info("set inactive users offline", "count", len(users))
	}
	return users
}

// GetAllAgents returns a list of all agents.
func (u *Manager) GetAgents() ([]models.UserCompact, error) {
	// Some dirty hack.
	return u.GetAllUsers(1, 999999999, []string{models.UserTypeAgent}, "desc", "users.updated_at", "", "")
}
