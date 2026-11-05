// Package auth implements OIDC multi-provider authentication and session management
package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/ssrf"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/knadh/go-i18n"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	sessredisstore "github.com/zerodha/simplesessions/stores/redis/v3"
	"github.com/zerodha/simplesessions/v3"

	"golang.org/x/oauth2"
)

const (
	defaultSessionLifetime = 9 * time.Hour
	sessionKeyPrefix       = "session:"
	userSessionKeyPrefix   = "user_sessions:"
)

var (
	ErrOIDCInvalidClient    = errors.New("oidc provider rejected client credentials")
	ErrOIDCEmailNotVerified = errors.New("oidc email is not verified")
)

// OIDCclaim holds OIDC token claims data
type OIDCclaim struct {
	Email         string `json:"email"`
	EmailVerified *bool  `json:"email_verified"`
	Sub           string `json:"sub"`
	Picture       string `json:"picture"`
}

// Provider defines an OIDC provider configuration
type Provider struct {
	ID           int
	Provider     string
	ProviderURL  string
	RedirectURL  func() (string, error)
	ClientID     string
	ClientSecret string
	VerifyEmail  bool
}

// Config holds OIDC providers and cookies security settings
type Config struct {
	Providers       []Provider
	SecureCookies   bool
	SessionLifetime time.Duration
}

type oidcProvider struct {
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	verifyEmail bool
}

// Auth is the auth service it manages OIDC authentication and sessions
type Auth struct {
	mu           sync.RWMutex
	cfg          Config
	i18n         *i18n.I18n
	oauthCfgs    map[int]oauth2.Config
	providers    map[int]oidcProvider
	redirectURLs map[int]func() (string, error)
	sess         *simplesessions.Manager
	logger       *logf.Logger
	rd           *redis.Client
	oidcClient   *http.Client
}

// New creates an Auth service with configured OIDC providers.
func New(cfg Config, i18n *i18n.I18n, rd *redis.Client, logger *logf.Logger, dialControl ssrf.Control) (*Auth, error) {
	oauthCfgs := make(map[int]oauth2.Config)
	providers := make(map[int]oidcProvider)
	redirectURLs := make(map[int]func() (string, error))

	oidcClient := newOIDCClient(dialControl)

	lifetime := cfg.SessionLifetime
	if lifetime <= 0 {
		lifetime = defaultSessionLifetime
	}
	cfg.SessionLifetime = lifetime

	sess := simplesessions.New(simplesessions.Options{
		EnableAutoCreate: false,
		SessionIDLength:  64,
		Cookie: simplesessions.CookieOptions{
			Name:       "libredesk_session",
			IsHTTPOnly: true,
			IsSecure:   cfg.SecureCookies,
			SameSite:   http.SameSiteLaxMode,
			MaxAge:     lifetime,
		},
	})

	st := sessredisstore.New(context.TODO(), rd)
	st.SetPrefix(sessionKeyPrefix)
	st.SetTTL(lifetime, false)
	sess.UseStore(st)
	sess.SetCookieHooks(simpleSessGetCookieCB, simpleSessSetCookieCB)

	a := &Auth{
		cfg:          cfg,
		i18n:         i18n,
		oauthCfgs:    oauthCfgs,
		providers:    providers,
		redirectURLs: redirectURLs,
		sess:         sess,
		logger:       logger,
		rd:           rd,
		oidcClient:   oidcClient,
	}
	// A provider that fails here stays in cfg.Providers and is initialized on first use.
	for _, provider := range cfg.Providers {
		a.initProvider(provider, oauthCfgs, providers, redirectURLs)
	}
	return a, nil
}

// TestProvider tests the OIDC provider url by doing a discovery on it.
func (a *Auth) TestProvider(url string) error {
	_, err := oidc.NewProvider(oidc.ClientContext(context.Background(), a.oidcClient), url)
	if err != nil {
		a.logger.Error("error testing oidc provider", "provider_url", url, "error", err)
		return envelope.NewError(envelope.GeneralError, err.Error(), nil)
	}
	return nil
}

