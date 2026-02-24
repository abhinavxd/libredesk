package main

import (
	"slices"
	"strconv"

	cmodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

var (
	// disallowedKeys contains keys that are not allowed for custom attributes as they're the default fields.
	disallowedKeys = []string{
		"contact_email",
		"content",
		"subject",
		"status",
		"priority",
		"assigned_team",
		"assigned_user",
		"hours_since_created",
		"hours_since_first_reply",
		"hours_since_last_reply",
		"hours_since_resolved",
		"inbox",
	}
)

// handleGetCustomAttribute retrieves a custom attribute by its ID.
func handleGetCustomAttribute(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	attribute, err := app.customAttribute.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(attribute)
}

// handleGetCustomAttributes retrieves all custom attributes from the database.
func handleGetCustomAttributes(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		appliesTo = string(r.RequestCtx.QueryArgs().Peek("applies_to"))
	)
	attributes, err := app.customAttribute.GetAll(appliesTo)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(attributes)
}

// handleCreateCustomAttribute creates a new custom attribute in the database.
func handleCreateCustomAttribute(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		attribute = cmodels.CustomAttribute{}
	)
	if err := r.Decode(&attribute, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if err := validateCustomAttribute(app, attribute); err != nil {
		return sendErrorEnvelope(r, err)
	}
	createdAttr, err := app.customAttribute.Create(attribute)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(createdAttr)
}

// handleUpdateCustomAttribute updates an existing custom attribute in the database.
func handleUpdateCustomAttribute(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		attribute = cmodels.CustomAttribute{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&attribute, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if err := validateCustomAttribute(app, attribute); err != nil {
		return sendErrorEnvelope(r, err)
	}
	updatedAttr, err := app.customAttribute.Update(id, attribute)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updatedAttr)
}

// handleDeleteCustomAttribute deletes a custom attribute from the database.
func handleDeleteCustomAttribute(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err = app.customAttribute.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// validateCustomAttribute validates a custom attribute.
func validateCustomAttribute(app *App, attribute cmodels.CustomAttribute) error {
	if attribute.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if attribute.AppliesTo == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`applies_to`"), nil)
	}
	if attribute.DataType == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`type`"), nil)
	}
	if attribute.Description == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`description`"), nil)
	}
	if attribute.Key == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`key`"), nil)
	}
	if slices.Contains(disallowedKeys, attribute.Key) {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.customAttributes.keyNotAllowed"), nil)
	}
	return nil
}
