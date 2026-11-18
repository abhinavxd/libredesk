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
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/zerodha/fastglue"
)

const (
	turnstileSiteverifyURL  = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	maxTurnstileKeyLength   = 200
	maxTurnstileTokenLength = 4096
)

var (
	errTurnstileRejected    = errors.New("turnstile rejected token")
	errTurnstileUnavailable = errors.New("turnstile verification unavailable")

	turnstileHTTPClient = &http.Client{Timeout: 5 * time.Second}
)

// chatInitNeedsTurnstile reports whether this conversation init must present a Turnstile token.
// The check is off when the inbox has no secret. Signed-in users skip it. A visitor needs a
// fresh token for every new conversation, including a reply that starts one from a campaign.
func chatInitNeedsTurnstile(secret string, contactID int, isVisitor bool) bool {
	if strings.TrimSpace(secret) == "" {
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

func enforceChatInitTurnstile(r *fastglue.Request, app *App, config livechat.Config, token, clientIP string, contactID int, isVisitor bool, inboxID int) error {
	if !chatInitNeedsTurnstile(config.TurnstileSecretKey, contactID, isVisitor) {
		return nil
	}
	err := verifyTurnstileToken(r.RequestCtx, turnstileHTTPClient, turnstileSiteverifyURL, config.TurnstileSecretKey, token, clientIP)
	if err == nil {
		return nil
	}
	app.lo.Warn("live chat turnstile check failed", "inbox_id", inboxID, "error", err)
	return r.SendErrorEnvelope(http.StatusBadRequest, app.i18n.T("widget.turnstileFailed"), nil, envelope.InputError)
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