// Reload reloads the auth configuration.
func (a *Auth) Reload(cfg Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Enforce stricter verification even when discovery prevents a full reload.
	for _, provider := range cfg.Providers {
		if current, ok := a.providers[provider.ID]; ok && provider.VerifyEmail {
			current.verifyEmail = true
			a.providers[provider.ID] = current
		}
		// a.cfg too, since a provider that is not initialized yet is initialized from it on first use.
		for i, p := range a.cfg.Providers {
			if p.ID == provider.ID && provider.VerifyEmail {
				a.cfg.Providers[i].VerifyEmail = true
			}
		}
	}

	oauthCfgs := make(map[int]oauth2.Config)
	providers := make(map[int]oidcProvider)
	redirectURLs := make(map[int]func() (string, error))

	for _, provider := range cfg.Providers {
		if err := a.initProvider(provider, oauthCfgs, providers, redirectURLs); err != nil {
			return envelope.NewError(envelope.GeneralError, err.Error(), nil)
		}
	}

	a.cfg.Providers = cfg.Providers
	a.oauthCfgs = oauthCfgs
	a.providers = providers
	a.redirectURLs = redirectURLs

	return nil
}

func (a *Auth) RemoveProvider(id int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.oauthCfgs, id)
	delete(a.providers, id)
	delete(a.redirectURLs, id)
	a.cfg.Providers = slices.DeleteFunc(a.cfg.Providers, func(p Provider) bool { return p.ID == id })
}

// initProvider runs OIDC discovery for the provider and adds it to the given maps.
func (a *Auth) initProvider(provider Provider, oauthCfgs map[int]oauth2.Config, providers map[int]oidcProvider, redirectURLs map[int]func() (string, error)) error {
	oauthCfg, oidcProv, err := a.buildProvider(provider)
	if err != nil {
		return err
	}
	oauthCfgs[provider.ID] = oauthCfg
	providers[provider.ID] = oidcProv
	redirectURLs[provider.ID] = provider.RedirectURL
	return nil
}

// buildProvider runs OIDC discovery for the provider; it does not touch the Auth state, so it can run without a.mu.
func (a *Auth) buildProvider(provider Provider) (oauth2.Config, oidcProvider, error) {
	oidcProv, err := oidc.NewProvider(oidc.ClientContext(context.Background(), a.oidcClient), provider.ProviderURL)
	if err != nil {
		a.logger.Error("error initializing oidc provider", "provider", provider.Provider, "provider_url", provider.ProviderURL, "error", err)
		return oauth2.Config{}, oidcProvider{}, err
	}

	oauthCfg := oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		Endpoint:     oidcProv.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := oidcProv.Verifier(&oidc.Config{ClientID: provider.ClientID})

	return oauthCfg, oidcProvider{provider: oidcProv, verifier: verifier, verifyEmail: provider.VerifyEmail}, nil
}

// configuredProvider returns the provider with the given id from a.cfg; callers hold a.mu.
func (a *Auth) configuredProvider(id int) (Provider, bool) {
	for _, provider := range a.cfg.Providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return Provider{}, false
}

// ensureProvider initializes a configured provider whose discovery failed earlier, e.g. because the identity provider started after Libredesk.
// Discovery runs without the lock, so an unreachable identity provider does not block other requests.
func (a *Auth) ensureProvider(id int) {
	a.mu.RLock()
	_, ok := a.providers[id]
	provider, configured := a.configuredProvider(id)
	a.mu.RUnlock()
	if ok || !configured {
		return
	}

	oauthCfg, oidcProv, err := a.buildProvider(provider)
	if err != nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	// Reload, RemoveProvider or another request may have run during discovery.
	current, configured := a.configuredProvider(id)
	if _, ok := a.providers[id]; ok || !configured {
		return
	}
	// A failed Reload may have tightened email verification in the meantime.
	oidcProv.verifyEmail = current.VerifyEmail
	a.oauthCfgs[id] = oauthCfg
	a.providers[id] = oidcProv
	a.redirectURLs[id] = current.RedirectURL
	a.logger.Info("initialized oidc provider on first use", "provider", current.Provider, "provider_id", id)
}

