package sla

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	businesshours "github.com/abhinavxd/libredesk/internal/business_hours"
	bmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/sla/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/template"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

const (
	MetricFirstResponse = "first_response"
	MetricResolution    = "resolution"
	MetricNextResponse  = "next_response"

	NotificationTypeWarning = "warning"
	NotificationTypeBreach  = "breach"
)

var metricLabels = map[string]string{
	MetricFirstResponse: "First Response",
	MetricResolution:    "Resolution",
	MetricNextResponse:  "Next Response",
}

// Manager manages SLA policies and calculations.
type Manager struct {
	q                queries
	lo               *logf.Logger
	i18n             *i18n.I18n
	teamStore        teamStore
	userStore        userStore
	appSettingsStore appSettingsStore
	businessHrsStore businessHrsStore
	notifier         *notifier.Service
	template         *template.Manager
	wg               sync.WaitGroup
	opts             Opts
}

// Opts defines the options for creating SLA manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// Deadlines holds the deadlines for an SLA policy.
type Deadlines struct {
	FirstResponse time.Time
	Resolution    time.Time
	NextResponse  time.Time
}

// Breaches holds the breach timestamps for an SLA policy.
type Breaches struct {
	FirstResponse time.Time
	Resolution    time.Time
	NextResponse  time.Time
}

type teamStore interface {
	Get(id int) (tmodels.Team, error)
}

type userStore interface {
	GetAgent(int, string) (umodels.User, error)
}

type appSettingsStore interface {
	GetByPrefix(prefix string) (types.JSONText, error)
}

type businessHrsStore interface {
	Get(id int) (bmodels.BusinessHours, error)
}

// queries hold prepared SQL queries.
type queries struct {
	// TODO: name queries better.
	GetSLA                             *sqlx.Stmt `query:"get-sla-policy"`
	GetAllSLA                          *sqlx.Stmt `query:"get-all-sla-policies"`
	GetAppliedSLA                      *sqlx.Stmt `query:"get-applied-sla"`
	GetSLAEvent                        *sqlx.Stmt `query:"get-sla-event"`
	GetScheduledSLANotifications       *sqlx.Stmt `query:"get-scheduled-sla-notifications"`
	GetLatestAppliedSLAForConversation *sqlx.Stmt `query:"get-latest-applied-sla-for-conversation"`
	InsertScheduledSLANotification     *sqlx.Stmt `query:"insert-scheduled-sla-notification"`
	InsertSLA                          *sqlx.Stmt `query:"insert-sla-policy"`
	InsertNextResponseSLAEvent         *sqlx.Stmt `query:"insert-next-response-sla-event"`
	DeleteSLA                          *sqlx.Stmt `query:"delete-sla-policy"`
	UpdateSLA                          *sqlx.Stmt `query:"update-sla-policy"`
	ApplySLA                           *sqlx.Stmt `query:"apply-sla"`
	GetPendingSLAs                     *sqlx.Stmt `query:"get-pending-slas"`
	UpdateBreach                       *sqlx.Stmt `query:"update-breach"`
	UpdateMet                          *sqlx.Stmt `query:"update-met"`
	SetConversationNextSLADeadline     *sqlx.Stmt `query:"set-conversation-sla-deadline"`
	GetPendingSLAEvents                *sqlx.Stmt `query:"get-pending-sla-events"`
	UpdateSLAStatus                    *sqlx.Stmt `query:"update-sla-status"`
	MarkNotificationProcessed          *sqlx.Stmt `query:"mark-notification-processed"`
	MarkSLAEventAsBreached             *sqlx.Stmt `query:"mark-sla-event-as-breached"`
	MarkSLAEventAsMet                  *sqlx.Stmt `query:"mark-sla-event-as-met"`
	SetLatestSLAEventMetAt             *sqlx.Stmt `query:"set-latest-sla-event-met-at"`
}

// New creates a new SLA manager.
func New(opts Opts, teamStore teamStore, appSettingsStore appSettingsStore, businessHrsStore businessHrsStore, notifier *notifier.Service, template *template.Manager, userStore userStore) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo, i18n: opts.I18n, teamStore: teamStore, appSettingsStore: appSettingsStore, businessHrsStore: businessHrsStore, notifier: notifier, template: template, userStore: userStore, opts: opts}, nil
}

