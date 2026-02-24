// Package customAttribute handles the management of custom attributes for contacts and conversations.
package customAttribute

import (
	"database/sql"
	"embed"

	"github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

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
	GetCustomAttribute     *sqlx.Stmt `query:"get-custom-attribute"`
	GetAllCustomAttributes *sqlx.Stmt `query:"get-all-custom-attributes"`
	InsertCustomAttribute  *sqlx.Stmt `query:"insert-custom-attribute"`
	DeleteCustomAttribute  *sqlx.Stmt `query:"delete-custom-attribute"`
	UpdateCustomAttribute  *sqlx.Stmt `query:"update-custom-attribute"`
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

// Get retrieves a custom attribute by ID.
func (m *Manager) Get(id int) (models.CustomAttribute, error) {
	var customAttribute models.CustomAttribute
	if err := m.q.GetCustomAttribute.Get(&customAttribute, id); err != nil {
		if err == sql.ErrNoRows {
			return customAttribute, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundCustomAttribute"), nil)
		}
		m.lo.Error("error fetching custom attribute", "error", err)
		return customAttribute, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return customAttribute, nil
}

// GetAll retrieves all custom attributes.
func (m *Manager) GetAll(appliesTo string) ([]models.CustomAttribute, error) {
	var customAttributes = make([]models.CustomAttribute, 0)
	if err := m.q.GetAllCustomAttributes.Select(&customAttributes, appliesTo); err != nil {
		m.lo.Error("error fetching custom attributes", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return customAttributes, nil
}

// Create creates a new custom attribute.
func (m *Manager) Create(attr models.CustomAttribute) (models.CustomAttribute, error) {
	var createdAttr models.CustomAttribute
	if err := m.q.InsertCustomAttribute.Get(&createdAttr, attr.AppliesTo, attr.Name, attr.Description, attr.Key, pq.Array(attr.Values), attr.DataType, attr.Regex, attr.RegexHint); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return models.CustomAttribute{}, envelope.NewError(envelope.InputError, m.i18n.T("errors.alreadyExistsCustomAttribute"), nil)
		}
		m.lo.Error("error inserting custom attribute", "error", err)
		return models.CustomAttribute{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return createdAttr, nil
}

// Update updates a custom attribute by ID.
func (m *Manager) Update(id int, attr models.CustomAttribute) (models.CustomAttribute, error) {
	var updatedAttr models.CustomAttribute
	if err := m.q.UpdateCustomAttribute.Get(&updatedAttr, id, attr.AppliesTo, attr.Name, attr.Description, pq.Array(attr.Values), attr.Regex, attr.RegexHint); err != nil {
		m.lo.Error("error updating custom attribute", "error", err)
		return models.CustomAttribute{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return updatedAttr, nil
}

// Delete deletes a custom attribute by ID.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteCustomAttribute.Exec(id); err != nil {
		m.lo.Error("error deleting custom attribute", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}
