package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
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
		app                             = r.Context.(*App)
		name                            = string(r.RequestCtx.PostArgs().Peek("name"))
		timezone                        = string(r.RequestCtx.PostArgs().Peek("timezone"))
		emoji                           = string(r.RequestCtx.PostArgs().Peek("emoji"))
		conversationAssignmentType      = string(r.RequestCtx.PostArgs().Peek("conversation_assignment_type"))
		businessHrsID, _                = strconv.Atoi(string(r.RequestCtx.PostArgs().Peek("business_hours_id")))
		slaPolicyID, _                  = strconv.Atoi(string(r.RequestCtx.PostArgs().Peek("sla_policy_id")))
		maxAutoAssignedConversations, _ = strconv.Atoi(string(r.RequestCtx.PostArgs().Peek("max_auto_assigned_conversations")))
	)
	if err := app.team.Create(name, timezone, conversationAssignmentType, null.NewInt(businessHrsID, businessHrsID != 0), null.NewInt(slaPolicyID, slaPolicyID != 0), emoji, maxAutoAssignedConversations); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateTeam updates an existing team.
func handleUpdateTeam(r *fastglue.Request) error {
	var (
		app                             = r.Context.(*App)
		name                            = string(r.RequestCtx.PostArgs().Peek("name"))
		timezone                        = string(r.RequestCtx.PostArgs().Peek("timezone"))
		emoji                           = string(r.RequestCtx.PostArgs().Peek("emoji"))
		conversationAssignmentType      = string(r.RequestCtx.PostArgs().Peek("conversation_assignment_type"))
		id, _                           = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		businessHrsID, _                = strconv.Atoi(string(r.RequestCtx.PostArgs().Peek("business_hours_id")))
		slaPolicyID, _                  = strconv.Atoi(string(r.RequestCtx.PostArgs().Peek("sla_policy_id")))
		maxAutoAssignedConversations, _ = strconv.Atoi(string(r.RequestCtx.PostArgs().Peek("max_auto_assigned_conversations")))
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid team `id`", nil, envelope.InputError)
	}
	if err := app.team.Update(id, name, timezone, conversationAssignmentType, null.NewInt(businessHrsID, businessHrsID != 0), null.NewInt(slaPolicyID, slaPolicyID != 0), emoji, maxAutoAssignedConversations); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
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
