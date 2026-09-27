package notifier

import (
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
	GetPreferences     *sqlx.Stmt `query:"get-notification-preferences"`
	GetPreferencesType *sqlx.Stmt `query:"get-notification-preferences-for-type"`
	UpsertPreference   *sqlx.Stmt `query:"upsert-notification-preference"`
}

type userChannelPreference struct {
	UserID  int                        `db:"user_id"`
	Channel models.NotificationChannel `db:"channel"`
	Enabled bool                       `db:"enabled"`
}

type PreferenceManager struct {
	db   *sqlx.DB
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
		db:   opts.DB,
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

func (m *PreferenceManager) EnabledChannels(userIDs []int, nType models.NotificationType) (map[int][]models.NotificationChannel, error) {
	var rows []userChannelPreference
	if err := m.q.GetPreferencesType.Select(&rows, pq.Array(userIDs), nType); err != nil {
		return nil, err
	}

	stored := make(map[[2]any]bool, len(rows))
	for _, r := range rows {
		stored[[2]any{r.UserID, r.Channel}] = r.Enabled
	}

	enabled := make(map[int][]models.NotificationChannel, len(userIDs))
	for _, userID := range userIDs {
		for _, channel := range models.NotificationChannels {
			on, ok := stored[[2]any{userID, channel}]
			if !ok {
				on = models.DefaultEnabled(nType)
			}
			if on {
				enabled[userID] = append(enabled[userID], channel)
			}
		}
	}
	return enabled, nil
}

// GetMatrix returns the effective preference for every agent notification type and channel.
func (m *PreferenceManager) GetMatrix(userID int) ([]models.NotificationPreference, error) {
	var rows []models.NotificationPreference
	if err := m.q.GetPreferences.Select(&rows, userID); err != nil {
		m.lo.Error("error fetching notification preferences", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	stored := make(map[[2]any]bool, len(rows))
	for _, r := range rows {
		stored[[2]any{r.NotificationType, r.Channel}] = r.Enabled
	}

	matrix := make([]models.NotificationPreference, 0, len(models.AgentNotificationTypes)*len(models.NotificationChannels))
	for _, nType := range models.AgentNotificationTypes {
		for _, channel := range models.NotificationChannels {
			on, ok := stored[[2]any{nType, channel}]
			if !ok {
				on = models.DefaultEnabled(nType)
			}
			matrix = append(matrix, models.NotificationPreference{
				NotificationType: nType,
				Channel:          channel,
				Enabled:          on,
			})
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
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting notification preference update", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	for _, p := range prefs {
		if _, err := tx.Stmtx(m.q.UpsertPreference).Exec(userID, p.NotificationType, p.Channel, p.Enabled); err != nil {
			m.lo.Error("error updating notification preference", "user_id", userID, "type", p.NotificationType, "channel", p.Channel, "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing notification preference update", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}
