package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai"
	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxTranscriptMessages bounds how many recent messages are fed to the LLM as context.
const maxTranscriptMessages = 50

type aiCompletionReq struct {
	PromptKey string `json:"prompt_key"`
	Content   string `json:"content"`
}

type providerUpdateReq struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
}

type generateReplyReq struct {
	ConversationUUID string `json:"conversation_uuid"`
	Instruction      string `json:"instruction"`
}

type copilotReq struct {
	ConversationUUID string                 `json:"conversation_uuid"`
	Messages         []aimodels.ChatMessage `json:"messages"`
}

type snippetReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Enabled bool   `json:"enabled"`
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
	resp, err := app.ai.Completion(req.PromptKey, req.Content)
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

// handleUpdateAIProvider sets the completion provider API key (inline reply-box prompt).
func handleUpdateAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req providerUpdateReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	if err := app.ai.UpdateProvider(req.Provider, req.APIKey); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
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
	item, err := app.ai.CreateKnowledgeBaseItem(req.Title, req.Content, req.Enabled)
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

	transcript := ""
	var extraTools []ai.Tool
	if req.ConversationUUID != "" {
		conv, err := enforceAIConversationAccess(r, req.ConversationUUID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		transcript = conversationTranscript(app, req.ConversationUUID)
		auser := r.RequestCtx.UserValue("user").(amodels.User)
		extraTools = contactHistoryTools(app, auser.ID, conv.ContactID, req.ConversationUUID)
	}
	resp, err := app.ai.GenerateReply(r.RequestCtx, transcript, req.Instruction, extraTools...)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
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
	if len(req.Messages) == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}

	convoContext := ""
	var extraTools []ai.Tool
	if req.ConversationUUID != "" {
		conv, err := enforceAIConversationAccess(r, req.ConversationUUID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		convoContext = conversationTranscript(app, req.ConversationUUID)
		auser := r.RequestCtx.UserValue("user").(amodels.User)
		extraTools = contactHistoryTools(app, auser.ID, conv.ContactID, req.ConversationUUID)
	}
	resp, err := app.ai.Copilot(r.RequestCtx, convoContext, req.Messages, extraTools...)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
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
	msgs, err := app.conversation.GetAllConversationMessages(uuid, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing})
	if err != nil {
		app.lo.Error("error building conversation transcript for AI", "error", err)
		return ""
	}
	if len(msgs) > maxTranscriptMessages {
		msgs = msgs[len(msgs)-maxTranscriptMessages:]
	}
	var b strings.Builder
	for _, msg := range msgs {
		role := "Agent"
		if msg.SenderType == cmodels.SenderTypeContact {
			role = "Customer"
		}
		text := strings.TrimSpace(msg.TextContent)
		if text == "" {
			continue
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(text)
		b.WriteString("\n")
	}
	return b.String()
}
