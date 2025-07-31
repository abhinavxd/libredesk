package main

import (
	"encoding/json"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type aiAssisantRequest struct {
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Email              string `json:"email"`
	AvatarURL          string `json:"avatar_url"`
	ProductName        string `json:"product_name"`
	ProductDescription string `json:"product_description"`
	AnswerLength       string `json:"answer_length"`
	AnswerTone         string `json:"answer_tone"`
	HandOff            bool   `json:"hand_off"`
	HandOffTeam        int    `json:"hand_off_team"`
	Enabled            bool   `json:"enabled"`
}

// handleGetAIAssistants returns all AI assistants from the database.
func handleGetAIAssistants(r *fastglue.Request) error {
	var app = r.Context.(*App)
	assistants, err := app.user.GetAIAssistants()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(assistants)
}

// handleGetAIAssistant returns a single AI assistant by ID.
func handleGetAIAssistant(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	assistant, err := app.user.GetAIAssistant(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(assistant)
}

// handleCreateAIAssistant creates a new AI assistant in the database.
func handleCreateAIAssistant(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = aiAssisantRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateAIAssistantRequest(req, app); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Prepare meta data
	meta := umodels.AIAssistantMeta{
		ProductName:        req.ProductName,
		ProductDescription: req.ProductDescription,
		AnswerLength:       req.AnswerLength,
		AnswerTone:         req.AnswerTone,
		HandOff:            req.HandOff,
		HandOffTeam:        req.HandOffTeam,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorMarshalling", "name", "{globals.terms.meta}"), err.Error(), envelope.GeneralError)
	}

	// Create AI assistant in the database
	assistant := &umodels.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     null.NewString(req.Email, req.Email != ""),
		AvatarURL: null.NewString(req.AvatarURL, req.AvatarURL != ""),
		Type:      umodels.UserTypeAIAssistant,
		Enabled:   true,
		Meta:      metaBytes,
	}
	if err := app.user.CreateAIAssistant(assistant); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(assistant)
}

// handleUpdateAIAssistant updates an existing AI assistant in the database.
func handleUpdateAIAssistant(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		req   = aiAssisantRequest{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if err := validateAIAssistantRequest(req, app); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Prepare meta data
	meta := umodels.AIAssistantMeta{
		ProductName:        req.ProductName,
		ProductDescription: req.ProductDescription,
		AnswerLength:       req.AnswerLength,
		AnswerTone:         req.AnswerTone,
		HandOff:            req.HandOff,
		HandOffTeam:        req.HandOffTeam,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error encoding meta data", err.Error(), envelope.GeneralError)
	}

	// Update AI assistant in the database
	assistant := umodels.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     null.NewString(req.Email, req.Email != ""),
		AvatarURL: null.NewString(req.AvatarURL, req.AvatarURL != ""),
		Enabled:   req.Enabled,
		Meta:      metaBytes,
	}
	if err := app.user.UpdateAIAssistant(id, assistant); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Return the updated assistant
	updatedAssistant, err := app.user.GetAIAssistant(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(updatedAssistant)
}

// handleDeleteAIAssistant soft deletes an AI assistant from the database.
func handleDeleteAIAssistant(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := app.user.SoftDeleteAIAssistant(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateAIAssistantRequest validates the fields of an aiAssisantRequest.
func validateAIAssistantRequest(req aiAssisantRequest, app *App) error {
	if req.FirstName == "" {
		return envelope.NewError("validation_error", app.i18n.Ts("globals.messages.empty", "name", "`first_name`"), nil)
	}
	if req.ProductName == "" {
		return envelope.NewError("validation_error", app.i18n.Ts("globals.messages.empty", "name", "`product_name`"), nil)
	}
	if req.ProductDescription == "" {
		return envelope.NewError("validation_error", app.i18n.Ts("globals.messages.empty", "name", "`product_description`"), nil)
	}
	if req.AnswerLength == "" {
		return envelope.NewError("validation_error", app.i18n.Ts("globals.messages.empty", "name", "`answer_length`"), nil)
	}
	if req.AnswerTone == "" {
		return envelope.NewError("validation_error", app.i18n.Ts("globals.messages.empty", "name", "`answer_tone`"), nil)
	}
	return nil
}
