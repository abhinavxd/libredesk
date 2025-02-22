package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetRoles(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.role.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

func handleGetRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	role, err := app.role.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(role)
}

func handleDeleteRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	err := app.role.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleCreateRole(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Role{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}
	err := app.role.Create(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdateRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   = models.Role{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}
	err := app.role.Update(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