// Get retrieves an SLA by ID.
func (m *Manager) Get(id int) (models.SLAPolicy, error) {
	var sla models.SLAPolicy
	if err := m.q.GetSLA.Get(&sla, id); err != nil {
		if err == sql.ErrNoRows {
			return sla, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.sla}"), nil)
		}
		m.lo.Error("error fetching SLA", "error", err)
		return sla, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.sla}"), nil)
	}
	return sla, nil
}

// GetAll fetches all SLA policies.
func (m *Manager) GetAll() ([]models.SLAPolicy, error) {
	var slas = make([]models.SLAPolicy, 0)
	if err := m.q.GetAllSLA.Select(&slas); err != nil {
		m.lo.Error("error fetching SLAs", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.terms.sla")), nil)
	}
	return slas, nil
}

// Create creates a new SLA policy.
func (m *Manager) Create(name, description string, firstResponseTime, resolutionTime, nextResponseTime string, notifications models.SlaNotifications) error {
	if _, err := m.q.InsertSLA.Exec(name, description, firstResponseTime, resolutionTime, nextResponseTime, notifications); err != nil {
		m.lo.Error("error inserting SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.sla}"), nil)
	}
	return nil
}

// Update updates a SLA policy.
func (m *Manager) Update(id int, name, description string, firstResponseTime, resolutionTime, nextResponseTime string, notifications models.SlaNotifications) error {
	if _, err := m.q.UpdateSLA.Exec(id, name, description, firstResponseTime, resolutionTime, nextResponseTime, notifications); err != nil {
		m.lo.Error("error updating SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.sla}"), nil)
	}
	return nil
}

// Delete deletes an SLA policy.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteSLA.Exec(id); err != nil {
		m.lo.Error("error deleting SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.sla}"), nil)
	}
	return nil
}

// GetDeadlines returns the deadline for a given start time, sla policy and assigned team.
func (m *Manager) GetDeadlines(startTime time.Time, slaPolicyID, assignedTeamID int) (Deadlines, error) {
	var deadlines Deadlines

	businessHrs, timezone, err := m.getBusinessHoursAndTimezone(assignedTeamID)
	if err != nil {
		return deadlines, err
	}

	m.lo.Info("calculating deadlines", "timezone", timezone, "business_hours_always_open", businessHrs.IsAlwaysOpen, "business_hours", businessHrs.Hours)

	sla, err := m.Get(slaPolicyID)
	if err != nil {
		return deadlines, err
	}

	// Helper function to calculate deadlines by parsing the duration string.
	calculateDeadline := func(durationStr string) (time.Time, error) {
		if durationStr == "" {
			return time.Time{}, nil
		}
		dur, err := time.ParseDuration(durationStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing SLA duration (%s): %v", durationStr, err)
		}
		deadline, err := m.CalculateDeadline(startTime, int(dur.Minutes()), businessHrs, timezone)
		if err != nil {
			return time.Time{}, err
		}
		return deadline, nil
	}

	if deadlines.FirstResponse, err = calculateDeadline(sla.FirstResponseTime); err != nil {
		return deadlines, err
	}
	if deadlines.Resolution, err = calculateDeadline(sla.ResolutionTime); err != nil {
		return deadlines, err
	}
	if deadlines.NextResponse, err = calculateDeadline(sla.NextResponseTime); err != nil {
		return deadlines, err
	}
	return deadlines, nil
}

// ApplySLA applies an SLA policy to a conversation by calculating and setting the deadlines.
func (m *Manager) ApplySLA(startTime time.Time, conversationID, assignedTeamID, slaPolicyID int) (models.SLAPolicy, error) {
	var sla models.SLAPolicy

	// Get deadlines for the SLA policy and assigned team.
	deadlines, err := m.GetDeadlines(startTime, slaPolicyID, assignedTeamID)
	if err != nil {
		return sla, err
	}
	// Next response is not set at this point, next response are stored in SLA events as there can be multiple entries for next response.
	deadlines.NextResponse = time.Time{}

	// Insert applied SLA entry.
	var appliedSLAID int
	if err := m.q.ApplySLA.QueryRowx(
		conversationID,
		slaPolicyID,
		deadlines.FirstResponse,
		deadlines.Resolution,
	).Scan(&appliedSLAID); err != nil {
		m.lo.Error("error applying SLA", "error", err)
		return sla, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorApplying", "name", "{globals.terms.sla}"), nil)
	}

	// Schedule SLA notifications if any exist. SLA breaches have not occurred yet, as this is the first time the SLA is being applied.
	// Therefore, only schedule notifications for the deadlines.
	sla, err = m.Get(slaPolicyID)
	if err != nil {
		return sla, err
	}
	m.createNotificationSchedule(sla.Notifications, appliedSLAID, null.Int{}, deadlines, Breaches{})

	return sla, nil
}

