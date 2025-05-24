// Package macro provides functionality for managing templated text responses and actions.
package macro

import (
	"database/sql"
	"embed"
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/macro/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager is the macro manager.
type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Predefined queries.
type queries struct {
	Get            *sqlx.Stmt `query:"get"`
	GetAll         *sqlx.Stmt `query:"get-all"`
	Create         *sqlx.Stmt `query:"create"`
	Update         *sqlx.Stmt `query:"update"`
	Delete         *sqlx.Stmt `query:"delete"`
	IncrUsageCount *sqlx.Stmt `query:"increment-usage-count"`
}

// Opts contains the dependencies for the macro manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// New initializes a macro manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs)
	if err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo, i18n: opts.I18n}, nil
}

// Get returns a macro by ID.
func (m *Manager) Get(id int) (models.Macro, error) {
	macro := models.Macro{}
	if err := m.q.Get.Get(&macro, id); err != nil {
		if err == sql.ErrNoRows {
			return macro, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.macro}"), nil)
		}
		m.lo.Error("error getting macro", "error", err)
		return macro, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.macro}"), nil)
	}
	return macro, nil
}

// Create adds a new macro.
func (m *Manager) Create(name, messageContent string, userID, teamID *int, visibility string, actions json.RawMessage) error {
	_, err := m.q.Create.Exec(name, messageContent, userID, teamID, visibility, actions)
	if err != nil {
		m.lo.Error("error creating macro", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.macro}"), nil)
	}
	return nil
}

// Update modifies an existing macro.
func (m *Manager) Update(id int, name, messageContent string, userID, teamID *int, visibility string, actions json.RawMessage) error {
	result, err := m.q.Update.Exec(id, name, messageContent, userID, teamID, visibility, actions)
	if err != nil {
		m.lo.Error("error updating macro", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.macro}"), nil)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.macro}"), nil)
	}
	return nil
}

// GetAll returns all macros.
func (m *Manager) GetAll() ([]models.Macro, error) {
	macros := make([]models.Macro, 0)
	err := m.q.GetAll.Select(&macros)
	if err != nil {
		m.lo.Error("error fetching macros", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.terms.macro")), nil)
	}
	return macros, nil
}

// Delete deletes a macro by ID.
func (m *Manager) Delete(id int) error {
	result, err := m.q.Delete.Exec(id)
	if err != nil {
		m.lo.Error("error deleting macro", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.macro}"), nil)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.macro}"), nil)
	}
	return nil
}

// IncrementUsageCount increments the usage count of a macro.
func (m *Manager) IncrementUsageCount(id int) error {
	if _, err := m.q.IncrUsageCount.Exec(id); err != nil {
		m.lo.Error("error incrementing usage count", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "macro usage count"), nil)
	}
	return nil
}
