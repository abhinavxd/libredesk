package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetTeams returns a list of all teams.
func handleGetTeams(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	teams, err := app.team.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

// handleGetTeamsCompact returns a list of all teams in a compact format.
func handleGetTeamsCompact(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	teams, err := app.team.GetAllCompact()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

// handleGetTeam returns a single team.
func handleGetTeam(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	team, err := app.team.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(team)
}

// handleCreateTeam creates a new team.
func handleCreateTeam(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Team{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}

	createdTeam, err := app.team.Create(req.Name, req.Timezone, req.ConversationAssignmentType, req.BusinessHoursID, req.SLAPolicyID, req.Emoji.String, req.MaxAutoAssignedConversations)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(createdTeam)
}

// handleUpdateTeam updates an existing team.
func handleUpdateTeam(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   = models.Team{}
	)

	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}

	updatedTeam, err := app.team.Update(id, req.Name, req.Timezone, req.ConversationAssignmentType, req.BusinessHoursID, req.SLAPolicyID, req.Emoji.String, req.MaxAutoAssignedConversations)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updatedTeam)
}

// handleDeleteTeam deletes a team
func handleDeleteTeam(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	err = app.team.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
