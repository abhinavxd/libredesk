package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetAutomationRules gets all automation rules
func handleGetAutomationRules(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		typ = r.RequestCtx.QueryArgs().Peek("type")
	)
	out, err := app.automation.GetAllRules(typ)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleGetAutomationRuleByID gets an automation rule by ID
func handleGetAutomationRule(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	out, err := app.automation.GetRule(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleToggleAutomationRule toggles an automation rule
func handleToggleAutomationRule(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err := app.automation.ToggleRule(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateAutomationRule updates an automation rule
func handleUpdateAutomationRule(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		rule    = amodels.RuleRecord{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&rule, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.InputError)
	}

	if err = app.automation.UpdateRule(id, rule); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleCreateAutomationRule creates a new automation rule
func handleCreateAutomationRule(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		rule = amodels.RuleRecord{}
	)
	if err := r.Decode(&rule, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.InputError)
	}
	if err := app.automation.CreateRule(rule); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleDeleteAutomationRule deletes an automation rule
func handleDeleteAutomationRule(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)

		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err = app.automation.DeleteRule(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateAutomationRuleWeights updates the weights of the automation rules
func handleUpdateAutomationRuleWeights(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		weights = make(map[int]int)
	)
	if err := r.Decode(&weights, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), nil, envelope.InputError)
	}
	err := app.automation.UpdateRuleWeights(weights)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateAutomationRuleExecutionMode updates the execution mode of the automation rules for a given type
func handleUpdateAutomationRuleExecutionMode(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		mode = string(r.RequestCtx.PostArgs().Peek("mode"))
	)
	if mode != amodels.ExecutionModeAll && mode != amodels.ExecutionModeFirstMatch {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("automation.invalidRuleExecutionMode"), nil, envelope.InputError)
	}
	// Only new conversation rules can be updated as they are the only ones that have execution mode.
	if err := app.automation.UpdateRuleExecutionMode(amodels.RuleTypeNewConversation, mode); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
