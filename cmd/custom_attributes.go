package main

import (
	"encoding/json"
	"slices"
	"strconv"

	cmodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
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
	if err := removeAttributeFromPreChatForms(app, id); err != nil {
		return sendErrorEnvelope(r, err)
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

func removeAttributeFromPreChatForms(app *App, attributeID int) error {
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return err
	}
	var failed int
	for _, inb := range inboxes {
		if inb.Channel != livechat.ChannelLiveChat {
			continue
		}
		config, changed, err := stripPreChatFormAttribute(inb.Config, attributeID)
		if err != nil {
			app.lo.Error("error parsing live chat config", "id", inb.ID, "error", err)
			failed++
			continue
		}
		if !changed {
			continue
		}
		if err := app.inbox.UpdateConfig(inb.ID, config); err != nil {
			failed++
			continue
		}
		if err := reloadInbox(app, inb.ID); err != nil {
			app.lo.Error("error reloading inbox", "id", inb.ID, "error", err)
			failed++
		}
	}
	if failed > 0 {
		app.lo.Error("error clearing deleted custom attribute from live chat inboxes", "inboxes", failed)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func stripPreChatFormAttribute(raw json.RawMessage, attributeID int) (json.RawMessage, bool, error) {
	var config map[string]any
	if len(raw) == 0 {
		return raw, false, nil
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return raw, false, err
	}
	form, ok := config["prechat_form"].(map[string]any)
	if !ok {
		return raw, false, nil
	}
	var changed bool
	for _, holder := range []map[string]any{form, asObject(form["visitors"]), asObject(form["users"])} {
		if holder == nil {
			continue
		}
		fields, ok := holder["fields"].([]any)
		if !ok {
			continue
		}
		kept := make([]any, 0, len(fields))
		for _, field := range fields {
			if attributeIDOf(field) == attributeID {
				changed = true
				continue
			}
			kept = append(kept, field)
		}
		holder["fields"] = kept
	}
	if !changed {
		return raw, false, nil
	}
	updated, err := json.Marshal(config)
	if err != nil {
		return raw, false, err
	}
	return updated, true, nil
}

func asObject(value any) map[string]any {
	object, _ := value.(map[string]any)
	return object
}

func attributeIDOf(field any) int {
	object, ok := field.(map[string]any)
	if !ok {
		return 0
	}
	id, ok := object["custom_attribute_id"].(float64)
	if !ok {
		return 0
	}
	return int(id)
}
