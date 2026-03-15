package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type updateAutomationRuleExecutionModeReq struct {
	Mode string `json:"mode"`
}

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
	toggledRule, err := app.automation.ToggleRule(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(toggledRule)
}

// handleUpdateAutomationRule updates an automation rule
func handleUpdateAutomationRule(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		rule    = amodels.RuleRecord{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err := r.Decode(&rule, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	updatedRule, err := app.automation.UpdateRule(id, rule)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updatedRule)
}

// handleCreateAutomationRule creates a new automation rule
func handleCreateAutomationRule(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		rule = amodels.RuleRecord{}
	)
	if err := r.Decode(&rule, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	createdRule, err := app.automation.CreateRule(rule)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(createdRule)
}

// handleDeleteAutomationRule deletes an automation rule
func handleDeleteAutomationRule(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)

		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
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
		app = r.Context.(*App)
		req = updateAutomationRuleExecutionModeReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}

	if req.Mode != amodels.ExecutionModeAll && req.Mode != amodels.ExecutionModeFirstMatch {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("automation.invalidRuleExecutionMode"), nil, envelope.InputError)
	}

	// Only new conversation rules can be updated as they are the only ones that have execution mode.
	if err := app.automation.UpdateRuleExecutionMode(amodels.RuleTypeNewConversation, req.Mode); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
