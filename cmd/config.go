package main

import (
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

// handleGetConfig returns the public configuration needed for app initialization, this includes minimal app settings and enabled SSO providers (without secrets).
func handleGetConfig(r *fastglue.Request) error {
	var app = r.Context.(*App)

	// Get app settings
	settingsJSON, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unmarshal settings
	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		app.lo.Error("error unmarshalling settings", "err", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", app.i18n.T("globals.terms.setting")), nil))
	}

	// Filter to only include public fields needed for initial app load
	publicSettings := map[string]any{
		"app.lang":        settings["app.lang"],
		"app.favicon_url": settings["app.favicon_url"],
		"app.logo_url":    settings["app.logo_url"],
		"app.site_name":   settings["app.site_name"],
	}

	// Get all OIDC providers
	oidcProviders, err := app.oidc.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Filter for enabled providers and remove client_secret
	enabledProviders := make([]map[string]any, 0)
	for _, provider := range oidcProviders {
		if provider.Enabled {
			providerMap := map[string]any{
				"id":           provider.ID,
				"name":         provider.Name,
				"provider":     provider.Provider,
				"provider_url": provider.ProviderURL,
				"client_id":    provider.ClientID,
				"logo_url":     provider.ProviderLogoURL,
				"enabled":      provider.Enabled,
				"redirect_uri": provider.RedirectURI,
			}
			enabledProviders = append(enabledProviders, providerMap)
		}
	}

	// Add SSO providers to the response
	publicSettings["app.sso_providers"] = enabledProviders

	return r.SendEnvelope(publicSettings)
}