// LoginURL returns the login URL for the given provider.
func (a *Auth) LoginURL(providerID int, state string) (string, error) {
	a.ensureProvider(providerID)
	a.mu.RLock()
	defer a.mu.RUnlock()
	oauthCfg, ok := a.oauthCfgs[providerID]
	if !ok {
		return "", envelope.NewError(envelope.InputError, a.i18n.T("validation.notFoundProvider"), nil)
	}
	redirectURL, err := a.resolveRedirectURL(providerID)
	if err != nil {
		a.logger.Error("error resolving oidc redirect url", "provider_id", providerID, "error", err)
		return "", envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	oauthCfg.RedirectURL = redirectURL
	return oauthCfg.AuthCodeURL(state), nil
}
func (a *Auth) ExchangeOIDCToken(ctx context.Context, providerID int, code string) (string, OIDCclaim, error) {
	a.ensureProvider(providerID)
	a.mu.RLock()
	defer a.mu.RUnlock()

	oauthCfg, ok := a.oauthCfgs[providerID]
	if !ok {
		a.logger.Error("oidc provider not configured, it may have failed to initialize at startup", "provider_id", providerID)
		return "", OIDCclaim{}, fmt.Errorf("invalid provider ID: %d", providerID)
	}
	redirectURL, err := a.resolveRedirectURL(providerID)
	if err != nil {
		a.logger.Error("error resolving oidc redirect url", "provider_id", providerID, "error", err)
		return "", OIDCclaim{}, err
	}
	oauthCfg.RedirectURL = redirectURL

	provider, ok := a.providers[providerID]
	if !ok {
		a.logger.Error("oidc verifier not configured, it may have failed to initialize at startup", "provider_id", providerID)
		return "", OIDCclaim{}, fmt.Errorf("invalid provider ID: %d", providerID)
	}

	// Otherwise the token exchange and JWKS fetch use http.DefaultClient, which has no SSRF guard and no timeout.
	ctx = oidc.ClientContext(ctx, a.oidcClient)

	tk, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		a.logger.Error("error exchanging oidc authorization code for token, check the provider client ID / client secret (it may have expired) and redirect URL", "provider_id", providerID, "error", err)
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_client" {
			return "", OIDCclaim{}, ErrOIDCInvalidClient
		}
		return "", OIDCclaim{}, fmt.Errorf("error exchanging token: %v", err)
	}

	// Extract the ID Token from OAuth2 token.
	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		a.logger.Error("oidc token response has no id_token, check that the provider supports OIDC and the openid scope is allowed", "provider_id", providerID)
		return "", OIDCclaim{}, errors.New("id_token missing")
	}

	// Parse and verify ID Token payload.
	idTk, err := provider.verifier.Verify(ctx, rawIDTk)
	if err != nil {
		a.logger.Error("error verifying oidc id_token", "provider_id", providerID, "error", err)
		return "", OIDCclaim{}, fmt.Errorf("error verifying ID token: %v", err)
	}

	var claims OIDCclaim
	if err := idTk.Claims(&claims); err != nil {
		a.logger.Error("error parsing claims from oidc id_token", "provider_id", providerID, "error", err)
		return "", OIDCclaim{}, errors.New("error getting user from OIDC")
	}
	if strings.TrimSpace(claims.Email) == "" || (provider.verifyEmail && claims.EmailVerified == nil) {
		userInfo, err := provider.provider.UserInfo(ctx, oauth2.StaticTokenSource(tk))
		if err != nil {
			a.logger.Error("error fetching oidc user info", "provider_id", providerID, "error", err)
			return "", OIDCclaim{}, fmt.Errorf("error fetching OIDC user info: %w", err)
		}
		if userInfo.Subject == "" || userInfo.Subject != idTk.Subject {
			return "", OIDCclaim{}, errors.New("oidc user info subject does not match ID token")
		}
		// Verification from UserInfo must apply to the email returned by UserInfo.
		claims = OIDCclaim{}
		if err := userInfo.Claims(&claims); err != nil {
			return "", OIDCclaim{}, fmt.Errorf("error parsing OIDC user info: %w", err)
		}
	}
	if strings.TrimSpace(claims.Email) == "" {
		a.logger.Error("oidc id_token has no email, check that the provider sends the email claim", "provider_id", providerID)
		return "", OIDCclaim{}, errors.New("oidc id_token has no email")
	}
	if provider.verifyEmail && (claims.EmailVerified == nil || !*claims.EmailVerified) {
		a.logger.Warn("oidc login rejected because email is not verified", "provider_id", providerID)
		return "", OIDCclaim{}, ErrOIDCEmailNotVerified
	}
	a.logger.Debug("oidc token exchange successful", "provider_id", providerID, "email", claims.Email, "email_verified", claims.EmailVerified != nil && *claims.EmailVerified)
	return rawIDTk, claims, nil
}

