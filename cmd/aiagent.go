package main

import (
	"encoding/json"
	"mime/multipart"
	"path/filepath"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/aiagent/models"
	authmodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetAIAssistants returns all AI assistants.
func handleGetAIAssistants(r *fastglue.Request) error {
	app := r.Context.(*App)
	assistants, err := app.aiAgent.GetAssistants()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(assistants)
}

func handleGetAIAssistant(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	assistant, err := app.aiAgent.GetAssistant(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(assistant)
}

func handleCreateAIAssistant(r *fastglue.Request) error {
	app := r.Context.(*App)
	req, files, err := decodeAssistantForm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := validateAvatarFile(r, files); err != nil {
		return sendErrorEnvelope(r, err)
	}
	assistant, err := app.aiAgent.CreateAssistant(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := applyAssistantAvatar(r, assistant.UserID, files, req.RemoveAvatar); err != nil {
		if delErr := app.aiAgent.DeleteAssistant(assistant.ID); delErr != nil {
			app.lo.Error("error rolling back assistant after avatar failure", "assistant_id", assistant.ID, "error", delErr)
		}
		return sendErrorEnvelope(r, err)
	}
	assistant, err = app.aiAgent.GetAssistant(assistant.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(assistant)
}

func handleUpdateAIAssistant(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	req, files, err := decodeAssistantForm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := validateAvatarFile(r, files); err != nil {
		return sendErrorEnvelope(r, err)
	}
	assistant, err := app.aiAgent.UpdateAssistant(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := applyAssistantAvatar(r, assistant.UserID, files, req.RemoveAvatar); err != nil {
		return sendErrorEnvelope(r, err)
	}
	assistant, err = app.aiAgent.GetAssistant(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(assistant)
}

func handleDeleteAIAssistant(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.aiAgent.DeleteAssistant(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleAIAssistantPreview drafts a reply to a test message without any side effects.
func handleAIAssistantPreview(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req struct {
			Message string `json:"message"`
		}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	reply, sources, err := app.aiAgent.PreviewReply(r.RequestCtx, id, req.Message)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]any{"reply": reply, "sources": sources})
}

func handleGetAIAssistantStats(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	rangeDays, _ := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("range")))
	stats, err := app.aiAgent.GetAssistantStats(id, rangeDays)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

// decodeAssistantForm parses the multipart save payload: a JSON "data" field plus an optional avatar file.
func decodeAssistantForm(r *fastglue.Request) (amodels.Assistant, []*multipart.FileHeader, error) {
	app := r.Context.(*App)
	var req amodels.Assistant
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return req, nil, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil)
	}
	data := form.Value["data"]
	if len(data) == 0 {
		return req, nil, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil)
	}
	if err := json.Unmarshal([]byte(data[0]), &req); err != nil {
		return req, nil, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil)
	}
	return req, form.File["files"], nil
}

// applyAssistantAvatar uploads a new avatar for the assistant's paired user, or clears it when remove is set.
func applyAssistantAvatar(r *fastglue.Request, userID int, files []*multipart.FileHeader, remove bool) error {
	app := r.Context.(*App)
	if len(files) == 0 && !remove {
		return nil
	}
	user, err := app.user.GetAgentCachedOrLoad(userID)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		return uploadUserAvatar(r, user, files)
	}
	if user.AvatarURL.String == "" {
		return nil
	}
	if err := app.media.Delete(filepath.Base(user.AvatarURL.String)); err != nil {
		return err
	}
	if err := app.user.UpdateAvatar(userID, ""); err != nil {
		return err
	}
	app.user.InvalidateAgentCache(userID)
	return nil
}

type faqApproveReq struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type faqLearningReq struct {
	Enabled bool `json:"enabled"`
}

// handleGetAIFaqSuggestions returns FAQ suggestions filtered by ?status ("" returns all).
func handleGetAIFaqSuggestions(r *fastglue.Request) error {
	app := r.Context.(*App)
	status := string(r.RequestCtx.QueryArgs().Peek("status"))
	items, err := app.aiAgent.GetFAQSuggestions(status)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(items)
}

// handleApproveAIFaqSuggestion turns a suggestion into a knowledge base snippet and marks it approved.
func handleApproveAIFaqSuggestion(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req faqApproveReq
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	auser := r.RequestCtx.UserValue("user").(authmodels.User)
	if err := app.aiAgent.ApproveFAQSuggestion(id, req.Question, req.Answer, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleRejectAIFaqSuggestion marks a suggestion rejected without creating a snippet.
func handleRejectAIFaqSuggestion(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	auser := r.RequestCtx.UserValue("user").(authmodels.User)
	if err := app.aiAgent.RejectFAQSuggestion(id, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetAIFaqLearning returns whether FAQ learning from resolved conversations is enabled.
func handleGetAIFaqLearning(r *fastglue.Request) error {
	app := r.Context.(*App)
	b, err := app.setting.Get("ai_agent.faq_learning_enabled")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var enabled bool
	_ = json.Unmarshal(b, &enabled)
	return r.SendEnvelope(map[string]any{"enabled": enabled})
}

// handleUpdateAIFaqLearning toggles FAQ learning from resolved conversations.
func handleUpdateAIFaqLearning(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req faqLearningReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := app.setting.Update(map[string]any{"ai_agent.faq_learning_enabled": req.Enabled}); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
