package main

import (
	"strconv"

	businessHours "github.com/abhinavxd/libredesk/internal/business_hours"
	models "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetBusinessHours returns all business hours.
func handleGetBusinessHours(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	businessHours, err := app.businessHours.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(businessHours)
}

// handleGetBusinessHour returns the business hour with the given id.
func handleGetBusinessHour(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	businessHour, err := app.businessHours.Get(id)
	if err != nil {
		if err == businessHours.ErrBusinessHoursNotFound {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, err.Error(), nil, envelope.NotFoundError)
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.businessHour}"), nil, "")
	}
	return r.SendEnvelope(businessHour)
}

// handleCreateBusinessHours creates a new business hour.
func handleCreateBusinessHours(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		businessHours = models.BusinessHours{}
	)
	if err := r.Decode(&businessHours, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.InputError)
	}

	if businessHours.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.entitites.name}"), nil, envelope.InputError)
	}

	if err := app.businessHours.Create(businessHours.Name, businessHours.Description, businessHours.IsAlwaysOpen, businessHours.Hours, businessHours.Holidays); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleDeleteBusinessHour deletes the business hour with the given id.
func handleDeleteBusinessHour(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err = app.businessHours.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateBusinessHours updates the business hour with the given id.
func handleUpdateBusinessHours(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		businessHours = models.BusinessHours{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&businessHours, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), err.Error(), envelope.InputError)
	}
	if businessHours.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "business hour `Name`"), nil, envelope.InputError)
	}
	if err := app.businessHours.Update(id, businessHours.Name, businessHours.Description, businessHours.IsAlwaysOpen, businessHours.Hours, businessHours.Holidays); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