// CreateNextResponseSLAEvent creates a next response SLA event for a conversation.
func (m *Manager) CreateNextResponseSLAEvent(conversationID, assignedTeamID int) (time.Time, error) {
	// Fetch the latest applied SLA for the conversation.
	var appliedSLA models.AppliedSLA
	if err := m.q.GetLatestAppliedSLAForConversation.Get(&appliedSLA, conversationID); err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, fmt.Errorf("no applied SLA found for conversation: %d", conversationID)
		}
		m.lo.Error("error fetching latest applied SLA for conversation", "error", err)
		return time.Time{}, fmt.Errorf("fetching latest applied SLA for conversation: %w", err)
	}

	var slaPolicy models.SLAPolicy
	if err := m.q.GetSLA.Get(&slaPolicy, appliedSLA.SLAPolicyID); err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, fmt.Errorf("SLA policy not found: %d", appliedSLA.SLAPolicyID)
		}
		m.lo.Error("error fetching SLA policy", "error", err)
		return time.Time{}, fmt.Errorf("fetching SLA policy: %w", err)
	}

	if slaPolicy.NextResponseTime == "" {
		m.lo.Info("no next response time set for SLA policy, skipping event creation",
			"conversation_id", conversationID,
			"policy_id", appliedSLA.SLAPolicyID,
			"applied_sla_id", appliedSLA.ID,
		)
		return time.Time{}, fmt.Errorf("no next response time set for SLA policy: %d, applied_sla: %d", appliedSLA.SLAPolicyID, appliedSLA.ID)
	}

	// Calculate the deadline for the next response SLA event.
	deadlines, err := m.GetDeadlines(time.Now(), slaPolicy.ID, assignedTeamID)
	if err != nil {
		m.lo.Error("error calculating deadlines for next response SLA event", "error", err)
		return time.Time{}, fmt.Errorf("calculating deadlines for next response SLA event: %w", err)
	}

	if deadlines.NextResponse.IsZero() {
		m.lo.Info("next response deadline is zero, skipping event creation",
			"conversation_id", conversationID,
			"policy_id", appliedSLA.SLAPolicyID,
			"applied_sla_id", appliedSLA.ID,
		)
		return time.Time{}, fmt.Errorf("next response deadline is zero for conversation: %d, policy: %d, applied_sla: %d", conversationID, appliedSLA.SLAPolicyID, appliedSLA.ID)
	}

	var slaEventID int
	if err := m.q.InsertNextResponseSLAEvent.QueryRow(appliedSLA.ID, appliedSLA.SLAPolicyID, deadlines.NextResponse).Scan(&slaEventID); err != nil {
		if err == sql.ErrNoRows {
			m.lo.Debug("unmet SLA event for next response already exists, skipping creation",
				"conversation_id", conversationID,
				"policy_id", slaPolicy.ID,
				"applied_sla_id", appliedSLA.ID,
			)
			return time.Time{}, fmt.Errorf("unmet next response SLA event already exists for conversation: %d, policy: %d, applied_sla: %d", conversationID, slaPolicy.ID, appliedSLA.ID)
		}
		m.lo.Error("error inserting SLA event",
			"error", err,
			"conversation_id", conversationID,
			"applied_sla_id", appliedSLA.ID,
		)
		return time.Time{}, fmt.Errorf("inserting SLA event (applied_sla: %d): %w", appliedSLA.ID, err)
	}

	// Update next SLA deadline (SLA target) in the conversation.
	if _, err := m.q.SetConversationNextSLADeadline.Exec(conversationID, deadlines.NextResponse); err != nil {
		m.lo.Error("error updating conversation next SLA deadline",
			"error", err,
			"conversation_id", conversationID,
			"applied_sla_id", appliedSLA.ID,
		)
		return time.Time{}, fmt.Errorf("updating conversation next SLA deadline (applied_sla: %d): %w", appliedSLA.ID, err)
	}

	// Create notification schedule for the next response SLA event.
	deadlines.FirstResponse = time.Time{}
	deadlines.Resolution = time.Time{}
	m.createNotificationSchedule(slaPolicy.Notifications, appliedSLA.ID, null.IntFrom(slaEventID), deadlines, Breaches{})

	return deadlines.NextResponse, nil
}

