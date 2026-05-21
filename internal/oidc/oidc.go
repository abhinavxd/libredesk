package oidc

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs         embed.FS
	redirectURL = "/api/v1/oidc/%d/finish"
)

// Manager handles oidc-related operations.
type Manager struct {
	q             queries
	lo            *logf.Logger
	i18n          *i18n.I18n
	setting       settingsStore
	encryptionKey string
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB            *sqlx.DB
	Lo            *logf.Logger
	I18n          *i18n.I18n
	EncryptionKey string
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllOIDC *sqlx.Stmt `query:"get-all-oidc"`
	GetOIDC    *sqlx.Stmt `query:"get-oidc"`
	InsertOIDC *sqlx.Stmt `query:"insert-oidc"`
	UpdateOIDC *sqlx.Stmt `query:"update-oidc"`
	DeleteOIDC *sqlx.Stmt `query:"delete-oidc"`
}

type settingsStore interface {
	GetAppRootURL() (string, error)
}

// New creates and returns a new instance of the oidc Manager.
func New(opts Opts, setting settingsStore) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:             q,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		setting:       setting,
		encryptionKey: opts.EncryptionKey,
	}, nil
}

// Get returns an oidc by id.
func (o *Manager) Get(id int) (models.OIDC, error) {
	var oidc models.OIDC
	if err := o.q.GetOIDC.Get(&oidc, id); err != nil {
		if err == sql.ErrNoRows {
			return oidc, envelope.NewError(envelope.NotFoundError, o.i18n.T("validation.notFoundOidcProvider"), nil)
		}

		o.lo.Error("error fetching oidc", "error", err)
		return oidc, envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	o.decryptOIDC(&oidc)

	oidc.SetProviderLogo()
	rootURL, err := o.setting.GetAppRootURL()
	if err != nil {
		return models.OIDC{}, err
	}
	oidc.RedirectURI = fmt.Sprintf(rootURL+redirectURL, oidc.ID)
	return oidc, nil
}

// GetAll retrieves all oidc.
func (o *Manager) GetAll() ([]models.OIDC, error) {
	var oidc = make([]models.OIDC, 0)
	if err := o.q.GetAllOIDC.Select(&oidc); err != nil {
		o.lo.Error("error fetching oidc", "error", err)
		return oidc, envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Get root URL of the app.
	rootURL, err := o.setting.GetAppRootURL()
	if err != nil {
		return nil, err
	}

	o.decryptOIDCSlice(oidc)

	// Set logo and redirect URL for each record
	for i := range oidc {
		oidc[i].RedirectURI = fmt.Sprintf(rootURL+redirectURL, oidc[i].ID)
		oidc[i].SetProviderLogo()
	}
	return oidc, nil
}

// Create adds a new oidc.
func (o *Manager) Create(oidc models.OIDC) (models.OIDC, error) {
	// Encrypt sensitive fields before saving
	encryptedClientID, encryptedClientSecret, err := o.encryptOIDC(oidc.ClientID, oidc.ClientSecret)
	if err != nil {
		return models.OIDC{}, envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var createdOIDC models.OIDC
	if err := o.q.InsertOIDC.Get(&createdOIDC, oidc.Name, oidc.Provider, oidc.ProviderURL, encryptedClientID, encryptedClientSecret, oidc.LogoURL); err != nil {
		o.lo.Error("error inserting oidc", "error", err)
		return models.OIDC{}, envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	o.decryptOIDC(&createdOIDC)

	return createdOIDC, nil
}

// Update updates a oidc by id.
func (o *Manager) Update(id int, oidc models.OIDC) (models.OIDC, error) {
	current, err := o.Get(id)
	if err != nil {
		return models.OIDC{}, err
	}

	// If client secret is not provided, use the current one (already decrypted from Get)
	if oidc.ClientSecret == "" {
		oidc.ClientSecret = current.ClientSecret
	}

	// Encrypt sensitive fields before updating
	encryptedClientID, encryptedClientSecret, err := o.encryptOIDC(oidc.ClientID, oidc.ClientSecret)
	if err != nil {
		return models.OIDC{}, envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var updatedOIDC models.OIDC
	if err := o.q.UpdateOIDC.Get(&updatedOIDC, id, oidc.Name, oidc.Provider, oidc.ProviderURL, encryptedClientID, encryptedClientSecret, oidc.Enabled, oidc.LogoURL); err != nil {
		o.lo.Error("error updating oidc", "error", err)
		return models.OIDC{}, envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	o.decryptOIDC(&updatedOIDC)

	return updatedOIDC, nil
}

// Delete deletes a oidc by its id.
func (o *Manager) Delete(id int) error {
	if _, err := o.q.DeleteOIDC.Exec(id); err != nil {
		o.lo.Error("error deleting oidc", "error", err)
		return envelope.NewError(envelope.GeneralError, o.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// encryptOIDC encrypts sensitive OIDC fields (ClientID and ClientSecret).
// Returns the encrypted values and any error encountered.
func (o *Manager) encryptOIDC(clientID, clientSecret string) (encClientID, encClientSecret string, err error) {
	encClientID, err = crypto.Encrypt(clientID, o.encryptionKey)
	if err != nil {
		o.lo.Error("error encrypting client_id", "error", err)
		return "", "", err
	}

	encClientSecret, err = crypto.Encrypt(clientSecret, o.encryptionKey)
	if err != nil {
		o.lo.Error("error encrypting client_secret", "error", err)
		return "", "", err
	}

	return encClientID, encClientSecret, nil
}

// Decrypt failures clear the field so the app stays usable across encryption_key rotation.
func (o *Manager) decryptOIDC(oidc *models.OIDC) {
	if oidc.ClientID != "" {
		decrypted, err := crypto.Decrypt(oidc.ClientID, o.encryptionKey)
		if err != nil {
			o.lo.Error("error decrypting client_id, clearing field", "error", err, "oidc_id", oidc.ID)
			oidc.ClientID = ""
		} else {
			oidc.ClientID = decrypted
		}
	}

	if oidc.ClientSecret != "" {
		decrypted, err := crypto.Decrypt(oidc.ClientSecret, o.encryptionKey)
		if err != nil {
			o.lo.Error("error decrypting client_secret, clearing field", "error", err, "oidc_id", oidc.ID)
			oidc.ClientSecret = ""
		} else {
			oidc.ClientSecret = decrypted
		}
	}
}

func (o *Manager) decryptOIDCSlice(oidcs []models.OIDC) {
	for i := range oidcs {
		o.decryptOIDC(&oidcs[i])
	}
}
