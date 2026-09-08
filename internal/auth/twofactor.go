package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	pendingLoginCookie   = "libredesk_two_factor"
	pendingLoginPrefix   = "libredesk:pending_login:"
	pendingLoginLifetime = 5 * time.Minute
)

type PendingLogin struct {
	UserID         int    `json:"user_id"`
	CredentialHash string `json:"credential_hash"`
	Next           string `json:"next"`
}

func (a *Auth) BeginPendingLogin(r *fastglue.Request, pending PendingLogin) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return a.pendingLoginError(err)
	}
	token := hex.EncodeToString(raw)
	data, err := json.Marshal(pending)
	if err != nil {
		return a.pendingLoginError(err)
	}
	if err := a.rd.Set(context.Background(), pendingLoginPrefix+token, data, pendingLoginLifetime).Err(); err != nil {
		return a.pendingLoginError(err)
	}
	if previous := string(r.RequestCtx.Request.Header.Cookie(pendingLoginCookie)); len(previous) == 64 {
		if err := a.rd.Del(context.Background(), pendingLoginPrefix+previous).Err(); err != nil {
			return a.pendingLoginError(err)
		}
	}
	a.setPendingLoginCookie(r, token, int(pendingLoginLifetime.Seconds()))
	return nil
}

func (a *Auth) PendingLogin(r *fastglue.Request) (PendingLogin, error) {
	var pending PendingLogin
	token := string(r.RequestCtx.Request.Header.Cookie(pendingLoginCookie))
	if len(token) != 64 {
		return pending, a.pendingLoginError(redis.Nil)
	}
	data, err := a.rd.Get(context.Background(), pendingLoginPrefix+token).Bytes()
	if err != nil {
		return pending, a.pendingLoginError(err)
	}
	if err := json.Unmarshal(data, &pending); err != nil {
		return pending, a.pendingLoginError(err)
	}
	if pending.UserID <= 0 {
		return pending, a.pendingLoginError(redis.Nil)
	}
	return pending, nil
}

func (a *Auth) ConsumePendingLogin(r *fastglue.Request) error {
	token := string(r.RequestCtx.Request.Header.Cookie(pendingLoginCookie))
	if len(token) != 64 {
		return a.pendingLoginError(redis.Nil)
	}
	if err := a.rd.GetDel(context.Background(), pendingLoginPrefix+token).Err(); err != nil {
		return a.pendingLoginError(err)
	}
	a.setPendingLoginCookie(r, "", -1)
	return nil
}

func (a *Auth) HasRecentTwoFactorVerification(r *fastglue.Request) bool {
	value, err := a.GetSessionValue(r, "two_factor_verified_at")
	if err != nil {
		return false
	}
	raw, ok := value.(string)
	if !ok {
		return false
	}
	verifiedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return false
	}
	age := time.Since(verifiedAt)
	return age >= 0 && age < 5*time.Minute
}

func (a *Auth) setPendingLoginCookie(r *fastglue.Request, value string, maxAge int) {
	a.mu.RLock()
	secure := a.cfg.SecureCookies
	a.mu.RUnlock()
	var cookie fasthttp.Cookie
	cookie.SetKey(pendingLoginCookie)
	cookie.SetValue(value)
	cookie.SetPath("/api/v1/auth/2fa")
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(secure)
	cookie.SetSameSite(fasthttp.CookieSameSiteStrictMode)
	cookie.SetMaxAge(maxAge)
	r.RequestCtx.Response.Header.SetCookie(&cookie)
}

func (a *Auth) pendingLoginError(err error) error {
	if errors.Is(err, redis.Nil) {
		return envelope.NewError(envelope.UnauthorizedError, a.i18n.T("twoFactor.loginExpired"), nil)
	}
	a.logger.Error("pending two-factor login failed", "error", err)
	return envelope.NewError(envelope.GeneralError, a.i18n.T("globals.messages.somethingWentWrong"), nil)
}
