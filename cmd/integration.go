package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/integration/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetIntegrations returns all integrations.
func handleGetIntegrations(r *fastglue.Request) error {
	app := r.Context.(*App)
	out, err := app.integration.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for i := range out {
		maskSecrets(&out[i])
	}
	return r.SendEnvelope(out)
}

// handleGetIntegration returns a single integration by provider.
func handleGetIntegration(r *fastglue.Request) error {
	app := r.Context.(*App)
	provider := r.RequestCtx.UserValue("provider").(string)
	if provider == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`provider`"), nil, envelope.InputError)
	}

	intg, err := app.integration.Get(provider)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	maskSecrets(&intg)
	return r.SendEnvelope(intg)
}

// handleCreateIntegration creates or updates an integration.
// It merges the incoming config with any existing config so that
// fields like access_token (set via OAuth) are not wiped out.
func handleCreateIntegration(r *fastglue.Request) error {
	app := r.Context.(*App)
	var intg imodels.Integration
	if err := r.Decode(&intg, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}
	if intg.Provider == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`provider`"), nil, envelope.InputError)
	}

	intg.Config = mergeWithExistingConfig(app, intg.Provider, intg.Config)

	result, err := app.integration.Upsert(intg)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	maskSecrets(&result)
	return r.SendEnvelope(result)
}

// handleUpdateIntegration updates an existing integration.
func handleUpdateIntegration(r *fastglue.Request) error {
	app := r.Context.(*App)
	provider := r.RequestCtx.UserValue("provider").(string)
	if provider == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`provider`"), nil, envelope.InputError)
	}

	var intg imodels.Integration
	if err := r.Decode(&intg, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}
	intg.Provider = provider
	intg.Config = mergeWithExistingConfig(app, provider, intg.Config)

	result, err := app.integration.Upsert(intg)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	maskSecrets(&result)
	return r.SendEnvelope(result)
}

// handleDeleteIntegration deletes an integration.
func handleDeleteIntegration(r *fastglue.Request) error {
	app := r.Context.(*App)
	provider := r.RequestCtx.UserValue("provider").(string)
	if provider == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`provider`"), nil, envelope.InputError)
	}
	if err := app.integration.Delete(provider); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleToggleIntegration toggles the enabled status.
func handleToggleIntegration(r *fastglue.Request) error {
	app := r.Context.(*App)
	provider := r.RequestCtx.UserValue("provider").(string)
	if provider == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`provider`"), nil, envelope.InputError)
	}
	result, err := app.integration.Toggle(provider)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	maskSecrets(&result)
	return r.SendEnvelope(result)
}

// handleTestIntegration tests the connection for a provider.
func handleTestIntegration(r *fastglue.Request) error {
	app := r.Context.(*App)
	provider := r.RequestCtx.UserValue("provider").(string)

	switch provider {
	case "shopify":
		return handleTestShopifyConnection(r, app)
	default:
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "unsupported integration provider", nil, envelope.InputError)
	}
}

// handleTestShopifyConnection tests the Shopify API credentials.
func handleTestShopifyConnection(r *fastglue.Request, app *App) error {
	intg, err := app.integration.Get("shopify")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	cfg, err := parseShopifyConfig(intg.Config)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid Shopify configuration", nil, envelope.InputError)
	}

	shopName, err := app.shopify.TestConnection(cfg.StoreURL, cfg.AccessToken)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, err.Error(), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]string{"shop_name": shopName})
}

// handleGetShopifyCustomer looks up a Shopify customer by email.
func handleGetShopifyCustomer(r *fastglue.Request) error {
	app := r.Context.(*App)
	email := string(r.RequestCtx.QueryArgs().Peek("email"))
	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	intg, err := app.integration.Get("shopify")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !intg.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Shopify integration is disabled", nil, envelope.NotFoundError)
	}

	cfg, err := parseShopifyConfig(intg.Config)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "invalid Shopify configuration", nil, envelope.GeneralError)
	}

	result, err := app.shopify.LookupCustomer(cfg.StoreURL, cfg.AccessToken, email)
	if err != nil {
		app.lo.Error("error looking up Shopify customer", "email", email, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "error contacting Shopify", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(result)
}

// handleShopifyOAuthAuthorize builds the Shopify OAuth authorize URL and returns it.
func handleShopifyOAuthAuthorize(r *fastglue.Request) error {
	app := r.Context.(*App)

	intg, err := app.integration.Get("shopify")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cfg, err := parseShopifyConfig(intg.Config)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid Shopify configuration", nil, envelope.InputError)
	}
	if cfg.StoreURL == "" || cfg.APIKey == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "store URL and API key are required before authorizing", nil, envelope.InputError)
	}

	rootURL, err := app.setting.GetAppRootURL()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "unable to determine app root URL", nil, envelope.GeneralError)
	}

	redirectURI := rootURL + "/api/v1/integrations/shopify/oauth/callback"
	scopes := "read_customers,read_orders"

	authURL := fmt.Sprintf("https://%s/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		cfg.StoreURL,
		url.QueryEscape(cfg.APIKey),
		url.QueryEscape(scopes),
		url.QueryEscape(redirectURI))

	return r.SendEnvelope(map[string]string{"authorize_url": authURL})
}

