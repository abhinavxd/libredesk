package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// customAnswerReq represents the request payload for custom answers.
type customAnswerReq struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Enabled  bool   `json:"enabled"`
}

// validateCustomAnswerReq validates the custom answer request payload.
func validateCustomAnswerReq(r *fastglue.Request, customAnswerData *customAnswerReq) error {
	var app = r.Context.(*App)
	if customAnswerData.Question == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`question`"), nil, envelope.InputError)
	}
	if customAnswerData.Answer == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`answer`"), nil, envelope.InputError)
	}
	return nil
}

// handleGetAICustomAnswers returns all AI custom answers from the database.
func handleGetAICustomAnswers(r *fastglue.Request) error {
	var app = r.Context.(*App)
	customAnswers, err := app.ai.GetAICustomAnswers()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(customAnswers)
}

// handleGetAICustomAnswer returns a single AI custom answer by ID.
func handleGetAICustomAnswer(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	customAnswer, err := app.ai.GetAICustomAnswer(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(customAnswer)
}

// handleCreateAICustomAnswer creates a new AI custom answer in the database.
func handleCreateAICustomAnswer(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		customAnswerData customAnswerReq
	)
	if err := r.Decode(&customAnswerData, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if err := validateCustomAnswerReq(r, &customAnswerData); err != nil {
		return err
	}
	customAnswer, err := app.ai.CreateAICustomAnswer(customAnswerData.Question, customAnswerData.Answer, customAnswerData.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(customAnswer)
}

// handleUpdateAICustomAnswer updates an existing AI custom answer in the database.
func handleUpdateAICustomAnswer(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		customAnswerData customAnswerReq
		id, _            = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&customAnswerData, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if err := validateCustomAnswerReq(r, &customAnswerData); err != nil {
		return err
	}
	customAnswer, err := app.ai.UpdateAICustomAnswer(id, customAnswerData.Question, customAnswerData.Answer, customAnswerData.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(customAnswer)
}

// handleDeleteAICustomAnswer deletes an AI custom answer from the database.
func handleDeleteAICustomAnswer(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.ai.DeleteAICustomAnswer(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
