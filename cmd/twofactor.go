package main

import (
	"crypto/sha256"
	"encoding/hex"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/fastglue"
)

type twoFactorRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

func handleVerifyTwoFactorLogin(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req twoFactorRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	pending, err := app.auth.PendingLogin(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	user, err := app.user.GetAgent(pending.UserID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// A password reset invalidates any unfinished sign-in.
	if !user.Enabled || user.Type != models.UserTypeAgent || credentialHash(user.Password.String) != pending.CredentialHash {
		return sendErrorEnvelope(r, envelope.NewError(envelope.UnauthorizedError, app.i18n.T("twoFactor.loginExpired"), nil))
	}
	if err := app.twoFactor.Verify(user.ID, req.Code); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.auth.ConsumePendingLogin(r); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if pending.Next != "" {
		r.RequestCtx.Response.Header.Set("X-Login-Next", pending.Next)
	}
	r.RequestCtx.SetUserValue("two_factor_verified", true)
	return completeLogin(r, user)
}

func handleGetTwoFactor(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	status, err := app.twoFactor.Status(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(map[string]any{"enabled": status.Enabled, "recovery_codes_remaining": status.RecoveryCodesRemaining, "verification_required": !app.auth.HasRecentTwoFactorVerification(r)})
}

func handleSetupTwoFactor(r *fastglue.Request) error {
	app := r.Context.(*App)
	_, user, err := verifyTwoFactorPassword(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	setup, err := app.twoFactor.BeginSetup(user.ID, user.Email.String)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(setup)
}

func handleEnableTwoFactor(r *fastglue.Request) error {
	app := r.Context.(*App)
	req, user, err := verifyTwoFactorPassword(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	codes, err := app.twoFactor.ConfirmSetup(user.ID, req.Code)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.lo.Info("two-factor authentication enabled", "user_id", user.ID)
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(map[string]any{"recovery_codes": codes})
}

func handleDisableTwoFactor(r *fastglue.Request) error {
	app := r.Context.(*App)
	req, user, err := verifyTwoFactorPassword(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if app.auth.HasRecentTwoFactorVerification(r) {
		err = app.twoFactor.DisableAfterVerification(user.ID)
	} else {
		err = app.twoFactor.Disable(user.ID, req.Code)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.lo.Info("two-factor authentication disabled", "user_id", user.ID)
	return r.SendEnvelope(true)
}

func handleRegenerateTwoFactorCodes(r *fastglue.Request) error {
	app := r.Context.(*App)
	req, user, err := verifyTwoFactorPassword(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var codes []string
	// This sign-in may have consumed the last recovery code.
	if app.auth.HasRecentTwoFactorVerification(r) {
		codes, err = app.twoFactor.RegenerateRecoveryCodesAfterVerification(user.ID)
	} else {
		codes, err = app.twoFactor.RegenerateRecoveryCodes(user.ID, req.Code)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.lo.Info("two-factor recovery codes regenerated", "user_id", user.ID)
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(map[string]any{"recovery_codes": codes})
}

func verifyTwoFactorPassword(r *fastglue.Request) (twoFactorRequest, models.User, error) {
	app := r.Context.(*App)
	var req twoFactorRequest
	var user models.User
	if r.RequestCtx.UserValue("auth_method") != authMethodSession {
		return req, user, envelope.NewError(envelope.PermissionError, app.i18n.T("twoFactor.sessionRequired"), nil)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return req, user, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil)
	}
	current := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.VerifyPassword(current.Email, []byte(req.Password))
	if err != nil {
		return req, user, err
	}
	if user.ID != current.ID || !user.Enabled {
		return req, user, envelope.NewError(envelope.PermissionError, app.i18n.T("twoFactor.sessionRequired"), nil)
	}
	return req, user, nil
}

func credentialHash(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}
