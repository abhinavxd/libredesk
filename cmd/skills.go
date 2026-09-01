package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type skillReq struct {
	Name string `json:"name"`
}

type agentSkillsReq struct {
	SkillIDs []int `json:"skill_ids"`
}

func handleGetSkills(r *fastglue.Request) error {
	app := r.Context.(*App)
	skills, err := app.user.GetAllSkills()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(skills)
}

func handleCreateSkill(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req skillReq
	)
	if err := r.Decode(&req, "json"); err != nil || req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	skill, err := app.user.CreateSkill(req.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(skill)
}

func handleUpdateSkill(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req skillReq
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil || req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	skill, err := app.user.UpdateSkill(id, req.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(skill)
}

func handleDeleteSkill(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := app.user.DeleteSkill(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleGetAgentSkills(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	skills, err := app.user.ListAgentSkills(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(skills)
}

func handleSetAgentSkills(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req agentSkillsReq
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if req.SkillIDs == nil {
		req.SkillIDs = []int{}
	}
	if err := app.user.SetAgentSkills(id, req.SkillIDs); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