// SetLatestSLAEventMetAt marks the latest SLA event as met for a given applied SLA.
func (m *Manager) SetLatestSLAEventMetAt(appliedSLAID int, metric string) (time.Time, error) {
	var metAt time.Time
	if err := m.q.SetLatestSLAEventMetAt.QueryRow(appliedSLAID, metric).Scan(&metAt); err != nil {
		if err == sql.ErrNoRows {
			m.lo.Warn("no SLA event found for applied SLA ID and metric to update met at", "applied_sla_id", appliedSLAID, "metric", metric)
			return metAt, fmt.Errorf("no SLA event found for applied SLA ID: %d and metric: %s to update met at", appliedSLAID, metric)
		}
		m.lo.Error("error marking SLA event as met", "error", err)
		return metAt, fmt.Errorf("marking SLA event as met: %w", err)
	}
	return metAt, nil
}

// evaluatePendingSLAEvents fetches pending SLA events and marks them as breached if the deadline has passed.
func (m *Manager) evaluatePendingSLAEvents(ctx context.Context) error {
	var slaEvents []models.SLAEvent
	if err := m.q.GetPendingSLAEvents.SelectContext(ctx, &slaEvents); err != nil {
		m.lo.Error("error fetching pending SLA events", "error", err)
		return fmt.Errorf("fetching pending SLA events: %w", err)
	}
	m.lo.Info("found SLA events that have breached", "count", len(slaEvents))

	// Cache for SLA policies.
	var slaPolicyCache = make(map[int]models.SLAPolicy)
	for _, event := range slaEvents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := m.q.GetSLAEvent.GetContext(ctx, &event, event.ID); err != nil {
			m.lo.Error("error fetching SLA event", "error", err)
			continue
		}

		if event.DeadlineAt.IsZero() {
			m.lo.Warn("SLA event deadline is zero, skipping marking as breached", "sla_event_id", event.ID)
			continue
		}

		// Met at after the deadline or current time is after the deadline - mark event breached.
		var hasBreached bool
		if (event.MetAt.Valid && event.MetAt.Time.After(event.DeadlineAt)) || (time.Now().After(event.DeadlineAt) && !event.MetAt.Valid) {
			hasBreached = true
			if _, err := m.q.MarkSLAEventAsBreached.Exec(event.ID); err != nil {
				m.lo.Error("error marking SLA event as breached", "error", err)
				continue
			}
		}

		// Met at before the deadline - mark event met.
		if event.MetAt.Valid && event.MetAt.Time.Before(event.DeadlineAt) {
			if _, err := m.q.MarkSLAEventAsMet.Exec(event.ID); err != nil {
				m.lo.Error("error marking SLA event as met", "error", err)
				continue
			}
		}

		// Schedule a breach notification if the event is not met at all.
		if !event.MetAt.Valid && hasBreached {
			// Check if the SLA policy is already cached.
			slaPolicy, ok := slaPolicyCache[event.SlaPolicyID]
			if !ok {
				var err error
				slaPolicy, err = m.Get(event.SlaPolicyID)
				if err != nil {
					m.lo.Error("error fetching SLA policy", "error", err)
					continue
				}
				slaPolicyCache[event.SlaPolicyID] = slaPolicy
			}
			m.createNotificationSchedule(slaPolicy.Notifications, event.AppliedSLAID, null.IntFrom(event.ID), Deadlines{}, Breaches{
				NextResponse: time.Now(),
			})
		}
	}
	return nil
}

// Start begins SLA and SLA event evaluation loops in separate goroutines.
func (m *Manager) Start(ctx context.Context, interval time.Duration) {
	m.wg.Add(2)
	go m.runSLAEvaluation(ctx, interval)
	go m.runSLAEventEvaluation(ctx, interval)
}

// runSLAEvaluation periodically evaluates pending SLAs.
func (m *Manager) runSLAEvaluation(ctx context.Context, interval time.Duration) {
	defer m.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.evaluatePendingSLAs(ctx); err != nil {
				m.lo.Error("error processing pending SLAs", "error", err)
			}
		}
	}
}

