package models

import (
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// providerLogos holds known provider logos.
var providerLogos = map[string]string{
	"Google": "/images/google-logo.png",
	"Custom": "",
}

// OIDC represents an OpenID Connect configuration.
type OIDC struct {
	ID              int       `db:"id" json:"id"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	Name            string    `db:"name" json:"name"`
	Enabled         bool      `db:"enabled" json:"enabled"`
	ClientID        string    `db:"client_id" json:"client_id,omitempty"`
	ClientSecret    string    `db:"client_secret" json:"client_secret,omitempty"`
	Provider        string    `db:"provider" json:"provider"`
	ProviderURL     string    `db:"provider_url" json:"provider_url"`
	RedirectURI     string    `db:"-" json:"redirect_uri"`
	ProviderLogoURL string    `db:"-" json:"logo_url"`
}

// SetProviderLogo provider logo to the OIDC model.
func (oidc *OIDC) SetProviderLogo() {
	for provider, logo := range providerLogos {
		if oidc.Provider == provider {
			oidc.ProviderLogoURL = logo
		}
	}
}

// ClearSecrets masks sensitive fields with dummy values for API responses.
func (oidc *OIDC) ClearSecrets() {
	if oidc.ClientSecret != "" {
		oidc.ClientSecret = strings.Repeat(stringutil.PasswordDummy, 10)
	}
}
