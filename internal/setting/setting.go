// Package setting handles the management of application settings.
package setting

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/abhinavxd/libredesk/internal/crypto"
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
	q               queries
	lo              *logf.Logger
	encryptionKey   string
	encryptedFields map[string]bool
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB            *sqlx.DB
	Lo            *logf.Logger
	EncryptionKey string
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

	// Fields that need encryption.
	encryptedFields := map[string]bool{
		"notification.email.password": true,
	}

	return &Manager{
		q:               q,
		lo:              opts.Lo,
		encryptionKey:   opts.EncryptionKey,
		encryptedFields: encryptedFields,
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

	// Decrypt sensitive fields.
	decryptedData, err := m.decryptSettings(b)
	if err != nil {
		m.lo.Error("error decrypting settings", "error", err)
		return out, err
	}

	if err := json.Unmarshal([]byte(decryptedData), &out); err != nil {
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

	// Decrypt sensitive fields.
	decryptedData, err := m.decryptSettings(b)
	if err != nil {
		m.lo.Error("error decrypting settings", "error", err)
		return b, err
	}

	return decryptedData, nil
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

	// Encrypt sensitive fields.
	encryptedData, err := m.encryptSettings(b)
	if err != nil {
		m.lo.Error("error encrypting settings", "error", err)
		return envelope.NewError(
			envelope.GeneralError,
			"Error encrypting settings",
			nil,
		)
	}

	// Update the settings in the DB.
	if _, err := m.q.Update.Exec(encryptedData); err != nil {
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

	// Decrypt sensitive fields.
	decryptedData, err := m.decryptSettings(b)
	if err != nil {
		m.lo.Error("error decrypting settings", "prefix", prefix, "error", err)
		return b, envelope.NewError(
			envelope.GeneralError,
			"Error decrypting settings",
			nil,
		)
	}

	return decryptedData, nil
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

	// Decrypt if this is an encrypted field
	if m.encryptedFields[key] {
		var valueStr string
		if err := json.Unmarshal(b, &valueStr); err == nil && valueStr != "" {
			decrypted, err := m.decryptIfNeeded(key, valueStr)
			if err != nil {
				return b, envelope.NewError(
					envelope.GeneralError,
					"Error decrypting setting",
					nil,
				)
			}
			b, err = json.Marshal(decrypted)
			if err != nil {
				return b, envelope.NewError(
					envelope.GeneralError,
					"Error marshalling decrypted setting",
					nil,
				)
			}
		}
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

// encryptSettings encrypts sensitive fields in the settings JSON.
func (m *Manager) encryptSettings(data []byte) ([]byte, error) {
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	for key := range settings {
		if valueStr, ok := settings[key].(string); ok && valueStr != "" {
			encrypted, err := m.encryptIfNeeded(key, valueStr)
			if err != nil {
				return nil, err
			}
			if encrypted != valueStr {
				settings[key] = encrypted
			}
		}
	}

	return json.Marshal(settings)
}

// decryptSettings decrypts sensitive fields in the settings JSON.
func (m *Manager) decryptSettings(data []byte) ([]byte, error) {
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	for key := range settings {
		if valueStr, ok := settings[key].(string); ok && valueStr != "" {
			decrypted, err := m.decryptIfNeeded(key, valueStr)
			if err != nil {
				m.lo.Error("error decrypting setting", "key", key, "error", err)
				continue
			}
			if decrypted != valueStr {
				settings[key] = decrypted
			}
		}
	}

	return json.Marshal(settings)
}

// encryptIfNeeded encrypts a value if the key requires encryption.
// Returns the encrypted value or the original value if encryption is not needed.
func (m *Manager) encryptIfNeeded(key, value string) (string, error) {
	if !m.encryptedFields[key] || value == "" {
		return value, nil
	}

	// Skip if already encrypted
	if crypto.IsEncrypted(value) {
		return value, nil
	}

	encrypted, err := crypto.Encrypt(value, m.encryptionKey)
	if err != nil {
		m.lo.Error("error encrypting setting", "key", key, "error", err)
		return "", err
	}

	return encrypted, nil
}

// decryptIfNeeded decrypts a value if the key requires decryption.
// Returns the decrypted value or the original value if decryption is not needed.
func (m *Manager) decryptIfNeeded(key, value string) (string, error) {
	if !m.encryptedFields[key] || value == "" {
		return value, nil
	}

	decrypted, err := crypto.Decrypt(value, m.encryptionKey)
	if err != nil {
		m.lo.Error("error decrypting setting", "key", key, "error", err)
		return "", err
	}

	return decrypted, nil
}