// runSLAEventEvaluation periodically evaluates pending SLA events.
func (m *Manager) runSLAEventEvaluation(ctx context.Context, interval time.Duration) {
	defer m.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.evaluatePendingSLAEvents(ctx); err != nil {
				m.lo.Error("error marking SLA events as breached", "error", err)
			}
		}
	}
}

// SendNotifications picks scheduled SLA notifications from the database and sends them to agents as emails.
func (m *Manager) SendNotifications(ctx context.Context) error {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var notifications []models.ScheduledSLANotification
			if err := m.q.GetScheduledSLANotifications.SelectContext(ctx, &notifications); err != nil {
				if err == ctx.Err() {
					return err
				}
				m.lo.Error("error fetching scheduled SLA notifications", "error", err)
			} else if len(notifications) > 0 {
				m.lo.Debug("found scheduled SLA notifications", "count", len(notifications))
				for _, notification := range notifications {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if err := m.SendNotification(notification); err != nil {
						m.lo.Error("error sending notification", "error", err)
					}
				}
				m.lo.Debug("sent SLA notifications", "count", len(notifications))
			}
			<-ticker.C
		}
	}
}

// SendNotification sends a SLA notification to agents.
func (m *Manager) SendNotification(scheduledNotification models.ScheduledSLANotification) error {
	var (
		appliedSLA models.AppliedSLA
		slaEvent   models.SLAEvent
	)
	if scheduledNotification.SlaEventID.Int != 0 {
		if err := m.q.GetSLAEvent.Get(&slaEvent, scheduledNotification.SlaEventID.Int); err != nil {
			m.lo.Error("error fetching SLA event", "error", err)
			return fmt.Errorf("fetching SLA event for notification: %w", err)
		}
	}
	if err := m.q.GetAppliedSLA.Get(&appliedSLA, scheduledNotification.AppliedSLAID); err != nil {
		m.lo.Error("error fetching applied SLA", "error", err)
		return fmt.Errorf("fetching applied SLA for notification: %w", err)
	}

	// If conversation is `Resolved` / `Closed`, mark the notification as processed and return.
	if appliedSLA.ConversationStatus == cmodels.StatusResolved || appliedSLA.ConversationStatus == cmodels.StatusClosed {
		m.lo.Info("marking sla notification as processed as the conversation is resolved/closed", "status", appliedSLA.ConversationStatus, "scheduled_notification_id", scheduledNotification.ID)
		if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
			m.lo.Error("error marking notification as processed", "error", err)
		}
		return nil
	}

	// Send to all recipients (agents).
	for _, recipientS := range scheduledNotification.Recipients {
		// Check if SLA is already met, if met mark notification as processed and return.
		switch scheduledNotification.Metric {
		case MetricFirstResponse:
			if appliedSLA.FirstResponseMetAt.Valid {
				m.lo.Info("skipping notification as first response is already met", "applied_sla_id", appliedSLA.ID)
				if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
					m.lo.Error("error marking notification as processed", "error", err)
				}
				continue
			}
		case MetricResolution:
			if appliedSLA.ResolutionMetAt.Valid {
				m.lo.Info("skipping notification as resolution is already met", "applied_sla_id", appliedSLA.ID)
				if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
					m.lo.Error("error marking notification as processed", "error", err)
				}
				continue
			}
		case MetricNextResponse:
			if slaEvent.ID == 0 {
				m.lo.Warn("next response SLA event not found", "scheduled_notification_id", scheduledNotification.ID)
				return fmt.Errorf("next response SLA event not found for notification: %d", scheduledNotification.ID)
			}
			if slaEvent.MetAt.Valid {
				m.lo.Info("skipping notification as next response is already met", "applied_sla_id", appliedSLA.ID)
				if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
					m.lo.Error("error marking notification as processed", "error", err)
				}
				continue
			}
		default:
			m.lo.Error("unknown metric type", "metric", scheduledNotification.Metric)
			continue
		}

		// Get recipient agent, recipient can be a specific agent or assigned user.
		recipientID, err := strconv.Atoi(recipientS)
		if recipientS == "assigned_user" {
			recipientID = appliedSLA.ConversationAssignedUserID.Int
		} else if err != nil {
			m.lo.Error("error parsing recipient ID", "error", err, "recipient_id", recipientS)
			continue
		}

		// Recipient not found?
		if recipientID == 0 {
			if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
				m.lo.Error("error marking notification as processed", "error", err)
			}
			continue
		}

		agent, err := m.userStore.GetAgent(recipientID, "")
		if err != nil {
			m.lo.Error("error fetching agent for SLA notification", "recipient_id", recipientID, "error", err)
			if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
				m.lo.Error("error marking notification as processed", "error", err)
			}
			continue
		}

		var (
			dueIn, overdueBy string
			tmpl             string
		)
		// Set the template based on the notification type.
		switch scheduledNotification.NotificationType {
		case NotificationTypeBreach:
			tmpl = template.TmplSLABreached
		case NotificationTypeWarning:
			tmpl = template.TmplSLABreachWarning
		default:
			m.lo.Error("unknown notification type", "notification_type", scheduledNotification.NotificationType)
			return fmt.Errorf("unknown notification type: %s", scheduledNotification.NotificationType)
		}

		// Set the dueIn and overdueBy values based on the metric.
		// These are relative to the current time as setting exact time would require agent's timezone.
		getFriendlyDuration := func(target time.Time) string {
			d := time.Until(target)
			if d < 0 {
				return stringutil.FormatDuration(-d, false)
			}
			return stringutil.FormatDuration(d, false)
		}

		switch scheduledNotification.Metric {
		case MetricFirstResponse:
			dueIn = getFriendlyDuration(appliedSLA.FirstResponseDeadlineAt)
			overdueBy = getFriendlyDuration(appliedSLA.FirstResponseBreachedAt.Time)
		case MetricResolution:
			dueIn = getFriendlyDuration(appliedSLA.ResolutionDeadlineAt)
			overdueBy = getFriendlyDuration(appliedSLA.ResolutionBreachedAt.Time)
		case MetricNextResponse:
			dueIn = getFriendlyDuration(slaEvent.DeadlineAt)
			overdueBy = getFriendlyDuration(slaEvent.BreachedAt.Time)
		default:
			m.lo.Error("unknown metric type", "metric", scheduledNotification.Metric)
			return fmt.Errorf("unknown metric type: %s", scheduledNotification.Metric)
		}

		// Set the metric label.
		var metricLabel string
		if label, ok := metricLabels[scheduledNotification.Metric]; ok {
			metricLabel = label
		}

		// Render the email template.
		content, subject, err := m.template.RenderStoredEmailTemplate(tmpl,
			map[string]any{
				"SLA": map[string]any{
					"DueIn":     dueIn,
					"OverdueBy": overdueBy,
					"Metric":    metricLabel,
				},
				"Conversation": map[string]any{
					"ReferenceNumber": appliedSLA.ConversationReferenceNumber,
					"Subject":         appliedSLA.ConversationSubject,
					"Priority":        "",
					"UUID":            appliedSLA.ConversationUUID,
				},
				"Agent": map[string]any{
					"FirstName": agent.FirstName,
					"LastName":  agent.LastName,
					"FullName":  agent.FullName(),
					"Email":     agent.Email,
				},
				"Recipient": map[string]any{
					"FirstName": agent.FirstName,
					"LastName":  agent.LastName,
					"FullName":  agent.FullName(),
					"Email":     agent.Email,
				},
			})

		if err != nil {
			m.lo.Error("error rendering email template", "template", template.TmplConversationAssigned, "scheduled_notification_id", scheduledNotification.ID, "error", err)
			continue
		}

		// Enqueue email notification.
		if err := m.notifier.Send(notifier.Message{
			RecipientEmails: []string{
				agent.Email.String,
			},
			Subject:  subject,
			Content:  content,
			Provider: notifier.ProviderEmail,
		}); err != nil {
			m.lo.Error("error sending email notification", "error", err)
		}

		// Mark the notification as processed.
		if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
			m.lo.Error("error marking notification as processed", "error", err)
		}
	}
	return nil
}

