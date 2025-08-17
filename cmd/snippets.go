package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// snippetReq represents the request payload for snippets creation and updates.
type snippetReq struct {
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}

// validateSnippetReq validates the snippet request payload.
func validateSnippetReq(r *fastglue.Request, snippetData *snippetReq) error {
	var app = r.Context.(*App)
	if snippetData.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`content`"), nil, envelope.InputError)
	}
	return nil
}

// handleGetAISnippets returns all AI snippets from the database.
func handleGetAISnippets(r *fastglue.Request) error {
	var app = r.Context.(*App)
	snippets, err := app.ai.GetKnowledgeBaseItems()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(snippets)
}

// handleGetAISnippet returns a single AI snippet by ID.
func handleGetAISnippet(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	snippet, err := app.ai.GetKnowledgeBaseItem(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(snippet)
}

// handleCreateAISnippet creates a new AI snippet in the database.
func handleCreateAISnippet(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		snippetData snippetReq
	)
	if err := r.Decode(&snippetData, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if err := validateSnippetReq(r, &snippetData); err != nil {
		return err
	}

	snippet, err := app.ai.CreateKnowledgeBaseItem("snippet", snippetData.Content, snippetData.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(snippet)
}

// handleUpdateAISnippet updates an existing AI snippet in the database.
func handleUpdateAISnippet(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		snippetData snippetReq
		id, _       = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&snippetData, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	if err := validateSnippetReq(r, &snippetData); err != nil {
		return err
	}
	snippet, err := app.ai.UpdateKnowledgeBaseItem(id, "snippet", snippetData.Content, snippetData.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(snippet)
}

// handleDeleteAISnippet deletes an AI snippet from the database.
func handleDeleteAISnippet(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.ai.DeleteKnowledgeBaseItem(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
