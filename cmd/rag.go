package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/rag/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetRAGSources returns all RAG knowledge sources.
func handleGetRAGSources(r *fastglue.Request) error {
	app := r.Context.(*App)

	sources, err := app.rag.GetSources()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(sources)
}

// handleGetRAGSource returns a single RAG source by ID.
func handleGetRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	source, err := app.rag.GetSource(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(source)
}

// handleCreateRAGSource creates a new knowledge source.
func handleCreateRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req struct {
		Name       string          `json:"name"`
		SourceType string          `json:"source_type"`
		Config     json.RawMessage `json:"config"`
		Enabled    bool            `json:"enabled"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Validate
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name is required", nil, envelope.InputError)
	}
	if req.SourceType != "macro" && req.SourceType != "webpage" && req.SourceType != "custom" && req.SourceType != "file" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source type", nil, envelope.InputError)
	}

	source, err := app.rag.CreateSource(req.Name, req.SourceType, req.Config, req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(source)
}

// handleUpdateRAGSource updates a knowledge source.
func handleUpdateRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	var req struct {
		Name    string          `json:"name"`
		Config  json.RawMessage `json:"config"`
		Enabled bool            `json:"enabled"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	source, err := app.rag.UpdateSource(id, req.Name, req.Config, req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(source)
}

// handleDeleteRAGSource deletes a knowledge source.
func handleDeleteRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	if err := app.rag.DeleteSource(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleSyncRAGSource triggers a sync for a source.
func handleSyncRAGSource(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid source ID", nil, envelope.InputError)
	}

	// Sync in background
	go func() {
		if err := app.ragSync.SyncSourceByID(id); err != nil {
			app.lo.Error("error syncing source", "source_id", id, "error", err)
		}
	}()

	return r.SendEnvelope(map[string]string{"status": "sync_started"})
}

// handleRAGSearch searches the knowledge base.
func handleRAGSearch(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req struct {
		Query     string  `json:"query"`
		Limit     int     `json:"limit"`
		Threshold float64 `json:"threshold"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if strings.TrimSpace(req.Query) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Query is required", nil, envelope.InputError)
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.25
	}

	results, err := app.rag.Search(req.Query, req.Limit, req.Threshold)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(results)
}

// handleRAGGenerateResponse generates a response using RAG.
func handleRAGGenerateResponse(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req struct {
		ConversationID  int    `json:"conversation_id"`
		CustomerMessage string `json:"customer_message"`
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if strings.TrimSpace(req.CustomerMessage) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Customer message is required", nil, envelope.InputError)
	}

	// Get AI settings
	aiSettings, err := app.setting.GetAISettings()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Use sensible defaults for RAG search
	threshold := aiSettings.SimilarityThreshold
	if threshold <= 0 || threshold > 0.5 {
		threshold = 0.25
	}
	maxChunks := aiSettings.MaxContextChunks
	if maxChunks <= 0 {
		maxChunks = 5
	}

	// Search knowledge base
	results, err := app.rag.Search(req.CustomerMessage, maxChunks, threshold)
	if err != nil {
		app.lo.Warn("RAG search failed, continuing without context", "error", err)
		results = []models.SearchResult{}
	}

	app.lo.Info("RAG generate response", "query", req.CustomerMessage, "results_count", len(results), "threshold", threshold)

	// Build context from results
	var contextParts, macroParts []string
	for _, res := range results {
		if strings.HasPrefix(res.SourceRef, "macro_") {
			macroParts = append(macroParts, "- "+res.Title+": "+res.Content)
		} else {
			contextParts = append(contextParts, "## "+res.Title+"\n"+res.Content)
		}
	}

	contextStr := strings.Join(contextParts, "\n\n")
	macrosStr := strings.Join(macroParts, "\n")

	// Build system prompt from template - use default if not set
	systemPrompt := aiSettings.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `You are a helpful customer support assistant for {{site_name}}. Use the following knowledge base content to answer questions accurately.

Knowledge Base Context:
{{context}}

Customer Question: {{enquiry}}

Provide a helpful, accurate response based on the context above. If the context doesn't contain relevant information, let the customer know you'll need to check and get back to them.`
	}
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{site_name}}", ko.String("app.site_name"))
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{context}}", contextStr)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{macros}}", macrosStr)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{enquiry}}", req.CustomerMessage)

	// Generate response using the system prompt with RAG context
	response, err := app.ai.CompletionWithSystemPrompt(systemPrompt, req.CustomerMessage)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]interface{}{
		"response": response,
		"sources":  results,
	})
}

// handleRAGFileUpload handles file uploads for RAG knowledge sources.
func handleRAGFileUpload(r *fastglue.Request) error {
	app := r.Context.(*App)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid form data", nil, envelope.InputError)
	}

	// Get file
	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, envelope.InputError)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error opening file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, envelope.GeneralError)
	}
	defer file.Close()

	// Read file content
	content := make([]byte, fileHeader.Size)
	if _, err := file.Read(content); err != nil {
		app.lo.Error("error reading file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, envelope.GeneralError)
	}

	// Get file extension and determine type
	filename := fileHeader.Filename
	var fileType string
	if strings.HasSuffix(strings.ToLower(filename), ".txt") {
		fileType = "txt"
	} else if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		fileType = "csv"
	} else if strings.HasSuffix(strings.ToLower(filename), ".json") {
		fileType = "json"
	} else {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported file type. Only .txt, .csv, and .json files are supported", nil, envelope.InputError)
	}

	// Get name from form or use filename
	var name string
	if names, ok := form.Value["name"]; ok && len(names) > 0 && strings.TrimSpace(names[0]) != "" {
		name = strings.TrimSpace(names[0])
	} else {
		name = filename
	}

	// Check if enabled
	enabled := true
	if enabledVals, ok := form.Value["enabled"]; ok && len(enabledVals) > 0 {
		enabled = enabledVals[0] == "true"
	}

	// Create file config
	config := models.FileConfig{
		Filename: filename,
		FileType: fileType,
		Content:  string(content),
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		app.lo.Error("error marshaling config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to process file", nil, envelope.GeneralError)
	}

	// Create the source
	source, err := app.rag.CreateSource(name, "file", configJSON, enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Sync immediately in background
	go func() {
		if err := app.ragSync.SyncSourceByID(source.ID); err != nil {
			app.lo.Error("error syncing file source", "source_id", source.ID, "error", err)
		}
	}()

	return r.SendEnvelope(source)
}