// Close closes the SLA evaluation loop by stopping the worker pool.
func (m *Manager) Close() error {
	m.wg.Wait()
	return nil
}

// getBusinessHoursAndTimezone returns the business hours ID and timezone for a team, falling back to app settings i.e. default helpdesk settings.
func (m *Manager) getBusinessHoursAndTimezone(assignedTeamID int) (bmodels.BusinessHours, string, error) {
	var (
		businessHrsID int
		timezone      string
		bh            bmodels.BusinessHours
	)

	// Fetch from team if assignedTeamID is provided.
	if assignedTeamID != 0 {
		team, err := m.teamStore.Get(assignedTeamID)
		if err == nil {
			businessHrsID = team.BusinessHoursID.Int
			timezone = team.Timezone
		}
	}

	// Else fetch from app settings, this is System default.
	if businessHrsID == 0 || timezone == "" {
		settingsJ, err := m.appSettingsStore.GetByPrefix("app")
		if err != nil {
			return bh, "", err
		}

		var out map[string]interface{}
		if err := json.Unmarshal([]byte(settingsJ), &out); err != nil {
			return bh, "", fmt.Errorf("parsing settings: %v", err)
		}

		businessHrsIDStr, _ := out["app.business_hours_id"].(string)
		businessHrsID, _ = strconv.Atoi(businessHrsIDStr)
		timezone, _ = out["app.timezone"].(string)
	}

	// If still not found, return error.
	if businessHrsID == 0 || timezone == "" {
		return bh, "", fmt.Errorf("business hours or timezone not configured")
	}

	bh, err := m.businessHrsStore.Get(businessHrsID)
	if err != nil {
		if err == businesshours.ErrBusinessHoursNotFound {
			m.lo.Warn("business hours not found", "team_id", assignedTeamID)
			return bh, "", fmt.Errorf("business hours not found")
		}
		m.lo.Error("error fetching business hours for SLA", "error", err)
		return bh, "", err
	}
	return bh, timezone, nil
}

