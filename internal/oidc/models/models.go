package models

import (
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// providerLogos holds known provider logos.
var providerLogos = map[string]string{
	"Google":    "/images/google-logo.svg",
	"Microsoft": "/images/microsoft-logo.svg",
	"Custom":    "",
}

// OIDC represents an OpenID Connect configuration.
type OIDC struct {
	ID           int       `db:"id" json:"id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	Name         string    `db:"name" json:"name"`
	Enabled      bool      `db:"enabled" json:"enabled"`
	ClientID     string    `db:"client_id" json:"client_id,omitempty"`
	ClientSecret string    `db:"client_secret" json:"client_secret,omitempty"`
	Provider     string    `db:"provider" json:"provider"`
	ProviderURL  string    `db:"provider_url" json:"provider_url"`
	LogoURL      string    `db:"logo_url" json:"logo_url"`
	RedirectURI  string    `db:"-" json:"redirect_uri"`
}

// SetProviderLogo sets the logo URL if not already set.
// Falls back to built-in provider logos when no custom logo is provided.
func (oidc *OIDC) SetProviderLogo() {
	if oidc.LogoURL != "" {
		return
	}
	if logo, ok := providerLogos[oidc.Provider]; ok {
		oidc.LogoURL = logo
	}
}

// ClearSecrets masks sensitive fields with dummy values for API responses.
func (oidc *OIDC) ClearSecrets() {
	if oidc.ClientSecret != "" {
		oidc.ClientSecret = strings.Repeat(stringutil.PasswordDummy, 10)
	}
}
