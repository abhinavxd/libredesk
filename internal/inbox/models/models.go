package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/volatiletech/null/v9"
)

// Inbox represents a inbox record in DB.
type Inbox struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	Name        string          `db:"name" json:"name"`
	Channel     string          `db:"channel" json:"channel"`
	Enabled     bool            `db:"enabled" json:"enabled"`
	CSATEnabled bool            `db:"csat_enabled" json:"csat_enabled"`
	From        string          `db:"from" json:"from"`
	Config      json.RawMessage `db:"config" json:"config"`
	Secret      null.String     `db:"secret" json:"secret,omitempty"`
}

// ClearPasswords masks all config passwords
func (m *Inbox) ClearPasswords() error {
	switch m.Channel {
	case "email":
		var cfg struct {
			IMAP []map[string]interface{} `json:"imap"`
			SMTP []map[string]interface{} `json:"smtp"`
		}

		if err := json.Unmarshal(m.Config, &cfg); err != nil {
			return err
		}

		dummyPassword := strings.Repeat(stringutil.PasswordDummy, 10)

		for i := range cfg.IMAP {
			cfg.IMAP[i]["password"] = dummyPassword
		}

		for i := range cfg.SMTP {
			cfg.SMTP[i]["password"] = dummyPassword
		}

		clearedConfig, err := json.Marshal(cfg)
		if err != nil {
			return err
		}

		m.Config = clearedConfig
	case "livechat":
		// Mask the secret field for livechat
		if m.Secret.Valid && m.Secret.String != "" {
			m.Secret = null.StringFrom(strings.Repeat(stringutil.PasswordDummy, 10))
		}
	default:
		return nil
	}

	return nil
}
