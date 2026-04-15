// Package integration manages external integration configurations.
package integration

import (
	"database/sql"
	"embed"
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/integration/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// secretFields lists config keys whose values must be encrypted at rest.
var secretFields = map[string][]string{
	"shopify": {"access_token", "api_secret"},
}

// Manager handles integration CRUD.
type Manager struct {
	q             queries
	lo            *logf.Logger
	i18n          *i18n.I18n
	db            *sqlx.DB
	encryptionKey string
}

// Opts contains options for creating a Manager.
type Opts struct {
	DB            *sqlx.DB
	Lo            *logf.Logger
	I18n          *i18n.I18n
	EncryptionKey string
}

// queries contains prepared SQL statements.
type queries struct {
	GetAll  *sqlx.Stmt `query:"get-all-integrations"`
	Get     *sqlx.Stmt `query:"get-integration"`
	Upsert  *sqlx.Stmt `query:"upsert-integration"`
	Delete  *sqlx.Stmt `query:"delete-integration"`
	Toggle  *sqlx.Stmt `query:"toggle-integration"`
}

// New creates a new Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:             q,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		db:            opts.DB,
		encryptionKey: opts.EncryptionKey,
	}, nil
}

// GetAll returns all integrations with secrets decrypted.
func (m *Manager) GetAll() ([]models.Integration, error) {
	out := make([]models.Integration, 0)
	if err := m.q.GetAll.Select(&out); err != nil {
		m.lo.Error("error fetching integrations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "integrations"), nil)
	}
	for i := range out {
		m.decryptConfig(&out[i])
	}
	return out, nil
}

// Get returns a single integration by provider name.
func (m *Manager) Get(provider string) (models.Integration, error) {
	var out models.Integration
	if err := m.q.Get.Get(&out, provider); err != nil {
		if err == sql.ErrNoRows {
			return out, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "integration"), nil)
		}
		m.lo.Error("error fetching integration", "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "integration"), nil)
	}
	m.decryptConfig(&out)
	return out, nil
}

// Upsert creates or updates an integration.
func (m *Manager) Upsert(intg models.Integration) (models.Integration, error) {
	cfg, err := m.encryptConfig(intg.Provider, intg.Config)
	if err != nil {
		m.lo.Error("error encrypting integration config", "error", err)
		return models.Integration{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "integration"), nil)
	}

	var result models.Integration
	if err := m.q.Upsert.Get(&result, intg.Provider, cfg, intg.Enabled); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return models.Integration{}, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "integration"), nil)
		}
		m.lo.Error("error upserting integration", "error", err)
		return models.Integration{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "integration"), nil)
	}
	m.decryptConfig(&result)
	return result, nil
}

// Delete removes an integration by provider.
func (m *Manager) Delete(provider string) error {
	if _, err := m.q.Delete.Exec(provider); err != nil {
		m.lo.Error("error deleting integration", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "integration"), nil)
	}
	return nil
}

// Toggle flips the enabled flag on an integration.
func (m *Manager) Toggle(provider string) (models.Integration, error) {
	var result models.Integration
	if err := m.q.Toggle.Get(&result, provider); err != nil {
		if err == sql.ErrNoRows {
			return result, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "integration"), nil)
		}
		m.lo.Error("error toggling integration", "error", err)
		return models.Integration{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "integration"), nil)
	}
	m.decryptConfig(&result)
	return result, nil
}

// encryptConfig encrypts secret fields inside a config JSON blob.
func (m *Manager) encryptConfig(provider string, raw json.RawMessage) (json.RawMessage, error) {
	fields, ok := secretFields[provider]
	if !ok || len(fields) == 0 {
		return raw, nil
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return raw, err
	}

	for _, f := range fields {
		val, ok := cfg[f]
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok || s == "" || crypto.IsEncrypted(s) {
			continue
		}
		enc, err := crypto.Encrypt(s, m.encryptionKey)
		if err != nil {
			return nil, err
		}
		cfg[f] = enc
	}

	return json.Marshal(cfg)
}

// decryptConfig decrypts secret fields inside an integration's config.
func (m *Manager) decryptConfig(intg *models.Integration) {
	fields, ok := secretFields[intg.Provider]
	if !ok || len(fields) == 0 {
		return
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(intg.Config, &cfg); err != nil {
		return
	}

	changed := false
	for _, f := range fields {
		val, ok := cfg[f]
		if !ok {
			continue
		}
		s, ok := val.(string)
		if !ok || !crypto.IsEncrypted(s) {
			continue
		}
		dec, err := crypto.Decrypt(s, m.encryptionKey)
		if err != nil {
			m.lo.Error("error decrypting integration config field", "provider", intg.Provider, "field", f, "error", err)
			continue
		}
		cfg[f] = dec
		changed = true
	}

	if changed {
		if b, err := json.Marshal(cfg); err == nil {
			intg.Config = b
		}
	}
}
