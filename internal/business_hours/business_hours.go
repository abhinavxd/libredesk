// Package businesshours handles the management of business hours and holidays.
package businesshours

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs                      embed.FS
	ErrBusinessHoursNotFound = errors.New("business hours not found")
)

// Manager manages business hours.
type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetBusinessHours    *sqlx.Stmt `query:"get-business-hours"`
	GetAllBusinessHours *sqlx.Stmt `query:"get-all-business-hours"`
	InsertBusinessHours *sqlx.Stmt `query:"insert-business-hours"`
	DeleteBusinessHours *sqlx.Stmt `query:"delete-business-hours"`
	UpdateBusinessHours *sqlx.Stmt `query:"update-business-hours"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// Get retrieves business hours.
func (m *Manager) Get(id int) (models.BusinessHours, error) {
	var businessHours models.BusinessHours
	if err := m.q.GetBusinessHours.Get(&businessHours, id); err != nil {
		if err == sql.ErrNoRows {
			return businessHours, ErrBusinessHoursNotFound
		}
		return businessHours, err
	}
	return businessHours, nil
}

// GetAll retrieves all business hours.
func (m *Manager) GetAll() ([]models.BusinessHours, error) {
	var hours = make([]models.BusinessHours, 0)
	if err := m.q.GetAllBusinessHours.Select(&hours); err != nil {
		m.lo.Error("error fetching business hours", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.businessHour}"), nil)
	}
	return hours, nil
}

// Create creates new business hours.
func (m *Manager) Create(name string, description null.String, isAlwaysOpen bool, workingHrs, holidays types.JSONText) error {
	if _, err := m.q.InsertBusinessHours.Exec(name, description, isAlwaysOpen, workingHrs, holidays); err != nil {
		m.lo.Error("error inserting business hours", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.businessHour}"), nil)
	}
	return nil
}

// Delete deletes business hours by ID.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteBusinessHours.Exec(id); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.businessHour}"), nil)
		}
		m.lo.Error("error deleting business hours", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.businessHour}"), nil)
	}
	return nil
}

// Update updates business hours by ID.
func (m *Manager) Update(id int, name string, description null.String, isAlwaysOpen bool, workingHrs, holidays types.JSONText) error {
	if _, err := m.q.UpdateBusinessHours.Exec(id, name, description, isAlwaysOpen, workingHrs, holidays); err != nil {
		m.lo.Error("error updating business hours", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.businessHour}"), nil)
	}
	return nil
}
