package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/zerodha/fastglue"
)

const (
	turnstileSiteverifyURL  = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	maxTurnstileKeyLength   = 200
	maxTurnstileTokenLength = 4096
	// turnstileRequiredCode is the error data field the widget uses to fetch a token and retry.
	// Matching the translated message would break when the locale changes.
	turnstileRequiredCode = "turnstile_required"
)

var (
	errTurnstileRejected    = errors.New("turnstile rejected token")
	errTurnstileUnavailable = errors.New("turnstile verification unavailable")

	turnstileHTTPClient = &http.Client{Timeout: 5 * time.Second}
)

// chatInitNeedsTurnstile reports whether this conversation init must present a Turnstile token.
// The check is off only when both keys are empty. A site key whose secret cannot be decrypted
// still requires the check, and that check fails closed. Signed-in users skip it. A visitor
// needs a fresh token for every new conversation. A reply that only continues one does not.
func chatInitNeedsTurnstile(siteKey, secret string, contactID int, isVisitor bool) bool {
	if strings.TrimSpace(siteKey) == "" && strings.TrimSpace(secret) == "" {
		return false
	}
	if contactID > 0 && !isVisitor {
		return false
	}
	return true
}

func validateTurnstileConfig(app *App, cfg livechat.Config) error {
	site := strings.TrimSpace(cfg.TurnstileSiteKey)
	secret := strings.TrimSpace(cfg.TurnstileSecretKey)
	if len(site) > maxTurnstileKeyLength || len(secret) > maxTurnstileKeyLength {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.turnstilePair"), nil)
	}
	// A masked secret means the saved value was not restored before validation.
	if (site == "") != (secret == "") || strings.Contains(secret, stringutil.PasswordDummy) {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.turnstilePair"), nil)
	}
	return nil
}

// enforceChatInitTurnstile returns nil when the visitor may continue. A failure is an
// envelope.Error and does not write the response. SendErrorEnvelope returns nil after a
// successful write, so writing here would let the caller treat a rejection as success.
func enforceChatInitTurnstile(r *fastglue.Request, app *App, config livechat.Config, token, clientIP string, contactID int, isVisitor bool, inboxID int) error {
	if !chatInitNeedsTurnstile(config.TurnstileSiteKey, config.TurnstileSecretKey, contactID, isVisitor) {
		return nil
	}
	secret := strings.TrimSpace(config.TurnstileSecretKey)
	var err error
	if secret == "" {
		app.lo.Warn("live chat turnstile secret is unusable", "inbox_id", inboxID)
		err = errTurnstileUnavailable
	} else {
		ctx := context.Background()
		if r != nil && r.RequestCtx != nil {
			ctx = r.RequestCtx
		}
		err = verifyTurnstileToken(ctx, turnstileHTTPClient, turnstileSiteverifyURL, secret, token, clientIP)
	}
	if err == nil {
		return nil
	}
	app.lo.Warn("live chat turnstile check failed", "inbox_id", inboxID, "error", err)
	return envelope.NewError(envelope.InputError, app.i18n.T("widget.turnstileFailed"), map[string]any{"code": turnstileRequiredCode})
}

// turnstileKeyAlias returns a config key that names a Turnstile field without the exact spelling.
// encoding/json accepts the alias into the struct. Encryption and masking look up the exact key,
// so the alias would be stored and returned in plaintext.
func turnstileKeyAlias(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", err
	}
	return inbox.NonCanonicalTurnstileKey(cfg), nil
}

type turnstileSiteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// verifyTurnstileToken checks a visitor token with Cloudflare. An empty token fails without a
// network call. Transport and HTTP failures fail closed.
func verifyTurnstileToken(ctx context.Context, client *http.Client, endpoint, secret, token, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxTurnstileTokenLength {
		return errTurnstileRejected
	}
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: %s", errTurnstileUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", errTurnstileUnavailable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%w: %s", errTurnstileUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", errTurnstileUnavailable, resp.StatusCode)
	}
	var parsed turnstileSiteverifyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("%w: %s", errTurnstileUnavailable, err)
	}
	if !parsed.Success {
		return fmt.Errorf("%w: %s", errTurnstileRejected, strings.Join(parsed.ErrorCodes, ","))
	}
	return nil
}
