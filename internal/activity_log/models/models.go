package models

import (
	"time"
)

const (
	AgentLogin                  = "agent_login"
	AgentLogout                 = "agent_logout"
	AgentAway                   = "agent_away"
	AgentAwayReassigned         = "agent_away_reassigned"
	AgentOnline                 = "agent_online"
	AgentPasswordSet            = "agent_password_set"
	AgentRolePermissionsChanged = "agent_role_permissions_changed"
)

type ActivityLog struct {
	ID                  int64     `db:"id" json:"id"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
	ActivityType        string    `db:"activity_type" json:"activity_type"`
	ActivityDescription string    `db:"activity_description" json:"activity_description"`
	ActorID             int       `db:"actor_id" json:"actor_id"`
	TargetModelType     string    `db:"target_model_type" json:"target_model_type"`
	TargetModelID       int       `db:"target_model_id" json:"target_model_id"`
	IP                  string    `db:"ip" json:"ip"`

	Total int `db:"total" json:"-"`
}
