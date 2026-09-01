package main

import (
	"context"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const totpPendingTTL = 5 * time.Minute

type totpVerifyReq struct {
	PendingToken string `json:"pending_token"`
	Code         string `json:"code"`
}

type totpCodeReq struct {
	Code string `json:"code"`
}

func handleVerifyTOTP(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		ip  = realip.FromRequest(r.RequestCtx)
		req totpVerifyReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if req.PendingToken == "" || req.Code == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	val, err := app.redis.Get(context.Background(), "totp_pending:"+req.PendingToken).Result()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("auth.totp.invalid"), nil, envelope.PermissionError)
	}
	userID, err := strconv.Atoi(val)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	if err := app.user.VerifyTOTP(userID, req.Code); err != nil {
		return sendErrorEnvelope(r, err)
	}
	user, err := app.user.GetAgent(userID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !user.Enabled {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.accountDisabled"), nil))
	}
	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	if err := app.auth.SetCSRFCookie(r); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	_ = app.user.UpdateLastLoginAt(user.ID)
	app.user.InvalidateAgentCache(user.ID)
	app.redis.Del(context.Background(), "totp_pending:"+req.PendingToken)
	if err := app.activityLog.Login(user.ID, user.Email.String, ip); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}
	return r.SendEnvelope(user)
}

func handleEnrollTOTP(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	secret, url, err := app.user.BeginTOTPEnroll(user.ID, user.Email.String)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]string{"secret": secret, "otpauth_url": url})
}

func handleConfirmTOTP(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   totpCodeReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if err := app.user.ConfirmTOTP(auser.ID, req.Code); err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.user.InvalidateAgentCache(auser.ID)
	return r.SendEnvelope(true)
}

func handleDisableTOTP(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if err := app.user.DisableTOTP(auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.user.InvalidateAgentCache(auser.ID)
	return r.SendEnvelope(true)
}

func issueTOTPPending(app *App, userID int) (string, error) {
	token, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		return "", err
	}
	if err := app.redis.Set(context.Background(), "totp_pending:"+token, strconv.Itoa(userID), totpPendingTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}
