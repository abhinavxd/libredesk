package models

type OverviewSLA struct {
	FirstResponseMetCount         int     `json:"first_response_met_count" db:"first_response_met_count"`
	FirstResponseBreachedCount    int     `json:"first_response_breached_count" db:"first_response_breached_count"`
	AvgFirstResponseTimeSec       float64 `json:"avg_first_response_time_sec" db:"avg_first_response_time_sec"`
	NextResponseMetCount          int     `json:"next_response_met_count" db:"next_response_met_count"`
	NextResponseBreachedCount     int     `json:"next_response_breached_count" db:"next_response_breached_count"`
	AvgNextResponseTimeSec        float64 `json:"avg_next_response_time_sec" db:"avg_next_response_time_sec"`
	ResolutionMetCount            int     `json:"resolution_met_count" db:"resolution_met_count"`
	ResolutionBreachedCount       int     `json:"resolution_breached_count" db:"resolution_breached_count"`
	AvgResolutionTimeSec          float64 `json:"avg_resolution_time_sec" db:"avg_resolution_time_sec"`
	FirstResponseCompliancePercent float64 `json:"first_response_compliance_percent" db:"first_response_compliance_percent"`
	NextResponseCompliancePercent  float64 `json:"next_response_compliance_percent" db:"next_response_compliance_percent"`
	ResolutionCompliancePercent    float64 `json:"resolution_compliance_percent" db:"resolution_compliance_percent"`
}

type AgentReport struct {
	ID                   int     `db:"id" json:"id"`
	FirstName            string  `db:"first_name" json:"first_name"`
	LastName             string  `db:"last_name" json:"last_name"`
	TicketsAssigned      int     `db:"tickets_assigned" json:"tickets_assigned"`
	TicketsResolved      int     `db:"tickets_resolved" json:"tickets_resolved"`
	Replies              int     `db:"replies" json:"replies"`
	AvgFirstReplySeconds float64 `db:"avg_first_reply_seconds" json:"avg_first_reply_seconds"`
	CSATAvg              float64 `db:"csat_avg" json:"csat_avg"`
}

type TeamReport struct {
	ID                   int     `db:"id" json:"id"`
	Name                 string  `db:"name" json:"name"`
	TicketsAssigned      int     `db:"tickets_assigned" json:"tickets_assigned"`
	TicketsResolved      int     `db:"tickets_resolved" json:"tickets_resolved"`
	Replies              int     `db:"replies" json:"replies"`
	AvgFirstReplySeconds float64 `db:"avg_first_reply_seconds" json:"avg_first_reply_seconds"`
	CSATAvg              float64 `db:"csat_avg" json:"csat_avg"`
}
