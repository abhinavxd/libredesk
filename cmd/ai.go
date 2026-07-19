package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai"
	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/aiagent"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxTranscriptMessages bounds how many recent messages are fed to the LLM as context.
const maxTranscriptMessages = 50

// maxCopilotHistoryMessages bounds how many persisted copilot turns are fed back to the LLM.
const maxCopilotHistoryMessages = 50

type aiCompletionReq struct {
	PromptKey string `json:"prompt_key"`
	Content   string `json:"content"`
}

type generateReplyReq struct {
	ConversationUUID string `json:"conversation_uuid"`
	Instruction      string `json:"instruction"`
}

type copilotReq struct {
	ConversationUUID string `json:"conversation_uuid"`
	Message          string `json:"message"`
	AssistantID      int    `json:"assistant_id"`
}

type summarizeReq struct {
	ConversationUUID string `json:"conversation_uuid"`
}

type suggestTagsReq struct {
	ConversationUUID string `json:"conversation_uuid"`
}

type snippetReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
}

type snippetImportReq struct {
	URL string `json:"url"`
}

// handleAICompletion runs a stored prompt over the supplied content (reply-box actions).
func handleAICompletion(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = aiCompletionReq{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	resp, err := app.ai.Completion(r.RequestCtx, req.PromptKey, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAIPrompts returns the agent-facing prompts.
func handleGetAIPrompts(r *fastglue.Request) error {
	app := r.Context.(*App)
	resp, err := app.ai.GetPrompts()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAIConfig returns the sanitized provider config for a type (completion/embedding).
func handleGetAIConfig(r *fastglue.Request) error {
	app := r.Context.(*App)
	providerType := r.RequestCtx.UserValue("type").(string)
	cfg, err := app.ai.GetProviderConfig(providerType)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(cfg)
}

// handleUpdateAIConfig updates the provider config for a type.
func handleUpdateAIConfig(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		providerType = r.RequestCtx.UserValue("type").(string)
		req          aimodels.ProviderConfig
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := app.ai.UpdateProviderConfig(providerType, req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleTestAIConfig makes one live provider request with the submitted config.
func handleTestAIConfig(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		providerType = r.RequestCtx.UserValue("type").(string)
		req          aimodels.ProviderConfig
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := app.ai.TestProviderConfig(providerType, req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetAITools returns all custom tools (auth secrets masked).
func handleGetAITools(r *fastglue.Request) error {
	app := r.Context.(*App)
	tools, err := app.ai.GetTools()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tools)
}

func handleGetAITool(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	tool, err := app.ai.GetTool(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tool)
}

func handleCreateAITool(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req aimodels.Tool
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	tool, err := app.ai.CreateTool(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tool)
}

func handleUpdateAITool(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req aimodels.Tool
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	tool, err := app.ai.UpdateTool(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tool)
}

func handleDeleteAITool(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.ai.DeleteTool(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetAISnippets returns the knowledge base snippets.
func handleGetAISnippets(r *fastglue.Request) error {
	app := r.Context.(*App)
	items, err := app.ai.GetKnowledgeBaseItems()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(items)
}

func handleCreateAISnippet(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req snippetReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	item, err := app.ai.CreateKnowledgeBaseItem(req.Title, req.Content, aimodels.KnowledgeSourceManual, "", req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(item)
}

func handleUpdateAISnippet(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req snippetReq
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	item, err := app.ai.UpdateKnowledgeBaseItem(id, req.Title, req.Content, req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(item)
}

// handleImportAISnippetFromURL fetches a page and adds its readable content as a snippet.
func handleImportAISnippetFromURL(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req snippetImportReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	item, err := app.ai.ImportKnowledgeBaseFromURL(r.RequestCtx, req.URL)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(item)
}

func handleDeleteAISnippet(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.ai.DeleteKnowledgeBaseItem(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleAIGenerateReply drafts a reply to a conversation using the agentic loop.
func handleAIGenerateReply(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req generateReplyReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	transcript := ""
	var conv *cmodels.Conversation
	if req.ConversationUUID != "" {
		conv, err = enforceAIConversationAccess(r, req.ConversationUUID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		transcript = conversationTranscript(app, req.ConversationUUID)
	}
	resp, err := app.ai.GenerateReply(r.RequestCtx, transcript, req.Instruction, ai.ToolContext{}, agentSurfaceTools(app, user, conv))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if strings.TrimSpace(resp) == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(resp)
}

// handleAISummarizeConversation summarizes a conversation and posts the summary as the requesting agent's private note.
func handleAISummarizeConversation(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req summarizeReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if req.ConversationUUID == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if _, err := enforceAIConversationAccess(r, req.ConversationUUID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	transcript := conversationTranscript(app, req.ConversationUUID)
	if strings.TrimSpace(transcript) == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("ai.summarizeEmptyConversation"), nil))
	}
	summary, err := app.ai.Summarize(r.RequestCtx, transcript)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if strings.TrimSpace(summary) == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	note := "**" + app.i18n.T("ai.summaryNoteTitle") + "**\n\n" + summary
	if _, err := app.conversation.SendPrivateNote(nil, auser.ID, req.ConversationUUID, stringutil.Markdown2HTML(note), nil); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleAISuggestTags suggests up to 3 existing tags for a conversation. Read-only: it never applies them.
func handleAISuggestTags(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req suggestTagsReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if req.ConversationUUID == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if _, err := enforceAIConversationAccess(r, req.ConversationUUID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	tags, err := app.tag.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(tags) == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("ai.noTagsConfigured"), nil))
	}

	transcript := conversationTranscript(app, req.ConversationUUID)
	if strings.TrimSpace(transcript) == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("ai.tagSuggestEmptyConversation"), nil))
	}

	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}

	suggestions, err := app.ai.SuggestTags(r.RequestCtx, transcript, names)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(suggestions)
}

// handleAICopilot answers an agent's copilot chat message.
func handleAICopilot(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req copilotReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" || req.ConversationUUID == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	conv, err := enforceAIConversationAccess(r, req.ConversationUUID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	persona := ""
	if req.AssistantID > 0 {
		assistant, err := app.aiAgent.GetAssistant(req.AssistantID)
		if err != nil || !assistant.Enabled {
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("ai.assistantUnavailable"), nil))
		}
		persona = aiagent.BuildCopilotPersona(assistant)
	}

	saved, err := app.ai.GetCopilotMessages(conv.ID, auser.ID, maxCopilotHistoryMessages)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	history := make([]aimodels.ChatMessage, 0, len(saved)+1)
	for _, m := range saved {
		history = append(history, aimodels.ChatMessage{Role: m.Role, Content: m.Content})
	}
	history = append(history, aimodels.ChatMessage{Role: aimodels.RoleUser, Content: req.Message})

	convoContext := conversationTranscript(app, req.ConversationUUID)
	resp, err := app.ai.Copilot(r.RequestCtx, convoContext, history, ai.ToolContext{}, agentSurfaceTools(app, user, conv), persona)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if strings.TrimSpace(resp) == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Copilot answers in markdown; store and return HTML so the panel, reply editor and private notes consume it as is.
	resp = stringutil.Markdown2HTML(resp)
	// Persist the exchange only after a successful reply, so a failed or empty call leaves no orphaned turn.
	if err := app.ai.SaveCopilotMessage(conv.ID, auser.ID, aimodels.RoleUser, req.Message); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.ai.SaveCopilotMessage(conv.ID, auser.ID, aimodels.RoleAssistant, resp); err != nil {
		app.lo.Error("error saving copilot reply", "error", err)
	}
	return r.SendEnvelope(resp)
}

// handleGetCopilotMessages returns the requesting agent's persisted copilot chat for a conversation.
func handleGetCopilotMessages(r *fastglue.Request) error {
	app := r.Context.(*App)
	uuid := string(r.RequestCtx.QueryArgs().Peek("conversation_uuid"))
	if uuid == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	conv, err := enforceAIConversationAccess(r, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	msgs, err := app.ai.GetCopilotMessages(conv.ID, auser.ID, maxCopilotHistoryMessages)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(msgs)
}

// handleClearCopilotMessages clears the requesting agent's copilot chat for a conversation.
func handleClearCopilotMessages(r *fastglue.Request) error {
	app := r.Context.(*App)
	uuid := string(r.RequestCtx.QueryArgs().Peek("conversation_uuid"))
	if uuid == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	conv, err := enforceAIConversationAccess(r, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	if err := app.ai.ClearCopilotMessages(conv.ID, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// enforceAIConversationAccess checks the requesting agent can access the conversation whose transcript is being fed to the LLM.
func enforceAIConversationAccess(r *fastglue.Request, uuid string) (*cmodels.Conversation, error) {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return nil, err
	}
	return enforceConversationAccess(app, uuid, user)
}

// conversationTranscript builds a plaintext transcript of a conversation's public messages for use as AI context.
func conversationTranscript(app *App, uuid string) string {
	private := false
	msgs, err := app.conversation.GetAllConversationMessages(uuid, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, maxTranscriptMessages)
	if err != nil {
		app.lo.Error("error building conversation transcript for AI", "error", err)
		return ""
	}
	return cmodels.Transcript(msgs, maxTranscriptMessages)
}