// createNotificationSchedule creates a notification schedule in database for the applied SLA.
func (m *Manager) createNotificationSchedule(notifications models.SlaNotifications, appliedSLAID int, slaEventID null.Int, deadlines Deadlines, breaches Breaches) {
	scheduleNotification := func(sendAt time.Time, metric, notifType string, recipients []string) {
		// Make sure the sendAt time is in not too far in the past.
		if sendAt.Before(time.Now().Add(-5 * time.Minute)) {
			m.lo.Debug("skipping scheduling notification as it is in the past", "send_at", sendAt, "applied_sla_id", appliedSLAID, "metric", metric, "type", notifType)
			return
		}
		if _, err := m.q.InsertScheduledSLANotification.Exec(appliedSLAID, slaEventID, metric, notifType, pq.Array(recipients), sendAt); err != nil {
			m.lo.Error("error inserting scheduled SLA notification", "error", err)
		}
	}

	// Insert scheduled entries for each notification.
	for _, notif := range notifications {
		var (
			delayDur time.Duration
			err      error
		)

		// No delay for immediate notifications.
		if notif.TimeDelayType == "immediately" {
			delayDur = 0
		} else {
			delayDur, err = time.ParseDuration(notif.TimeDelay)
			if err != nil {
				m.lo.Error("error parsing sla notification delay", "error", err)
				continue
			}
		}

		if notif.Type == NotificationTypeWarning {
			if !deadlines.FirstResponse.IsZero() {
				scheduleNotification(deadlines.FirstResponse.Add(-delayDur), MetricFirstResponse, notif.Type, notif.Recipients)
			}
			if !deadlines.Resolution.IsZero() {
				scheduleNotification(deadlines.Resolution.Add(-delayDur), MetricResolution, notif.Type, notif.Recipients)
			}
			if !deadlines.NextResponse.IsZero() {
				scheduleNotification(deadlines.NextResponse.Add(-delayDur), MetricNextResponse, notif.Type, notif.Recipients)
			}
		} else if notif.Type == NotificationTypeBreach {
			if !breaches.FirstResponse.IsZero() {
				scheduleNotification(breaches.FirstResponse.Add(delayDur), MetricFirstResponse, notif.Type, notif.Recipients)
			}
			if !breaches.Resolution.IsZero() {
				scheduleNotification(breaches.Resolution.Add(delayDur), MetricResolution, notif.Type, notif.Recipients)
			}
			if !breaches.NextResponse.IsZero() {
				scheduleNotification(breaches.NextResponse.Add(delayDur), MetricNextResponse, notif.Type, notif.Recipients)
			}
		}
	}
}

// evaluatePendingSLAs fetches pending SLAs and evaluates them, pending SLAs are applied SLAs that have not breached or met yet.
func (m *Manager) evaluatePendingSLAs(ctx context.Context) error {
	var pendingSLAs []models.AppliedSLA
	if err := m.q.GetPendingSLAs.SelectContext(ctx, &pendingSLAs); err != nil {
		m.lo.Error("error fetching pending SLAs", "error", err)
		return err
	}
	m.lo.Info("evaluating pending SLAs", "count", len(pendingSLAs))
	for _, sla := range pendingSLAs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := m.evaluateSLA(sla); err != nil {
				m.lo.Error("error evaluating SLA", "error", err)
			}
		}
	}
	m.lo.Info("evaluated pending SLAs", "count", len(pendingSLAs))
	return nil
}

