package main

import (
	"github.com/abhinavxd/libredesk/internal/ai"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

type aiCompletionReq struct {
	PromptKey string `json:"prompt_key"`
	Content   string `json:"content"`
}

type providerUpdateReq struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

type setDefaultProviderReq struct {
	Provider string `json:"provider"`
}

type testProviderReq struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// handleAICompletion handles AI completion requests
func handleAICompletion(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = aiCompletionReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}

	resp, err := app.ai.Completion(req.PromptKey, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAIPrompts returns AI prompts
func handleGetAIPrompts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	resp, err := app.ai.GetPrompts()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAIProviders returns configured AI providers
func handleGetAIProviders(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	resp, err := app.ai.GetProviders()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAvailableModels returns available models for OpenRouter
func handleGetAvailableModels(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	models := app.ai.GetAvailableModels()
	return r.SendEnvelope(models)
}

// handleUpdateAIProvider updates the AI provider
func handleUpdateAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req providerUpdateReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if err := app.ai.UpdateProvider(req.Provider, req.APIKey, req.Model); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Provider updated successfully")
}

// handleSetDefaultAIProvider sets the default AI provider
func handleSetDefaultAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req setDefaultProviderReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if err := app.ai.SetDefaultProvider(req.Provider); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Default provider updated successfully")
}

// handleTestAIProvider tests the AI provider connection
func handleTestAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req testProviderReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if err := app.ai.TestProvider(req.Provider, req.APIKey, req.Model); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Connection successful")
}

// handleGetSupportedProviders returns list of supported AI provider types
func handleGetSupportedProviders(r *fastglue.Request) error {
	return r.SendEnvelope(ai.SupportedProviders)
}
