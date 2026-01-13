package notifier

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

//go:embed queries.sql
var queriesFS embed.FS

type UserNotificationManager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

type UserNotificationOpts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

type queries struct {
	GetNotifications       *sqlx.Stmt `query:"get-notifications"`
	GetNotificationStats   *sqlx.Stmt `query:"get-notification-stats"`
	InsertNotification     *sqlx.Stmt `query:"insert-notification"`
	MarkAsRead             *sqlx.Stmt `query:"mark-as-read"`
	MarkAllAsRead          *sqlx.Stmt `query:"mark-all-as-read"`
	DeleteNotification     *sqlx.Stmt `query:"delete-notification"`
	DeleteAllNotifications *sqlx.Stmt `query:"delete-all-notifications"`
	DeleteOldNotifications *sqlx.Stmt `query:"delete-old-notifications"`
}

// NewUserNotificationManager creates and returns a new instance of UserNotificationManager.
func NewUserNotificationManager(opts UserNotificationOpts) (*UserNotificationManager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, queriesFS); err != nil {
		return nil, err
	}
	return &UserNotificationManager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll retrieves notifications for a user with pagination.
func (m *UserNotificationManager) GetAll(userID, limit, offset int) ([]models.UserNotification, error) {
	var notifications = make([]models.UserNotification, 0)
	if err := m.q.GetNotifications.Select(&notifications, userID, limit, offset); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notifications, nil
		}
		m.lo.Error("error fetching notifications", "user_id", userID, "error", err)
		return notifications, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.notification}"), nil)
	}
	return notifications, nil
}

// GetStats retrieves notification statistics for a user.
func (m *UserNotificationManager) GetStats(userID int) (models.NotificationStats, error) {
	var stats models.NotificationStats
	if err := m.q.GetNotificationStats.Get(&stats, userID); err != nil {
		m.lo.Error("error fetching notification stats", "user_id", userID, "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.notification}"), nil)
	}
	return stats, nil
}

// Create creates a new notification for a user.
func (m *UserNotificationManager) Create(userID int, notificationType models.NotificationType, title string, body null.String, conversationID, messageID, actorID null.Int, meta json.RawMessage) (models.UserNotification, error) {
	var notification models.UserNotification
	if meta == nil {
		meta = json.RawMessage("{}")
	}
	if err := m.q.InsertNotification.Get(&notification, userID, notificationType, title, body, conversationID, messageID, actorID, meta); err != nil {
		m.lo.Error("error creating notification", "user_id", userID, "error", err)
		return notification, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.notification}"), nil)
	}
	return notification, nil
}

// MarkAsRead marks a notification as read.
func (m *UserNotificationManager) MarkAsRead(id, userID int) error {
	var returnedID int
	if err := m.q.MarkAsRead.Get(&returnedID, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.notification}"), nil)
		}
		m.lo.Error("error marking notification as read", "id", id, "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.notification}"), nil)
	}
	return nil
}

// MarkAllAsRead marks all notifications as read for a user.
func (m *UserNotificationManager) MarkAllAsRead(userID int) error {
	if _, err := m.q.MarkAllAsRead.Exec(userID); err != nil {
		m.lo.Error("error marking all notifications as read", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.notification}"), nil)
	}
	return nil
}

// Delete deletes a notification.
func (m *UserNotificationManager) Delete(id, userID int) error {
	if _, err := m.q.DeleteNotification.Exec(id, userID); err != nil {
		m.lo.Error("error deleting notification", "id", id, "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.notification}"), nil)
	}
	return nil
}

// DeleteAll deletes all notifications for a user.
func (m *UserNotificationManager) DeleteAll(userID int) error {
	if _, err := m.q.DeleteAllNotifications.Exec(userID); err != nil {
		m.lo.Error("error deleting all notifications", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.notification}"), nil)
	}
	return nil
}

// DeleteOldNotifications deletes notifications older than 30 days.
func (m *UserNotificationManager) DeleteOldNotifications(ctx context.Context) error {
	res, err := m.q.DeleteOldNotifications.ExecContext(ctx)
	if err != nil {
		m.lo.Error("error deleting old notifications", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.notification}"), nil)
	}
	rowsAffected, _ := res.RowsAffected()
	m.lo.Info("deleted old notifications", "rows_affected", rowsAffected)
	return nil
}

// RunNotificationCleaner runs a background job to delete old notifications every 24 hours.
func (m *UserNotificationManager) RunNotificationCleaner(ctx context.Context) {
	time.Sleep(10 * time.Second)
	if err := m.DeleteOldNotifications(ctx); err != nil {
		m.lo.Error("error cleaning old notifications", "error", err)
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.DeleteOldNotifications(ctx); err != nil {
				m.lo.Error("error cleaning old notifications", "error", err)
			}
		}
	}
}