// SaveSession creates and sets a session (post successful login/auth).
func (a *Auth) SaveSession(user amodels.User, r *fastglue.Request) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.NewSession(r, r)
	if err != nil {
		a.logger.Error("error creating login session", "user_id", user.ID, "error", err)
		return err
	}

	indexKey := userSessionKeyPrefix + strconv.Itoa(user.ID)
	p := a.rd.TxPipeline()
	p.SAdd(r.RequestCtx, indexKey, sess.ID())
	p.ExpireNX(r.RequestCtx, indexKey, a.cfg.SessionLifetime)
	p.ExpireGT(r.RequestCtx, indexKey, a.cfg.SessionLifetime)
	if _, err := p.Exec(r.RequestCtx); err != nil {
		a.logger.Error("error indexing login session", "user_id", user.ID, "error", err)
		return err
	}

	if err := sess.SetMulti(map[string]any{
		"id":         user.ID,
		"indexed":    true,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}); err != nil {
		a.logger.Error("error setting login session", "user_id", user.ID, "error", err)
		return err
	}
	return nil
}

// SetSessionValues preserves an existing session and creates one only when the browser has no valid session.
func (a *Auth) SetSessionValues(r *fastglue.Request, values map[string]any) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.Acquire(r.RequestCtx, r, r)
	if err == nil {
		_, err = sess.GetMulti("id")
	}
	if errors.Is(err, simplesessions.ErrInvalidSession) {
		sess, err = a.sess.NewSession(r, r)
	}
	if err != nil {
		a.logger.Error("error creating session", "error", err)
		return err
	}

	if err := sess.SetMulti(values); err != nil {
		a.logger.Error("error setting session values", "error", err)
		return err
	}
	return nil
}

// GetSessionValue returns the value for the given key from the session.
func (a *Auth) GetSessionValue(r *fastglue.Request, key string) (any, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.Acquire(r.RequestCtx, r, r)
	if err != nil {
		a.logger.Error("error acquiring session", "error", err)
		return "", err
	}

	val, err := sess.Get(key)
	if err != nil {
		a.logger.Error("error fetching session value", "error", err)
		return "", err
	}
	return val, nil
}

// SetCSRFCookie sets the CSRF token in the response cookie if not already set.
func (a *Auth) SetCSRFCookie(r *fastglue.Request) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	cookie := r.RequestCtx.Request.Header.Cookie("csrf_token")
	if cookie == nil {
		token, err := generateCSRFToken()
		if err != nil {
			return err
		}
		var csrfCookie fasthttp.Cookie
		csrfCookie.SetKey("csrf_token")
		csrfCookie.SetValue(token)
		csrfCookie.SetPath("/")
		csrfCookie.SetSecure(a.cfg.SecureCookies)
		csrfCookie.SetHTTPOnly(false)
		r.RequestCtx.Response.Header.SetCookie(&csrfCookie)
		return nil
	}
	return nil
}

