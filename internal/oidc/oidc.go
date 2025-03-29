package oidc

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
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
	q       queries
	lo      *logf.Logger
	i18n    *i18n.I18n
	setting settingsStore
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllOIDC    *sqlx.Stmt `query:"get-all-oidc"`
	GetAllEnabled *sqlx.Stmt `query:"get-all-enabled"`
	GetOIDC       *sqlx.Stmt `query:"get-oidc"`
	InsertOIDC    *sqlx.Stmt `query:"insert-oidc"`
	UpdateOIDC    *sqlx.Stmt `query:"update-oidc"`
	DeleteOIDC    *sqlx.Stmt `query:"delete-oidc"`
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
		q:       q,
		lo:      opts.Lo,
		i18n:    opts.I18n,
		setting: setting,
	}, nil
}

// Get returns an oidc by id.
func (o *Manager) Get(id int, includeSecret bool) (models.OIDC, error) {
	var oidc models.OIDC
	if err := o.q.GetOIDC.Get(&oidc, id); err != nil {
		if err == sql.ErrNoRows {
			return oidc, envelope.NewError(envelope.NotFoundError, o.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.oidcProvider}"), nil)
		}

		o.lo.Error("error fetching oidc", "error", err)
		return oidc, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.oidcProvider}"), nil)
	}
	// Set logo and redirect URL.
	oidc.SetProviderLogo()
	rootURL, err := o.setting.GetAppRootURL()
	if err != nil {
		return models.OIDC{}, err
	}
	oidc.RedirectURI = fmt.Sprintf(rootURL+redirectURL, oidc.ID)
	// If secret is not to be included, replace it with dummy characters.
	if oidc.ClientSecret != "" && !includeSecret {
		oidc.ClientSecret = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	return oidc, nil
}

// GetAll retrieves all oidc.
func (o *Manager) GetAll() ([]models.OIDC, error) {
	var oidc = make([]models.OIDC, 0)
	if err := o.q.GetAllOIDC.Select(&oidc); err != nil {
		o.lo.Error("error fetching oidc", "error", err)
		return oidc, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.oidcProvider}"), nil)
	}

	// Get root URL of the app.
	rootURL, err := o.setting.GetAppRootURL()
	if err != nil {
		return nil, err
	}

	// Set logo and redirect URL.
	for i := range oidc {
		oidc[i].RedirectURI = fmt.Sprintf(rootURL+redirectURL, oidc[i].ID)
		oidc[i].SetProviderLogo()
	}
	return oidc, nil
}

// GetAllEnabled retrieves all enabled oidc.
func (o *Manager) GetAllEnabled() ([]models.OIDC, error) {
	var oidc = make([]models.OIDC, 0)
	if err := o.q.GetAllEnabled.Select(&oidc); err != nil {
		o.lo.Error("error fetching oidc", "error", err)
		return oidc, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.oidcProvider}"), nil)
	}
	for i := range oidc {
		oidc[i].SetProviderLogo()
	}
	return oidc, nil
}

// Create adds a new oidc.
func (o *Manager) Create(oidc models.OIDC) error {
	if _, err := o.q.InsertOIDC.Exec(oidc.Name, oidc.Provider, oidc.ProviderURL, oidc.ClientID, oidc.ClientSecret); err != nil {
		o.lo.Error("error inserting oidc", "error", err)
		return envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.oidcProvider}"), nil)
	}
	return nil
}

// Update updates a oidc by id.
func (o *Manager) Update(id int, oidc models.OIDC) error {
	current, err := o.Get(id, true)
	if err != nil {
		return err
	}
	if oidc.ClientSecret == "" {
		oidc.ClientSecret = current.ClientSecret
	}
	if _, err := o.q.UpdateOIDC.Exec(id, oidc.Name, oidc.Provider, oidc.ProviderURL, oidc.ClientID, oidc.ClientSecret, oidc.Enabled); err != nil {
		o.lo.Error("error updating oidc", "error", err)
		return envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.oidcProvider}"), nil)
	}
	return nil
}

// Delete deletes a oidc by its id.
func (o *Manager) Delete(id int) error {
	if _, err := o.q.DeleteOIDC.Exec(id); err != nil {
		o.lo.Error("error deleting oidc", "error", err)
		return envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.oidcProvider}"), nil)
	}
	return nil
}
