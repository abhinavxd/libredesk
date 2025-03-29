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
	MetricsResolution   = "resolution"

	NotificationTypeWarning = "warning"
	NotificationTypeBreach  = "breach"
)

var metricLabels = map[string]string{
	MetricFirstResponse: "First Response",
	MetricsResolution:   "Resolution",
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
}

// Breaches holds the breach timestamps for an SLA policy.
type Breaches struct {
	FirstResponse time.Time
	Resolution    time.Time
}

type teamStore interface {
	Get(id int) (tmodels.Team, error)
}

type userStore interface {
	GetAgent(int) (umodels.User, error)
}

type appSettingsStore interface {
	GetByPrefix(prefix string) (types.JSONText, error)
}

type businessHrsStore interface {
	Get(id int) (bmodels.BusinessHours, error)
}

// queries hold prepared SQL queries.
type queries struct {
	GetSLA                         *sqlx.Stmt `query:"get-sla-policy"`
	GetAllSLA                      *sqlx.Stmt `query:"get-all-sla-policies"`
	GetAppliedSLA                  *sqlx.Stmt `query:"get-applied-sla"`
	GetScheduledSLANotifications   *sqlx.Stmt `query:"get-scheduled-sla-notifications"`
	InsertScheduledSLANotification *sqlx.Stmt `query:"insert-scheduled-sla-notification"`
	InsertSLA                      *sqlx.Stmt `query:"insert-sla-policy"`
	DeleteSLA                      *sqlx.Stmt `query:"delete-sla-policy"`
	UpdateSLA                      *sqlx.Stmt `query:"update-sla-policy"`
	ApplySLA                       *sqlx.Stmt `query:"apply-sla"`
	GetPendingSLAs                 *sqlx.Stmt `query:"get-pending-slas"`
	UpdateBreach                   *sqlx.Stmt `query:"update-breach"`
	UpdateMet                      *sqlx.Stmt `query:"update-met"`
	SetNextSLADeadline             *sqlx.Stmt `query:"set-next-sla-deadline"`
	UpdateSLAStatus                *sqlx.Stmt `query:"update-sla-status"`
	MarkNotificationProcessed      *sqlx.Stmt `query:"mark-notification-processed"`
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
			return sla, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.sla}"), nil)
		}
		m.lo.Error("error fetching SLA", "error", err)
		return sla, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.sla}"), nil)
	}
	return sla, nil
}

// GetAll fetches all SLA policies.
func (m *Manager) GetAll() ([]models.SLAPolicy, error) {
	var slas = make([]models.SLAPolicy, 0)
	if err := m.q.GetAllSLA.Select(&slas); err != nil {
		m.lo.Error("error fetching SLAs", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.entities.sla")), nil)
	}
	return slas, nil
}

// Create creates a new SLA policy.
func (m *Manager) Create(name, description string, firstResponseTime, resolutionTime string, notifications models.SlaNotifications) error {
	if _, err := m.q.InsertSLA.Exec(name, description, firstResponseTime, resolutionTime, notifications); err != nil {
		m.lo.Error("error inserting SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.sla}"), nil)
	}
	return nil
}

// Update updates a SLA policy.
func (m *Manager) Update(id int, name, description string, firstResponseTime, resolutionTime string, notifications models.SlaNotifications) error {
	if _, err := m.q.UpdateSLA.Exec(id, name, description, firstResponseTime, resolutionTime, notifications); err != nil {
		m.lo.Error("error updating SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.sla}"), nil)
	}
	return nil
}

// Delete deletes an SLA policy.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteSLA.Exec(id); err != nil {
		m.lo.Error("error deleting SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.sla}"), nil)
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
			return time.Time{}, fmt.Errorf("parsing SLA duration: %v", err)
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

	// Insert applied SLA entry.
	var appliedSLAID int
	if err := m.q.ApplySLA.QueryRowx(
		conversationID,
		slaPolicyID,
		deadlines.FirstResponse,
		deadlines.Resolution,
	).Scan(&appliedSLAID); err != nil {
		m.lo.Error("error applying SLA", "error", err)
		return sla, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorApplying", "name", "{globals.entities.sla}"), nil)
	}

	sla, err = m.Get(slaPolicyID)
	if err != nil {
		return sla, err
	}

	// Schedule SLA notifications if there are any, SLA breaches did not happen yet as this is the first time SLA is applied.
	// So, only schedule SLA breach warnings.
	m.createNotificationSchedule(sla.Notifications, appliedSLAID, deadlines, Breaches{})

	return sla, nil
}