// handleShopifyOAuthCallback handles the redirect from Shopify after the merchant approves.
// This is a public endpoint (no auth) because Shopify redirects the browser here.
func handleShopifyOAuthCallback(r *fastglue.Request) error {
	app := r.Context.(*App)

	code := string(r.RequestCtx.QueryArgs().Peek("code"))
	shop := string(r.RequestCtx.QueryArgs().Peek("shop"))
	providedHMAC := string(r.RequestCtx.QueryArgs().Peek("hmac"))

	if code == "" || shop == "" {
		r.RequestCtx.SetStatusCode(fasthttp.StatusBadRequest)
		r.RequestCtx.SetBodyString("Missing code or shop parameter")
		return nil
	}

	intg, err := app.integration.Get("shopify")
	if err != nil {
		r.RequestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
		r.RequestCtx.SetBodyString("Shopify integration not configured")
		return nil
	}
	cfg, err := parseShopifyConfig(intg.Config)
	if err != nil || cfg.APISecret == "" {
		r.RequestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
		r.RequestCtx.SetBodyString("Invalid Shopify configuration")
		return nil
	}

	// Verify HMAC signature from Shopify.
	if providedHMAC != "" {
		if !verifyShopifyHMAC(r, cfg.APISecret, providedHMAC) {
			r.RequestCtx.SetStatusCode(fasthttp.StatusForbidden)
			r.RequestCtx.SetBodyString("HMAC verification failed")
			return nil
		}
	}

	// Exchange the code for a permanent access token.
	accessToken, err := app.shopify.ExchangeOAuthToken(cfg.StoreURL, cfg.APIKey, cfg.APISecret, code)
	if err != nil {
		app.lo.Error("error exchanging Shopify OAuth code", "error", err)
		r.RequestCtx.SetStatusCode(fasthttp.StatusBadGateway)
		r.RequestCtx.SetBodyString("Failed to exchange OAuth code: " + err.Error())
		return nil
	}

	// Update the integration with the access token.
	cfgMap := map[string]interface{}{
		"store_url":    cfg.StoreURL,
		"api_key":      cfg.APIKey,
		"api_secret":   cfg.APISecret,
		"access_token": accessToken,
	}
	cfgJSON, _ := json.Marshal(cfgMap)
	intg.Config = cfgJSON
	intg.Enabled = true
	if _, err := app.integration.Upsert(intg); err != nil {
		app.lo.Error("error saving Shopify access token", "error", err)
		r.RequestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
		r.RequestCtx.SetBodyString("Failed to save access token")
		return nil
	}

	// Redirect the admin back to the Shopify integration page.
	rootURL, _ := app.setting.GetAppRootURL()
	redirectTo := rootURL + "/admin/integrations/shopify?oauth=success"
	r.RequestCtx.Redirect(redirectTo, fasthttp.StatusFound)
	return nil
}

// verifyShopifyHMAC validates the HMAC signature from a Shopify OAuth callback.
func verifyShopifyHMAC(r *fastglue.Request, apiSecret, providedHMAC string) bool {
	args := r.RequestCtx.QueryArgs()
	params := make(map[string]string)
	args.VisitAll(func(key, value []byte) {
		k := string(key)
		if k != "hmac" {
			params[k] = string(value)
		}
	})

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	message := strings.Join(parts, "&")

	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(message))
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(computed), []byte(providedHMAC))
}

// shopifyConfig holds the parsed Shopify integration config.
type shopifyConfig struct {
	StoreURL    string `json:"store_url"`
	APIKey      string `json:"api_key"`
	APISecret   string `json:"api_secret"`
	AccessToken string `json:"access_token"`
}

func parseShopifyConfig(raw json.RawMessage) (shopifyConfig, error) {
	var cfg shopifyConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// maskSecrets replaces secret values in the config with dummy strings.
func maskSecrets(intg *imodels.Integration) {
	secretKeys := map[string][]string{
		"shopify": {"access_token", "api_secret"},
	}
	keys, ok := secretKeys[intg.Provider]
	if !ok {
		return
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(intg.Config, &cfg); err != nil {
		return
	}
	for _, k := range keys {
		if val, exists := cfg[k]; exists {
			if s, ok := val.(string); ok && s != "" {
				cfg[k] = strings.Repeat(stringutil.PasswordDummy, 10)
			}
		}
	}
	if b, err := json.Marshal(cfg); err == nil {
		intg.Config = b
	}
}

// mergeWithExistingConfig loads the current config from DB and merges new values on top,
// preserving fields (like access_token) that the caller didn't provide.
func mergeWithExistingConfig(app *App, provider string, incoming json.RawMessage) json.RawMessage {
	existing, err := app.integration.Get(provider)
	if err != nil {
		return incoming
	}

	var base map[string]interface{}
	if err := json.Unmarshal(existing.Config, &base); err != nil {
		return incoming
	}

	var overlay map[string]interface{}
	if err := json.Unmarshal(incoming, &overlay); err != nil {
		return incoming
	}

	for k, v := range overlay {
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		base[k] = v
	}

	merged, err := json.Marshal(base)
	if err != nil {
		return incoming
	}
	return merged
}
