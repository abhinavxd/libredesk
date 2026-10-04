package main

import (
	"slices"
	"strconv"

	"github.com/jakedolan443/fernmail/internal/activationkey"
	amodels "github.com/jakedolan443/fernmail/internal/auth/models"
	authzModels "github.com/jakedolan443/fernmail/internal/authz/models"
	cmodels "github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Key Distribution. The pools are Admin-only; anyone who can send or submit
// email reads the composer card's list of games. Keys themselves only leave
// the server one at a time through reveal, or inside an outgoing email.

type keyAppReq struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

type keyImportReq struct {
	Keys string `json:"keys"`
}

type keyComposerAppReq struct {
	AppID int `json:"app_id"`
}

type keyPage struct {
	Results    []activationkey.Key `json:"results"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PerPage    int                 `json:"per_page"`
	TotalPages int                 `json:"total_pages"`
}

func pathID(r *fastglue.Request) (int64, error) {
	id, err := strconv.ParseInt(r.RequestCtx.UserValue("id").(string), 10, 64)
	if err != nil || id <= 0 {
		return 0, envelope.NewError(envelope.InputError, "Invalid ID.", nil)
	}
	return id, nil
}

func requestUserID(r *fastglue.Request) int {
	return r.RequestCtx.UserValue("user").(amodels.User).ID
}

// canSeeActivationKeys reports whether keys quoted back in a conversation stay
// visible to user. Everyone else sees them masked.
func canSeeActivationKeys(user umodels.User) bool {
	return slices.Contains(user.Permissions, authzModels.PermActivationKeysManage)
}

// maskActivationKeysFor hides sent keys quoted in messages from viewers who
// may not see them. It fails closed: if the keys can't be checked, nothing is shown.
func maskActivationKeysFor(app *App, user umodels.User, messages []cmodels.Message) error {
	if canSeeActivationKeys(user) {
		return nil
	}
	if err := app.conversation.MaskActivationKeys(messages); err != nil {
		app.lo.Error("error masking activation keys", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func handleGetKeyDistributionSettings(r *fastglue.Request) error {
	app := r.Context.(*App)
	settings, err := app.activationKeys.GetSettings()
	if err != nil {
		app.lo.Error("error reading key distribution settings", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(settings)
}

func handleUpdateKeyDistributionSettings(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req activationkey.Settings
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	settings, err := app.activationKeys.UpdateSettings(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(settings)
}

func handleGetKeyApps(r *fastglue.Request) error {
	apps, err := r.Context.(*App).activationKeys.Apps()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(apps)
}

func handleCreateKeyApp(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.activationKeys.RequireEnabled(); err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req keyAppReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	created, err := app.activationKeys.CreateApp(req.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
}

func handleUpdateKeyApp(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.activationKeys.RequireEnabled(); err != nil {
		return sendErrorEnvelope(r, err)
	}
	id, err := pathID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req keyAppReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	updated, err := app.activationKeys.UpdateApp(int(id), req.Name, req.Archived)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updated)
}

func handleGetKeys(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := pathID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	page, perPage := getPagination(r)
	args := r.RequestCtx.QueryArgs()
	keys, total, err := app.activationKeys.Keys(activationkey.KeyQuery{
		AppID:   int(id),
		Status:  string(args.Peek("status")),
		Search:  string(args.Peek("search")),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(keyPage{Results: keys, Total: total, Page: page, PerPage: perPage, TotalPages: (total + perPage - 1) / perPage})
}

func handleImportKeys(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.activationKeys.RequireEnabled(); err != nil {
		return sendErrorEnvelope(r, err)
	}
	id, err := pathID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req keyImportReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	result, err := app.activationKeys.Import(int(id), req.Keys, requestUserID(r))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(result)
}

func handleRevealKey(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.activationKeys.RequireEnabled(); err != nil {
		return sendErrorEnvelope(r, err)
	}
	id, err := pathID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	key, err := app.activationKeys.Reveal(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(map[string]string{"key": key})
}

func handleVoidKey(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.activationKeys.RequireEnabled(); err != nil {
		return sendErrorEnvelope(r, err)
	}
	id, err := pathID(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.activationKeys.Void(id, requestUserID(r)); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// canUseKeyCard reports whether user writes email that can carry keys.
func canUseKeyCard(user umodels.User) bool {
	return canSendDirectly(user) || canSubmitForReview(user)
}

func handleGetKeyComposer(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canUseKeyCard(user) {
		return r.SendEnvelope(activationkey.ComposerInfo{Apps: []activationkey.ComposerApp{}})
	}
	info, err := app.activationKeys.Composer(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(info)
}

func handleSetKeyComposerApp(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canUseKeyCard(user) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	var req keyComposerAppReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if err := app.activationKeys.SetLastApp(user.ID, req.AppID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