// Run starts the SLA evaluation loop and evaluates pending SLAs.
func (m *Manager) Run(ctx context.Context, evalInterval time.Duration) {
	ticker := time.NewTicker(evalInterval)
	m.wg.Add(1)
	defer func() {
		m.wg.Done()
		ticker.Stop()
	}()

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

// SendNotifications picks scheduled SLA notifications from the database and sends them to agents as emails.
func (m *Manager) SendNotifications(ctx context.Context) error {
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
			} else {
				m.lo.Debug("found scheduled SLA notifications", "count", len(notifications))
				for _, notification := range notifications {
					// Exit early if context is done.
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if err := m.SendNotification(notification); err != nil {
							m.lo.Error("error sending notification", "error", err)
						}
					}
				}
				if len(notifications) > 0 {
					m.lo.Debug("sent SLA notifications", "count", len(notifications))
				}
			}

			// Sleep for short duration to avoid hammering the database.
			time.Sleep(30 * time.Second)
		}
	}
}

// SendNotification sends a SLA notification to agents.
func (m *Manager) SendNotification(scheduledNotification models.ScheduledSLANotification) error {
	var appliedSLA models.AppliedSLA
	if err := m.q.GetAppliedSLA.Get(&appliedSLA, scheduledNotification.AppliedSLAID); err != nil {
		m.lo.Error("error fetching applied SLA", "error", err)
		return fmt.Errorf("fetching applied SLA for notification: %w", err)
	}

	// Send to all recipients (agents).
	for _, recipientS := range scheduledNotification.Recipients {
		// Check if SLA is already met, if met for the metric, skip the notification and mark the notification as processed.
		switch scheduledNotification.Metric {
		case MetricFirstResponse:
			if appliedSLA.FirstResponseMetAt.Valid {
				m.lo.Debug("skipping notification as first response is already met", "applied_sla_id", appliedSLA.ID)
				if _, err := m.q.MarkNotificationProcessed.Exec(scheduledNotification.ID); err != nil {
					m.lo.Error("error marking notification as processed", "error", err)
				}
				continue
			}
		case MetricsResolution:
			if appliedSLA.ResolutionMetAt.Valid {
				m.lo.Debug("skipping notification as resolution is already met", "applied_sla_id", appliedSLA.ID)
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
		agent, err := m.userStore.GetAgent(recipientID)
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
				return "Overdue by " + stringutil.FormatDuration(-d, false)
			}
			return stringutil.FormatDuration(d, false)
		}

		switch scheduledNotification.Metric {
		case MetricFirstResponse:
			dueIn = getFriendlyDuration(appliedSLA.FirstResponseDeadlineAt)
			overdueBy = getFriendlyDuration(appliedSLA.FirstResponseBreachedAt.Time)
		case MetricsResolution:
			dueIn = getFriendlyDuration(appliedSLA.ResolutionDeadlineAt)
			overdueBy = getFriendlyDuration(appliedSLA.ResolutionBreachedAt.Time)
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

		// Set the notification as processed.
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
func (m *Manager) createNotificationSchedule(notifications models.SlaNotifications, appliedSLAID int, deadlines Deadlines, breaches Breaches) {
	scheduleNotification := func(sendAt time.Time, metric, notifType string, recipients []string) {
		if sendAt.Before(time.Now().Add(-5 * time.Minute)) {
			m.lo.Debug("skipping scheduling notification as it is in the past", "send_at", sendAt)
			return
		}

		if _, err := m.q.InsertScheduledSLANotification.Exec(appliedSLAID, metric, notifType, pq.Array(recipients), sendAt); err != nil {
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
				scheduleNotification(deadlines.Resolution.Add(-delayDur), MetricsResolution, notif.Type, notif.Recipients)
			}
		} else if notif.Type == NotificationTypeBreach {
			if !breaches.FirstResponse.IsZero() {
				scheduleNotification(breaches.FirstResponse.Add(delayDur), MetricFirstResponse, notif.Type, notif.Recipients)
			}
			if !breaches.Resolution.IsZero() {
				scheduleNotification(breaches.Resolution.Add(delayDur), MetricsResolution, notif.Type, notif.Recipients)
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
		m.lo.Debug("checking deadline", "deadline", sla.ResolutionDeadlineAt, "met_at", sla.ConversationResolvedAt.Time, "metric", MetricsResolution)
		if err := checkDeadline(sla.ResolutionDeadlineAt, sla.ConversationResolvedAt, MetricsResolution); err != nil {
			return err
		}
	}

	// Update the conversation next SLA deadline.
	if _, err := m.q.SetNextSLADeadline.Exec(sla.ConversationID); err != nil {
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
	} else if metric == MetricsResolution {
		resolution = time.Now()
	}

	// Create notification schedule.
	m.createNotificationSchedule(sla.Notifications, appliedSLAID, Deadlines{}, Breaches{
		FirstResponse: firstResponse,
		Resolution:    resolution,
	})

	return nil
}
