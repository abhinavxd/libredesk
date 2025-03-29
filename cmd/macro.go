package main

import (
	"encoding/json"
	"slices"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	autoModels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/macro/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetMacros returns all macros.
func handleGetMacros(r *fastglue.Request) error {
	var app = r.Context.(*App)
	macros, err := app.macro.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for i, m := range macros {
		var actions []autoModels.RuleAction
		if err := json.Unmarshal(m.Actions, &actions); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.macroAction}"), nil, envelope.GeneralError)
		}
		// Set display values for actions as the value field can contain DB IDs
		if err := setDisplayValues(app, actions); err != nil {
			app.lo.Warn("error setting display values", "error", err)
		}
		if macros[i].Actions, err = json.Marshal(actions); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.macroAction}"), nil, envelope.GeneralError)
		}
	}
	return r.SendEnvelope(macros)
}

// handleGetMacro returns a macro.
func handleGetMacro(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	macro, err := app.macro.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	var actions []autoModels.RuleAction
	if err := json.Unmarshal(macro.Actions, &actions); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.macroAction}"), nil, envelope.GeneralError)
	}
	// Set display values for actions as the value field can contain DB IDs
	if err := setDisplayValues(app, actions); err != nil {
		app.lo.Warn("error setting display values", "error", err)
	}
	if macro.Actions, err = json.Marshal(actions); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.macroAction}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(macro)
}

// handleCreateMacro creates new macro.
func handleCreateMacro(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		macro = models.Macro{}
	)

	if err := r.Decode(&macro, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), err.Error(), envelope.InputError)
	}

	if err := validateMacro(app, macro); err != nil {
		return sendErrorEnvelope(r, err)
	}

	err := app.macro.Create(macro.Name, macro.MessageContent, macro.UserID, macro.TeamID, macro.Visibility, macro.Actions)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(macro)
}

// handleUpdateMacro updates a macro.
func handleUpdateMacro(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		macro = models.Macro{}
	)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid macro `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&macro, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if err := validateMacro(app, macro); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.macro.Update(id, macro.Name, macro.MessageContent, macro.UserID, macro.TeamID, macro.Visibility, macro.Actions); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(macro)
}

// handleDeleteMacro deletes macro.
func handleDeleteMacro(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.macro.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleApplyMacro applies macro actions to a conversation.
func handleApplyMacro(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		auser            = r.RequestCtx.UserValue("user").(amodels.User)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		id, _            = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		incomingActions  = []autoModels.RuleAction{}
	)
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Enforce conversation access.
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if allowed, err := app.authz.EnforceConversationAccess(user, conversation); err != nil || !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("authz.permissionDenied"), nil))
	}

	macro, err := app.macro.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Decode incoming actions.
	if err := r.Decode(&incomingActions, "json"); err != nil {
		app.lo.Error("error unmashalling incoming actions", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.macroAction}"), err.Error(), envelope.InputError)
	}

	// Make sure no duplicate action types are present.
	actionTypes := make(map[string]bool, len(incomingActions))
	for _, act := range incomingActions {
		if actionTypes[act.Type] {
			app.lo.Warn("duplicate action types found in macro apply apply request", "action", act.Type, "user_id", user.ID)
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("macro.duplicateActionsNotAllowed"), nil, envelope.InputError)
		}
		actionTypes[act.Type] = true
	}

	// Validate action permissions.
	for _, act := range incomingActions {
		if !isMacroActionAllowed(act.Type) {
			app.lo.Warn("action not allowed in macro", "action", act.Type, "user_id", user.ID)
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("macro.actionNotAllowed", "name", act.Type), nil, envelope.PermissionError)
		}
		if !hasActionPermission(act.Type, user.Permissions) {
			app.lo.Warn("no permission to execute macro action", "action", act.Type, "user_id", user.ID)
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("macro.permissionDenied"), nil, envelope.PermissionError)
		}
	}

	// Apply actions.
	successCount := 0
	for _, act := range incomingActions {
		if err := app.conversation.ApplyAction(act, conversation, user); err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("macro.couldNotApply"), nil, envelope.GeneralError)
	}

	// Increment usage count.
	app.macro.IncrementUsageCount(macro.ID)

	if successCount < len(incomingActions) {
		return r.SendJSON(fasthttp.StatusMultiStatus, map[string]interface{}{
			"message": app.i18n.T("macro.partiallyApplied"),
		})
	}

	return r.SendJSON(fasthttp.StatusOK, map[string]interface{}{
		"message": app.i18n.T("macro.applied"),
	})
}

// hasActionPermission checks user permission for given action
func hasActionPermission(action string, userPerms []string) bool {
	requiredPerm, exists := autoModels.ActionPermissions[action]
	if !exists {
		return false
	}
	return slices.Contains(userPerms, requiredPerm)
}

// setDisplayValues sets display values for actions.
func setDisplayValues(app *App, actions []autoModels.RuleAction) error {
	getters := map[string]func(int) (string, error){
		autoModels.ActionAssignTeam: func(id int) (string, error) {
			t, err := app.team.Get(id)
			if err != nil {
				app.lo.Warn("team not found for macro action", "team_id", id)
				return "", err
			}
			return t.Name, nil
		},
		autoModels.ActionAssignUser: func(id int) (string, error) {
			u, err := app.user.GetAgent(id)
			if err != nil {
				app.lo.Warn("user not found for macro action", "user_id", id)
				return "", err
			}
			return u.FullName(), nil
		},
		autoModels.ActionSetPriority: func(id int) (string, error) {
			p, err := app.priority.Get(id)
			if err != nil {
				app.lo.Warn("priority not found for macro action", "priority_id", id)
				return "", err
			}
			return p.Name, nil
		},
		autoModels.ActionSetStatus: func(id int) (string, error) {
			s, err := app.status.Get(id)
			if err != nil {
				app.lo.Warn("status not found for macro action", "status_id", id)
				return "", err
			}
			return s.Name, nil
		},
	}
	for i := range actions {
		actions[i].DisplayValue = []string{}
		if getter, ok := getters[actions[i].Type]; ok {
			id, _ := strconv.Atoi(actions[i].Value[0])
			if name, err := getter(id); err == nil {
				actions[i].DisplayValue = append(actions[i].DisplayValue, name)
			}
		}
	}
	return nil
}

// validateMacro validates an incoming macro.
func validateMacro(app *App, macro models.Macro) error {
	if macro.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}

	var act []autoModels.RuleAction
	if err := json.Unmarshal(macro.Actions, &act); err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.macroAction}"), nil)
	}
	for _, a := range act {
		if len(a.Value) == 0 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.emptyActionValue", "name", a.Type), nil)
		}
	}
	return nil
}

// isMacroActionAllowed returns true if the action is allowed in a macro.
func isMacroActionAllowed(action string) bool {
	switch action {
	case autoModels.ActionSendPrivateNote, autoModels.ActionReply:
		return false
	case autoModels.ActionAssignTeam, autoModels.ActionAssignUser, autoModels.ActionSetStatus, autoModels.ActionSetPriority, autoModels.ActionSetTags:
		return true
	default:
		return false
	}
}
