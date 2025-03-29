package main

import (
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	smodels "github.com/abhinavxd/libredesk/internal/sla/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetSLAs returns all SLAs.
func handleGetSLAs(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	slas, err := app.sla.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(slas)
}

// handleGetSLA returns the SLA with the given ID.
func handleGetSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "SLA `id`"), nil, envelope.InputError)
	}

	sla, err := app.sla.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(sla)
}

// handleCreateSLA creates a new SLA.
func handleCreateSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		sla smodels.SLAPolicy
	)

	if err := r.Decode(&sla, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), err.Error(), envelope.InputError)
	}

	if err := validateSLA(app, &sla); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.sla.Create(sla.Name, sla.Description, sla.FirstResponseTime, sla.ResolutionTime, sla.Notifications); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("SLA created successfully.")
}

// handleUpdateSLA updates the SLA with the given ID.
func handleUpdateSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		sla smodels.SLAPolicy
	)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "SLA `id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&sla, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), err.Error(), envelope.InputError)
	}

	if err := validateSLA(app, &sla); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.sla.Update(id, sla.Name, sla.Description, sla.FirstResponseTime, sla.ResolutionTime, sla.Notifications); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("SLA updated successfully.")
}

// handleDeleteSLA deletes the SLA with the given ID.
func handleDeleteSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "SLA `id`"), nil, envelope.InputError)
	}

	if err = app.sla.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateSLA validates the SLA policy and returns an envelope.Error if any validation fails.
func validateSLA(app *App, sla *smodels.SLAPolicy) error {
	if sla.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA `name`"), nil)
	}
	if sla.FirstResponseTime == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA `first_response_time`"), nil)
	}
	if sla.ResolutionTime == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA `resolution_time`"), nil)
	}

	// Validate notifications if any
	for _, n := range sla.Notifications {
		if n.Type == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA notification `type`"), nil)
		}
		if n.TimeDelayType == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA notification `time_delay_type`"), nil)
		}
		if n.TimeDelayType != "immediately" {
			if n.TimeDelay == "" {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA notification `time_delay`"), nil)
			}
		}
		if len(n.Recipients) == 0 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "SLA notification `recipients`"), nil)
		}
	}

	// Validate time duration strings
	frt, err := time.ParseDuration(sla.FirstResponseTime)
	if err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`first_response_time`"), nil)
	}
	if frt.Minutes() < 1 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`first_response_time`"), nil)
	}

	rt, err := time.ParseDuration(sla.ResolutionTime)
	if err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`resolution_time`"), nil)
	}
	if rt.Minutes() < 1 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`resolution_time`"), nil)
	}
	if frt > rt {
		return envelope.NewError(envelope.InputError, app.i18n.T("sla.firstResponseTimeAfterResolution"), nil)
	}

	return nil
}
