package notifier

import (
	"database/sql"
	"errors"
	"slices"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

type prefQueries struct {
	GetPreferences   *sqlx.Stmt `query:"get-notification-preferences"`
	GetDisabled      *sqlx.Stmt `query:"get-disabled-notification-channels"`
	UpsertPreference *sqlx.Stmt `query:"upsert-notification-preference"`
}

type disabledChannel struct {
	UserID  int                        `db:"user_id"`
	Channel models.NotificationChannel `db:"channel"`
}

type PreferenceManager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    prefQueries
}

type PreferenceManagerOpts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// NewPreferenceManager creates a new PreferenceManager.
func NewPreferenceManager(opts PreferenceManagerOpts) (*PreferenceManager, error) {
	var q prefQueries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, queriesFS); err != nil {
		return nil, err
	}
	return &PreferenceManager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// DisabledChannels returns the channels each of the users has opted out of for the notification type.
func (m *PreferenceManager) DisabledChannels(userIDs []int, nType models.NotificationType) (map[int][]models.NotificationChannel, error) {
	var rows []disabledChannel
	if err := m.q.GetDisabled.Select(&rows, pq.Array(userIDs), nType); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching disabled notification channels", "type", nType, "error", err)
		return nil, err
	}
	disabled := make(map[int][]models.NotificationChannel, len(rows))
	for _, r := range rows {
		disabled[r.UserID] = append(disabled[r.UserID], r.Channel)
	}
	return disabled, nil
}

// GetMatrix returns the effective preference for every agent notification type and channel. Absent rows default to enabled.
func (m *PreferenceManager) GetMatrix(userID int) ([]models.NotificationPreference, error) {
	var stored []models.NotificationPreference
	if err := m.q.GetPreferences.Select(&stored, userID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching notification preferences", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	matrix := make([]models.NotificationPreference, 0, len(models.AgentNotificationTypes)*len(models.NotificationChannels))
	for _, nType := range models.AgentNotificationTypes {
		for _, channel := range models.NotificationChannels {
			pref := models.NotificationPreference{
				NotificationType: nType,
				Channel:          channel,
				Enabled:          true,
			}
			if i := slices.IndexFunc(stored, func(p models.NotificationPreference) bool {
				return p.NotificationType == nType && p.Channel == channel
			}); i >= 0 {
				pref.Enabled = stored[i].Enabled
			}
			matrix = append(matrix, pref)
		}
	}
	return matrix, nil
}

// Update upserts the given preferences for the user. Unknown types or channels are rejected.
func (m *PreferenceManager) Update(userID int, prefs []models.NotificationPreference) error {
	for _, p := range prefs {
		if !slices.Contains(models.AgentNotificationTypes, p.NotificationType) ||
			!slices.Contains(models.NotificationChannels, p.Channel) {
			return envelope.NewError(envelope.InputError, m.i18n.T("notification.invalidPreference"), nil)
		}
		if _, err := m.q.UpsertPreference.Exec(userID, p.NotificationType, p.Channel, p.Enabled); err != nil {
			m.lo.Error("error updating notification preference", "user_id", userID, "type", p.NotificationType, "channel", p.Channel, "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}
	return nil
}
