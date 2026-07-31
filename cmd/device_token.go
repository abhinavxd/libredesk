package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const maxDeviceNameLength = 140

type deviceTokenReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// handleCreateDeviceToken mints a long-lived token for a mobile device.
func handleCreateDeviceToken(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req deviceTokenReq
	)

	// A stolen token must not be able to mint siblings or extend itself.
	if method, ok := r.RequestCtx.UserValue("auth_method").(string); ok && method == "device_token" {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if len(req.Name) > maxDeviceNameLength {
		req.Name = req.Name[:maxDeviceNameLength]
	}

	user, err := app.user.VerifyPassword(req.Email, []byte(req.Password))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !user.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("user.accountDisabled"), nil, envelope.PermissionError)
	}

	token, record, err := app.user.MintDeviceToken(user.ID, req.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]any{
		"id":    record.ID,
		"token": token,
		"name":  record.Name,
		"email": user.Email.String,
	})
}

// handleGetDeviceTokens returns the calling agent's own device tokens.
func handleGetDeviceTokens(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	tokens, err := app.user.GetDeviceTokens(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tokens)
}

// handleDeleteDeviceToken revokes one of the calling agent's own device tokens.
func handleDeleteDeviceToken(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := app.user.RevokeDeviceToken(auser.ID, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