// ValidateSession validates the session and returns the user.
func (a *Auth) ValidateSession(r *fastglue.Request) (models.User, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.Acquire(r.RequestCtx, r, r)
	if err != nil {
		if err != simplesessions.ErrInvalidSession {
			a.logger.Error("error acquiring session", "error", err)
		}
		return models.User{}, err
	}

	sessVals, err := sess.GetMulti("id", "email", "first_name", "last_name", "indexed")
	if err != nil {
		a.logger.Error("error fetching session variables", "error", err)
		return models.User{}, err
	}

	var (
		userID, _    = sess.Int(sessVals["id"], nil)
		email, _     = sess.String(sessVals["email"], nil)
		firstName, _ = sess.String(sessVals["first_name"], nil)
		lastName, _  = sess.String(sessVals["last_name"], nil)
	)

	if userID > 0 {
		indexed, err := sess.Bool(sessVals["indexed"], nil /** err **/)
		if err != nil || !indexed {
			return models.User{}, simplesessions.ErrInvalidSession
		}
	}

	return models.User{
		ID:        userID,
		Email:     null.NewString(email, email != ""),
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

// DestroySession destroys session
func (a *Auth) DestroySession(r *fastglue.Request) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.Acquire(r.RequestCtx, r, r)
	if err == simplesessions.ErrInvalidSession {
		return nil
	}
	if err != nil {
		a.logger.Error("error acquiring session", "error", err)
		return err
	}
	userID, err := sess.Int(sess.Get("id"))
	if err != nil && !errors.Is(err, simplesessions.ErrNil) && !errors.Is(err, simplesessions.ErrInvalidSession) {
		return err
	}
	if err := sess.Destroy(); err != nil {
		a.logger.Error("error clearing session", "error", err)
		return err
	}
	if userID > 0 {
		if err := a.rd.SRem(r.RequestCtx, userSessionKeyPrefix+strconv.Itoa(userID), sess.ID()).Err(); err != nil {
			a.logger.Error("error removing session index entry", "user_id", userID, "error", err)
			return err
		}
	}
	return nil
}

func (a *Auth) DestroyUserSessions(ctx context.Context, userID int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := DestroyUserSessions(ctx, a.rd, userID); err != nil {
		a.logger.Error("error destroying user sessions", "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// resolveRedirectURL reads the provider's redirect URL from the current root URL setting.
func (a *Auth) resolveRedirectURL(providerID int) (string, error) {
	fn, ok := a.redirectURLs[providerID]
	if !ok || fn == nil {
		return "", fmt.Errorf("no redirect URL resolver for provider: %d", providerID)
	}
	return fn()
}

func DestroyUserSessions(ctx context.Context, rd *redis.Client, userID int) error {
	indexKey := userSessionKeyPrefix + strconv.Itoa(userID)
	ids, err := rd.SMembers(ctx, indexKey).Result()
	if err != nil {
		return fmt.Errorf("fetching user sessions: %w", err)
	}
	keys := make([]string, 0, len(ids)+1)
	keys = append(keys, indexKey)
	for _, id := range ids {
		keys = append(keys, sessionKeyPrefix+id)
	}
	if err := rd.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("destroying user sessions: %w", err)
	}
	return nil
}

func newOIDCClient(dialControl ssrf.Control) *http.Client {
	transport := ssrf.NewTransport(dialControl, 3*time.Second)
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = 5 * time.Second
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

// generateCSRFToken creates a random base64 encoded str.
func generateCSRFToken() (string, error) {
	b, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(b)), nil
}

// getRequestCookie returns fashttp.Cookie for the given name.
func getRequestCookie(name string, r *fastglue.Request) (*fasthttp.Cookie, error) {
	val := r.RequestCtx.Request.Header.Cookie(name)
	if len(val) == 0 {
		return nil, nil
	}

	c := fasthttp.AcquireCookie()
	if err := c.ParseBytes(val); err != nil {
		return nil, err
	}

	return c, nil
}

func simpleSessGetCookieCB(name string, r any) (*http.Cookie, error) {
	req, ok := r.(*fastglue.Request)
	if !ok {
		return nil, errors.New("session callback doesn't have fastglue.Request")
	}

	// Create fast http cookie and parse it from cookie bytes.
	c, err := getRequestCookie(name, req)
	if c == nil {
		if err == nil {
			return nil, http.ErrNoCookie
		} else {
			return nil, err
		}

	}

	// Convert fasthttp cookie to net http cookie.
	return &http.Cookie{
		Name:     name,
		Value:    string(c.Value()),
		Path:     string(c.Path()),
		Domain:   string(c.Domain()),
		Expires:  c.Expire(),
		MaxAge:   c.MaxAge(),
		Secure:   c.Secure(),
		HttpOnly: c.HTTPOnly(),
		SameSite: http.SameSite(c.SameSite()),
	}, nil
}

func simpleSessSetCookieCB(c *http.Cookie, w any) error {
	req, ok := w.(*fastglue.Request)
	if !ok {
		return errors.New("session callback doesn't have fastglue.Request")
	}

	fc := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(fc)

	fc.SetKey(c.Name)
	fc.SetValue(c.Value)
	fc.SetPath(c.Path)
	fc.SetDomain(c.Domain)
	fc.SetExpire(c.Expires)
	fc.SetMaxAge(int(c.MaxAge))
	fc.SetSecure(c.Secure)
	fc.SetHTTPOnly(c.HttpOnly)
	fc.SetSameSite(fasthttp.CookieSameSite(c.SameSite))

	req.RequestCtx.Response.Header.SetCookie(fc)
	return nil
}
