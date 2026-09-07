package main

import (
	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleLogin logs in the user and returns the user.
func handleLogin(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		loginReq loginRequest
	)

	// Decode JSON request.
	if err := r.Decode(&loginReq, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if loginReq.Email == "" || loginReq.Password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Verify email and password.
	user, err := app.user.VerifyPassword(loginReq.Email, []byte(loginReq.Password))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if user is enabled.
	if !user.Enabled {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.accountDisabled"), nil))
	}

	status, err := app.twoFactor.Status(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if status.Enabled {
		if err := app.auth.BeginPendingLogin(r, auth_.PendingLogin{UserID: user.ID, CredentialHash: credentialHash(user.Password.String)}); err != nil {
			return sendErrorEnvelope(r, err)
		}
		r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
		return r.SendEnvelope(map[string]bool{"two_factor_required": true})
	}
	return completeLogin(r, user)
}

func completeLogin(r *fastglue.Request, user models.User) error {
	app := r.Context.(*App)
	saveSession := app.auth.SaveSession
	if r.RequestCtx.UserValue("two_factor_verified") == true {
		saveSession = app.auth.SaveTwoFactorSession
	}
	if err := saveSession(amodels.User{ID: user.ID, Email: user.Email.String, FirstName: user.FirstName, LastName: user.LastName}, r); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	if err := app.auth.SetCSRFCookie(r); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.user.InvalidateAgentCache(user.ID)
	if err := app.activityLog.Login(user.ID, user.Email.String, realip.FromRequest(r.RequestCtx)); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(user)
}

// handleLogout logs out the user and redirects to the dashboard.
func handleLogout(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		ip    = realip.FromRequest(r.RequestCtx)
	)

	// Insert activity log.
	if err := app.activityLog.Logout(auser.ID, auser.Email, ip); err != nil {
		app.lo.Error("error creating logout activity log", "error", err)
	}

	if err := app.auth.DestroySession(r); err != nil {
		app.lo.Error("error destroying session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Add no-cache headers.
	r.RequestCtx.Response.Header.Add("Cache-Control",
		"no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")
	return r.RedirectURI("/", fasthttp.StatusFound, nil, "")
}
