package contextlink

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	authmodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/context_link/models"
	convmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/leonelquinteros/gotext"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q             queries
	lo            *logf.Logger
	i18n          *gotext.Locale
	encryptionKey string
}

type Opts struct {
	DB            *sqlx.DB
	Lo            *logf.Logger
	I18n          *gotext.Locale
	EncryptionKey string
}

type queries struct {
	GetAllContextLinks    *sqlx.Stmt `query:"get-all-context-links"`
	GetContextLink        *sqlx.Stmt `query:"get-context-link"`
	GetContextLinkSecret  *sqlx.Stmt `query:"get-context-link-signing-secret"`
	GetActiveContextLinks *sqlx.Stmt `query:"get-active-context-links"`
	InsertContextLink     *sqlx.Stmt `query:"insert-context-link"`
	UpdateContextLink     *sqlx.Stmt `query:"update-context-link"`
	DeleteContextLink     *sqlx.Stmt `query:"delete-context-link"`
	ToggleContextLink     *sqlx.Stmt `query:"toggle-context-link"`
}

func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:             q,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		encryptionKey: opts.EncryptionKey,
	}, nil
}

func (m *Manager) GetAll() ([]models.ContextLink, error) {
	var links = make([]models.ContextLink, 0)
	if err := m.q.GetAllContextLinks.Select(&links); err != nil {
		m.lo.Error("error fetching context links", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	m.decryptLinks(links)
	return links, nil
}

func (m *Manager) Get(id int) (models.ContextLink, error) {
	var link models.ContextLink
	if err := m.q.GetContextLink.Get(&link, id); err != nil {
		if err == sql.ErrNoRows {
			return link, envelope.NewError(envelope.NotFoundError, m.i18n.Get("Not found"), nil)
		}
		m.lo.Error("error fetching context link", "error", err)
		return link, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	if err := m.decryptLink(&link); err != nil {
		return link, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	return link, nil
}

func (m *Manager) GetActive() ([]models.ContextLink, error) {
	var links = make([]models.ContextLink, 0)
	if err := m.q.GetActiveContextLinks.Select(&links); err != nil {
		m.lo.Error("error fetching active context links", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	return links, nil
}

func (m *Manager) Create(link models.ContextLink) (models.ContextLink, error) {
	var result models.ContextLink

	encryptedSecret, err := m.encryptSecret(link.Secret)
	if err != nil {
		return models.ContextLink{}, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}

	if err := m.q.InsertContextLink.Get(&result, link.Name, link.URLTemplate, encryptedSecret, link.TokenExpirySeconds, link.IsActive); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return models.ContextLink{}, envelope.NewError(envelope.ConflictError, m.i18n.Get("Already exists"), nil)
		}
		m.lo.Error("error inserting context link", "error", err)
		return models.ContextLink{}, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}

	if err := m.decryptLink(&result); err != nil {
		m.lo.Error("error decrypting context link secret after creation", "id", result.ID, "error", err)
	}
	return result, nil
}

func (m *Manager) Update(id int, link models.ContextLink) (models.ContextLink, error) {
	var result models.ContextLink

	encryptedSecret := link.Secret
	if strings.Contains(link.Secret, stringutil.PasswordDummy) {
		var existingSecret string
		if err := m.q.GetContextLinkSecret.Get(&existingSecret, id); err != nil {
			m.lo.Error("error fetching existing context link secret", "id", id, "error", err)
			return models.ContextLink{}, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
		}
		encryptedSecret = existingSecret
	} else if !crypto.IsEncrypted(link.Secret) {
		var err error
		encryptedSecret, err = m.encryptSecret(link.Secret)
		if err != nil {
			return models.ContextLink{}, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
		}
	}

	if err := m.q.UpdateContextLink.Get(&result, id, link.Name, link.URLTemplate, encryptedSecret, link.TokenExpirySeconds, link.IsActive); err != nil {
		m.lo.Error("error updating context link", "error", err)
		return models.ContextLink{}, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}

	if err := m.decryptLink(&result); err != nil {
		m.lo.Error("error decrypting context link secret after update", "id", result.ID, "error", err)
	}
	return result, nil
}

func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteContextLink.Exec(id); err != nil {
		m.lo.Error("error deleting context link", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	return nil
}

func (m *Manager) Toggle(id int) (models.ContextLink, error) {
	var result models.ContextLink
	if err := m.q.ToggleContextLink.Get(&result, id); err != nil {
		m.lo.Error("error toggling context link", "error", err)
		return models.ContextLink{}, envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
	}
	return result, nil
}

// GenerateURL builds the final URL for a context link by substituting template
// variables and optionally generating an AES-256-GCM encrypted token.
func (m *Manager) GenerateURL(link models.ContextLink, contact convmodels.ConversationContact, conversationUUID string, agent authmodels.User) (string, error) {
	u := link.URLTemplate

	// Substitute plain template variables with URL-encoded values.
	replacements := map[string]string{
		"{{email}}":              url.QueryEscape(contact.Email.String),
		"{{phone}}":              url.QueryEscape(contact.PhoneNumber.String),
		"{{phone_country_code}}": url.QueryEscape(contact.PhoneNumberCountryCode.String),
		"{{external_user_id}}":   url.QueryEscape(contact.ExternalUserID.String),
		"{{contact_id}}":         fmt.Sprintf("%d", contact.ID),
		"{{first_name}}":         url.QueryEscape(contact.FirstName),
		"{{last_name}}":          url.QueryEscape(contact.LastName),
		"{{conversation_uuid}}":  conversationUUID,
	}
	for placeholder, value := range replacements {
		u = strings.ReplaceAll(u, placeholder, value)
	}

	// Generate encrypted token if {{token}} is in the URL and secret is set.
	if strings.Contains(u, "{{token}}") && link.Secret != "" {
		secret, err := crypto.Decrypt(link.Secret, m.encryptionKey)
		if err != nil {
			m.lo.Error("error decrypting context link secret", "id", link.ID, "error", err)
			return "", envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
		}

		now := time.Now()
		payload := map[string]any{
			"email":              contact.Email.String,
			"phone":              contact.PhoneNumber.String,
			"phone_country_code": contact.PhoneNumberCountryCode.String,
			"external_user_id":   contact.ExternalUserID.String,
			"contact_id":         contact.ID,
			"first_name":         contact.FirstName,
			"last_name":          contact.LastName,
			"conversation_uuid":  conversationUUID,
			"agent_id":           agent.ID,
			"agent_email":        agent.Email,
			"iat":                now.Unix(),
			"exp":                now.Add(time.Duration(link.TokenExpirySeconds) * time.Second).Unix(),
		}

		plaintext, err := json.Marshal(payload)
		if err != nil {
			m.lo.Error("error marshalling token payload", "id", link.ID, "error", err)
			return "", envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
		}

		token, err := crypto.Encrypt(string(plaintext), secret)
		if err != nil {
			m.lo.Error("error encrypting token for context link", "id", link.ID, "error", err)
			return "", envelope.NewError(envelope.GeneralError, m.i18n.Get("Something went wrong"), nil)
		}
		// Strip "enc:" prefix. External systems receive just the base64 ciphertext.
		token = strings.TrimPrefix(token, crypto.EncryptedPrefix)
		u = strings.ReplaceAll(u, "{{token}}", url.QueryEscape(token))
	}

	return u, nil
}

func (m *Manager) encryptSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	encrypted, err := crypto.Encrypt(secret, m.encryptionKey)
	if err != nil {
		m.lo.Error("error encrypting context link secret", "error", err)
		return "", err
	}
	return encrypted, nil
}

func (m *Manager) decryptLink(link *models.ContextLink) error {
	decrypted, err := crypto.Decrypt(link.Secret, m.encryptionKey)
	if err != nil {
		m.lo.Error("error decrypting context link secret", "id", link.ID, "error", err)
		return err
	}
	link.Secret = decrypted
	return nil
}

func (m *Manager) decryptLinks(links []models.ContextLink) {
	for i := range links {
		if err := m.decryptLink(&links[i]); err != nil {
			continue
		}
	}
}