// evaluateSLA evaluates an SLA policy on an applied SLA.
func (m *Manager) evaluateSLA(sla models.AppliedSLA) error {
	m.lo.Debug("evaluating SLA", "conversation_id", sla.ConversationID, "applied_sla_id", sla.ID)
	checkDeadline := func(deadline time.Time, metAt null.Time, metric string) error {
		if deadline.IsZero() {
			m.lo.Warn("deadline zero, skipping checking the deadline")
			return nil
		}

		now := time.Now()
		if !metAt.Valid && now.After(deadline) {
			m.lo.Debug("SLA breached as current time is after deadline", "deadline", deadline, "now", now, "metric", metric)
			if err := m.updateBreachAt(sla.ID, sla.SLAPolicyID, metric); err != nil {
				return fmt.Errorf("updating SLA breach timestamp: %w", err)
			}
			return nil
		}

		if metAt.Valid {
			if metAt.Time.After(deadline) {
				m.lo.Debug("SLA breached as met_at is after deadline", "deadline", deadline, "met_at", metAt.Time, "metric", metric)
				if err := m.updateBreachAt(sla.ID, sla.SLAPolicyID, metric); err != nil {
					return fmt.Errorf("updating SLA breach: %w", err)
				}
			} else {
				m.lo.Debug("SLA type met", "deadline", deadline, "met_at", metAt.Time, "metric", metric)
				if _, err := m.q.UpdateMet.Exec(sla.ID, metric); err != nil {
					return fmt.Errorf("updating SLA met: %w", err)
				}
			}
		}
		return nil
	}

	// If first response is not breached and not met, check the deadline and set them.
	if !sla.FirstResponseBreachedAt.Valid && !sla.FirstResponseMetAt.Valid {
		m.lo.Debug("checking deadline", "deadline", sla.FirstResponseDeadlineAt, "met_at", sla.ConversationFirstResponseAt.Time, "metric", MetricFirstResponse)
		if err := checkDeadline(sla.FirstResponseDeadlineAt, sla.ConversationFirstResponseAt, MetricFirstResponse); err != nil {
			return err
		}
	}

	// If resolution is not breached and not met, check the deadine and set them.
	if !sla.ResolutionBreachedAt.Valid && !sla.ResolutionMetAt.Valid {
		m.lo.Debug("checking deadline", "deadline", sla.ResolutionDeadlineAt, "met_at", sla.ConversationResolvedAt.Time, "metric", MetricResolution)
		if err := checkDeadline(sla.ResolutionDeadlineAt, sla.ConversationResolvedAt, MetricResolution); err != nil {
			return err
		}
	}

	// Update the conversation next SLA deadline.
	if _, err := m.q.SetConversationNextSLADeadline.Exec(sla.ConversationID, nil); err != nil {
		return fmt.Errorf("setting conversation next SLA deadline: %w", err)
	}

	// Update status of applied SLA.
	if _, err := m.q.UpdateSLAStatus.Exec(sla.ID); err != nil {
		return fmt.Errorf("updating applied SLA status: %w", err)
	}

	return nil
}

// updateBreachAt updates the breach timestamp for an SLA.
func (m *Manager) updateBreachAt(appliedSLAID, slaPolicyID int, metric string) error {
	if _, err := m.q.UpdateBreach.Exec(appliedSLAID, metric); err != nil {
		return err
	}

	// Schedule notification for the breach if there are any.
	sla, err := m.Get(slaPolicyID)
	if err != nil {
		m.lo.Error("error fetching SLA for scheduling breach notification", "error", err)
		return err
	}

	var firstResponse, resolution time.Time
	if metric == MetricFirstResponse {
		firstResponse = time.Now()
	} else if metric == MetricResolution {
		resolution = time.Now()
	}

	// Create notification schedule.
	m.createNotificationSchedule(sla.Notifications, appliedSLAID, null.Int{}, Deadlines{}, Breaches{
		FirstResponse: firstResponse,
		Resolution:    resolution,
	})

	return nil
}
