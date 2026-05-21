package main

import (
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/context_link/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetContextLinks(r *fastglue.Request) error {
	var app = r.Context.(*App)
	links, err := app.contextLink.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for i := range links {
		if links[i].Secret != "" {
			links[i].Secret = strings.Repeat(stringutil.PasswordDummy, 10)
		}
	}
	return r.SendEnvelope(links)
}

func handleGetContextLink(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	contextLink, err := app.contextLink.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if contextLink.Secret != "" {
		contextLink.Secret = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	return r.SendEnvelope(contextLink)
}

func handleCreateContextLink(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		contextLink = models.ContextLink{}
	)
	if err := r.Decode(&contextLink, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if err := validateContextLink(app, contextLink); err != nil {
		return sendErrorEnvelope(r, err)
	}

	result, err := app.contextLink.Create(contextLink)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result.Secret = strings.Repeat(stringutil.PasswordDummy, 10)
	return r.SendEnvelope(result)
}

func handleUpdateContextLink(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		contextLink = models.ContextLink{}
		id, _       = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&contextLink, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if err := validateContextLink(app, contextLink); err != nil {
		return sendErrorEnvelope(r, err)
	}

	result, err := app.contextLink.Update(id, contextLink)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result.Secret = strings.Repeat(stringutil.PasswordDummy, 10)
	return r.SendEnvelope(result)
}

func handleDeleteContextLink(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.contextLink.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleToggleContextLink(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	result, err := app.contextLink.Toggle(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result.Secret = strings.Repeat(stringutil.PasswordDummy, 10)
	return r.SendEnvelope(result)
}

// handleGetActiveContextLinks returns active context links for agents.
func handleGetActiveContextLinks(r *fastglue.Request) error {
	var app = r.Context.(*App)
	links, err := app.contextLink.GetActive()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(links)
}

// handleGetContextLinkURL generates the full URL with substituted variables and optional JWT.
func handleGetContextLinkURL(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		auser            = r.RequestCtx.UserValue("user").(amodels.User)
		id, _            = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		conversationUUID = string(r.RequestCtx.QueryArgs().Peek("conversation_uuid"))
	)
	if id <= 0 || conversationUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	agent, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conv, err := enforceConversationAccess(app, conversationUUID, agent)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	contextLink, err := app.contextLink.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	url, err := app.contextLink.GenerateURL(contextLink, conv.Contact, conversationUUID, auser)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(url)
}

func validateContextLink(app *App, contextLink models.ContextLink) error {
	if contextLink.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if contextLink.URLTemplate == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`url_template`"), nil)
	}
	if contextLink.Secret != "" && !strings.Contains(contextLink.Secret, stringutil.PasswordDummy) && len(contextLink.Secret) != 32 {
		return envelope.NewError(envelope.InputError, app.i18n.T("contextLink.secretLengthError"), nil)
	}
	if contextLink.TokenExpirySeconds <= 0 {
		contextLink.TokenExpirySeconds = 1200
	}
	return nil
}
