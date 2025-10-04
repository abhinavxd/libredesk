// Package setting handles the management of application settings.
package setting

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles setting-related operations.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	Get         *sqlx.Stmt `query:"get"`
	GetAll      *sqlx.Stmt `query:"get-all"`
	Update      *sqlx.Stmt `query:"update"`
	GetByPrefix *sqlx.Stmt `query:"get-by-prefix"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

// GetAll retrieves all settings as a models.Settings struct.
func (m *Manager) GetAll() (models.Settings, error) {
	var (
		b   types.JSONText
		out models.Settings
	)

	if err := m.q.GetAll.Get(&b); err != nil {
		return out, err
	}

	if err := json.Unmarshal([]byte(b), &out); err != nil {
		return out, err
	}

	return out, nil
}

// GetAllJSON retrieves all settings as JSON.
func (m *Manager) GetAllJSON() (types.JSONText, error) {
	var b types.JSONText
	if err := m.q.GetAll.Get(&b); err != nil {
		m.lo.Error("error fetching settings", "error", err)
		return b, err
	}
	return b, nil
}

// Update updates settings with the passed values.
func (m *Manager) Update(s any) error {
	// Marshal settings.
	b, err := json.Marshal(s)
	if err != nil {
		m.lo.Error("error marshalling settings", "error", err)
		return envelope.NewError(
			envelope.GeneralError,
			"Error marshalling settings",
			nil,
		)
	}
	// Update the settings in the DB.
	if _, err := m.q.Update.Exec(b); err != nil {
		m.lo.Error("error updating settings", "error", err)
		return envelope.NewError(
			envelope.GeneralError,
			"Error updating settings",
			nil,
		)
	}
	return nil
}

// GetByPrefix retrieves all settings start with the given prefix.
func (m *Manager) GetByPrefix(prefix string) (types.JSONText, error) {
	var b types.JSONText
	if err := m.q.GetByPrefix.Get(&b, prefix+"%"); err != nil {
		m.lo.Error("error fetching settings", "prefix", prefix, "error", err)
		return b, envelope.NewError(
			envelope.GeneralError,
			"Error fetching settings",
			nil,
		)
	}
	return b, nil
}

// Get retrieves a setting by key as JSON.
func (m *Manager) Get(key string) (types.JSONText, error) {
	var b types.JSONText
	if err := m.q.Get.Get(&b, key); err != nil {
		m.lo.Error("error fetching setting", "key", key, "error", err)
		return b, envelope.NewError(
			envelope.GeneralError,
			"Error fetching settings",
			nil,
		)
	}
	return b, nil
}

// GetAppRootURL returns the root URL of the app.
func (m *Manager) GetAppRootURL() (string, error) {
	rootURL, err := m.Get("app.root_url")
	if err != nil {
		m.lo.Error("error fetching root URL", "error", err)
		return "", envelope.NewError(
			envelope.GeneralError,
			"Error fetching root URL",
			nil,
		)
	}
	return strings.Trim(string(rootURL), "\""), nil
}
