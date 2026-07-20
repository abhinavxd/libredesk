package aiagent

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai"
	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/aiagent/models"
	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	statusmodels "github.com/abhinavxd/libredesk/internal/conversation/status/models"
	imageutil "github.com/abhinavxd/libredesk/internal/image"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

const (
	channelEmail = "email"

	maxHistoryMessages = 30

	maxImagesPerMessage = 3
	maxImagesTotal      = 4
	maxImageBytes       = 8 << 20

	// confirmMarker is the line the model emits before its trailing confirmation question so it
	// can be split off and sent as a separate message.
	confirmMarker = "[[confirm]]"

	// typingRefreshInterval must stay under the widget's 5s typing expiry (TYPING_RECEIVE_TIMEOUT).
	typingRefreshInterval = 3 * time.Second

	// Timeout model: two nested clocks.
	//
	// Run clock (below): caps one full agent run - every completion call, retry, backoff
	// wait, and tool call combined. Livechat gets a tighter cap since the customer is
	// watching a typing indicator; email can afford to keep trying.
	//
	// Call clock (ai.overallRequestTimeout, 60s): each individual provider HTTP call gets
	// its own fresh 60s, and that call's retries must fit inside it. Custom tool calls get
	// their own 20s each.
	//
	// The call clock nests inside the run clock, so whichever expires first wins. E.g. on
	// livechat: completion 1 takes 50s (slow model + retries), so completion 2 starts with
	// only 40s of run budget - if it needs more, the run is cancelled and the conversation
	// hands off to a human.
	emailRunTimeout    = 3 * time.Minute
	livechatRunTimeout = 90 * time.Second
)

// nonActionableCategories are status categories the assistant skips; it replies only to open
// conversations, never waiting (e.g. snoozed) or resolved ones. Keyed by category, not status name,
// so admin-defined custom statuses are covered too.
var nonActionableCategories = map[string]bool{
	statusmodels.CategoryWaiting:  true,
	statusmodels.CategoryResolved: true,
}

// Run starts the response worker pool.
func (m *Manager) Run(ctx context.Context, workers int) {
	if workers <= 0 {
		workers = 1
	}
	for range workers {
		go m.worker(ctx)
	}
	miners := min(workers, 2)
	for range miners {
		go m.miningWorker(ctx)
	}
	<-ctx.Done()
}

func (m *Manager) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case convID := <-m.queue:
			m.handleWithRecover(ctx, convID)
			m.markDone(convID)
		}
	}
}

// handleWithRecover runs handle and recovers from panics so a single bad run (LLM chain or tool
// execution) can't crash the whole process and take down every channel.
func (m *Manager) handleWithRecover(ctx context.Context, convID int) {
	defer func() {
		if r := recover(); r != nil {
			m.lo.Error("recovered from panic in ai agent worker", "conversation_id", convID, "panic", r)
		}
	}()
	m.handle(ctx, convID)
}

// HandleConversationEvent enqueues a response when the assignee is an AI assistant.
func (m *Manager) HandleConversationEvent(conversationID, assigneeUserID int) {
	if conversationID == 0 || assigneeUserID == 0 {
		return
	}
	if !m.isAssistantUser(assigneeUserID) {
		return
	}
	m.enqueue(conversationID)
}

func (m *Manager) enqueue(convID int) {
	m.mu.Lock()
	if m.inflight[convID] {
		// A response is already running; remember to run again so a message that arrives mid-response
		// still gets answered.
		m.pending[convID] = true
		m.mu.Unlock()
		return
	}
	m.inflight[convID] = true
	m.mu.Unlock()

	select {
	case m.queue <- convID:
	default:
		m.mu.Lock()
		delete(m.inflight, convID)
		delete(m.pending, convID)
		m.mu.Unlock()
		m.lo.Warn("ai agent queue full, dropping response job", "conversation_id", convID)
	}
}

func (m *Manager) markDone(convID int) {
	m.mu.Lock()
	requeue := m.pending[convID]
	delete(m.pending, convID)
	delete(m.inflight, convID)
	m.mu.Unlock()
	if requeue {
		m.enqueue(convID)
	}
}

func (m *Manager) handle(ctx context.Context, convID int) {
	conv, err := m.convo.GetConversation(convID, "", "")
	if err != nil {
		m.lo.Error("error fetching conversation for ai agent", "conversation_id", convID, "error", err)
		return
	}
	if !conv.AssignedUserID.Valid {
		return
	}
	assistant, err := m.GetAssistantByUserID(int(conv.AssignedUserID.Int))
	if err != nil {
		if err != sql.ErrNoRows {
			m.lo.Error("error fetching assistant for ai agent", "conversation_id", convID, "user_id", conv.AssignedUserID.Int, "error", err)
		}
		return
	}
	m.lo.Debug("ai agent handling conversation", "conversation_id", convID, "conversation_uuid", conv.UUID, "assistant_id", assistant.ID, "assistant", assistant.Name, "status", conv.Status.String, "enabled", assistant.Enabled)
	if !assistant.Enabled {
		return
	}
	if nonActionableCategories[conv.StatusCategory.String] {
		return
	}

	private := false
	msgs, err := m.convo.GetAllConversationMessages(conv.UUID, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, conversation.MaxAllMessages)
	if err != nil {
		m.lo.Error("error fetching messages for ai agent", "conversation_uuid", conv.UUID, "error", err)
		return
	}
	// Only respond to a fresh inbound customer message, never to the assistant's own or automated replies.
	if !lastIsInboundContact(msgs) {
		return
	}
	// The reply and any identity-scoped tools act as the primary contact. An email thread can carry
	// messages from other participants (CC'd, or joined via plus-address, each a distinct contact), so
	// only act on a turn the primary contact authored - otherwise a participant's message could drive
	// tool actions under the contact's identity. Anyone else gets a human.
	if msgs[len(msgs)-1].SenderID != conv.ContactID {
		m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffOtherParticipant"))
		return
	}
	// Turn cap is per engagement (since the assistant was last assigned), so a human reassigning a
	// capped conversation to the assistant gives it a fresh budget instead of bouncing straight back.
	var turns int
	if err := m.q.CountAITurns.Get(&turns, conv.ID, assistant.UserID); err != nil {
		m.lo.Error("error counting ai turns", "conversation_id", conv.ID, "error", err)
		m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffError"))
		return
	}
	if turns >= assistant.MaxTurns {
		m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffMaxTurns"))
		return
	}

	systemPrompt := buildSystemPrompt(assistant)
	if len(contactFieldLines(conv.Contact)) == 0 {
		systemPrompt += "\n\n" + noContactIdentityNote
	}
	history := m.buildHistory(msgs, conv.ContactID)
	// Keep customer-controlled data (contact fields, subject, attributes) out of the system prompt; it
	// stays in a user-role block so it ranks below the assistant's instructions, not beside them.
	if block := customerContextBlock(conv); block != "" {
		history = append([]aimodels.ChatMessage{{Role: aimodels.RoleUser, Content: block}}, history...)
	}
	m.lo.Debug("ai agent running", "conversation_uuid", conv.UUID, "history_messages", len(history), "turns", turns)

	// A JWT livechat contact is trusted by login; everyone else (email channel, anonymous visitor)
	// is trusted only within an OTP verification window. Read live so mid-turn verification counts.
	verified := func() bool {
		if conv.InboxChannel != channelEmail && conv.Contact.Type == umodels.UserTypeContact {
			return true
		}
		return m.isConversationVerified(conv.UUID)
	}
	// Snapshot for the run-start registration decisions (one Redis read); tctx still gets the live
	// closure so mid-turn verification is picked up per tool call.
	runVerified := verified()
	m.lo.Debug("ai agent verification state", "conversation_uuid", conv.UUID, "channel", conv.InboxChannel, "contact_type", conv.Contact.Type, "has_email", conv.Contact.Email.String != "", "verified", runVerified)

	outcome := &runOutcome{}
	tools := []ai.Tool{
		&searchKnowledgeTool{m: m},
		&resolveTool{m: m, conv: conv, outcome: outcome},
	}
	if assistant.HandoffEnabled {
		tools = append(tools, &handoffTool{m: m, conv: conv, assistant: assistant, outcome: outcome})
	}
	if recent := m.recentContactConversations(conv, runVerified); len(recent) > 0 {
		systemPrompt += fmt.Sprintf("\n\nThis customer has %d other conversation(s) from the last %d days. Call get_previous_conversations if the current issue might be a follow-up or related to them.", len(recent), recentConversationDays)
		tools = append(tools, &previousConversationsTool{m: m, conversations: recent})
	}

	// Offer the verification tools only while unverified. set_contact_email is visitor-only and only
	// when there's no email on file to send the code to.
	if !runVerified {
		offerSetEmail := conv.Contact.Type == umodels.UserTypeVisitor && conv.Contact.Email.String == ""
		tools = append(tools, &sendEmailVerificationTool{m: m, conv: &conv})
		tools = append(tools, &checkEmailVerificationTool{m: m, conv: &conv})
		if offerSetEmail {
			tools = append(tools, &setContactEmailTool{m: m, conv: &conv})
		}
		systemPrompt += "\n\n" + verificationNote
		m.lo.Debug("ai agent offering verification tools", "conversation_uuid", conv.UUID, "set_contact_email_offered", offerSetEmail)
	}

	tctx := ai.ToolContext{
		ContactExternalID: conv.Contact.ExternalUserID.String,
		ContactEmail:      conv.Contact.Email.String,
		Verified:          verified,
	}
	timeout := emailRunTimeout
	if conv.InboxChannel != channelEmail {
		timeout = livechatRunTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stopTyping := m.keepTyping(conv.UUID)
	answer, err := m.ai.RunAgentWithTools(runCtx, systemPrompt, history, aiRunMaxSteps, tctx, assistant.ToolIDs, false, tools)
	stopTyping()
	if err != nil {
		m.lo.Error("error running ai agent", "conversation_uuid", conv.UUID, "error", err)
		if !outcome.handedOff {
			m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffError"))
		}
		return
	}
	// The assistant escalated; the handoff tool already reassigned and noted it.
	if outcome.handedOff {
		m.lo.Debug("ai agent handed off", "conversation_uuid", conv.UUID)
		return
	}
	// The model's text answer is the reply to the customer. Handoff and resolve are separate tool actions.
	answer, confirm := splitConfirmation(strings.TrimSpace(answer))
	// Email gets one message; separate chat-style bubbles only suit the widget.
	if conv.InboxChannel == channelEmail && confirm != "" {
		answer, confirm = answer+"\n\n"+confirm, ""
	}
	if answer != "" {
		m.lo.Debug("ai agent replying", "conversation_uuid", conv.UUID, "reply_len", len(answer), "resolved", outcome.resolved)
		if err := m.postReply(conv, assistant, answer, nil); err != nil {
			m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffError"))
			return
		}
	}
	if confirm != "" {
		if err := m.postReply(conv, assistant, confirm, map[string]any{"is_confirmation": true}); err != nil {
			m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffError"))
			return
		}
	}
	if outcome.resolved && (answer != "" || turns > 0) {
		m.resolve(conv, assistant)
		return
	}
	if answer == "" {
		m.lo.Debug("ai agent no answer, handing off", "conversation_uuid", conv.UUID)
		m.handoff(conv, assistant, m.i18n.T("ai.agent.handoffNoAnswer"))
	}
}

// resolve runs after the reply is queued so the CSAT survey sent on resolve follows the answer.
func (m *Manager) resolve(conv cmodels.Conversation, assistant models.Assistant) {
	if err := m.convo.UpdateConversationStatus(conv.UUID, 0, cmodels.StatusResolved, "", m.actorUser(assistant)); err != nil {
		m.lo.Error("error resolving conversation", "conversation_uuid", conv.UUID, "error", err)
		return
	}
	m.recordEvent(assistant.ID, conv.ID, "resolve")
}

// keepTyping re-broadcasts typing until the returned stop func runs; the widget clears the indicator 5s after the last event.
func (m *Manager) keepTyping(conversationUUID string) func() {
	m.convo.BroadcastTypingToWidgetClientsOnly(conversationUUID, true)
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(typingRefreshInterval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				m.convo.BroadcastTypingToWidgetClientsOnly(conversationUUID, true)
			}
		}
	}()
	return func() {
		close(stop)
		m.convo.BroadcastTypingToWidgetClientsOnly(conversationUUID, false)
	}
}

func (m *Manager) postReply(conv cmodels.Conversation, assistant models.Assistant, text string, meta map[string]any) error {
	var to []string
	if conv.InboxChannel == channelEmail && conv.Contact.Email.String != "" {
		to = []string{conv.Contact.Email.String}
	}
	if meta == nil {
		meta = map[string]any{}
	}
	if _, err := m.convo.QueueReply(nil, conv.InboxID, assistant.UserID, conv.ContactID, conv.UUID, stringutil.Markdown2HTML(text), to, nil, nil, meta); err != nil {
		m.lo.Error("error sending assistant reply", "conversation_uuid", conv.UUID, "error", err)
		return err
	}
	return nil
}

// handoff notes the reason and moves the conversation to the fallback team, or unassigns if none is set.
func (m *Manager) handoff(conv cmodels.Conversation, assistant models.Assistant, reason string) {
	actor := m.actorUser(assistant)
	note := strings.TrimSpace(reason)
	if note == "" {
		note = m.i18n.T("ai.agent.handoffDefault")
	}
	if _, err := m.convo.SendPrivateNote(nil, assistant.UserID, conv.UUID, note, nil); err != nil {
		m.lo.Error("error posting handoff note", "conversation_uuid", conv.UUID, "error", err)
	}
	movedToTeam := false
	if assistant.FallbackTeamID.Valid {
		if err := m.convo.UpdateConversationTeamAssignee(conv.UUID, int(assistant.FallbackTeamID.Int), actor); err != nil {
			m.lo.Error("error assigning fallback team", "conversation_uuid", conv.UUID, "error", err)
		} else {
			movedToTeam = true
		}
		// A same-team assignment keeps the assigned user, and a failed one removes nothing, so
		// re-check who is assigned instead of trusting the pre-run snapshot.
		fresh, err := m.convo.GetConversation(conv.ID, "", "")
		if err == nil && (!fresh.AssignedUserID.Valid || int(fresh.AssignedUserID.Int) != assistant.UserID) {
			m.recordEvent(assistant.ID, conv.ID, "handoff")
			return
		}
	}
	if err := m.convo.RemoveConversationAssignee(conv.UUID, cmodels.AssigneeTypeUser, actor); err != nil {
		m.lo.Error("error unassigning assistant on handoff", "conversation_uuid", conv.UUID, "error", err)
		if !movedToTeam {
			return
		}
	}
	m.recordEvent(assistant.ID, conv.ID, "handoff")
}

func (m *Manager) actorUser(a models.Assistant) umodels.User {
	return umodels.User{ID: a.UserID, FirstName: a.Name, Type: umodels.UserTypeAIAssistant}
}

// PreviewReply runs the assistant against a test message with no side effects and returns the reply and the knowledge base items it was grounded in.
func (m *Manager) PreviewReply(ctx context.Context, assistantID int, message string) (string, []models.PreviewSource, error) {
	a, err := m.GetAssistant(assistantID)
	if err != nil {
		return "", nil, err
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return "", nil, nil
	}
	history := []aimodels.ChatMessage{{Role: aimodels.RoleUser, Content: message}}
	var hits []aimodels.SearchResult
	tools := []ai.Tool{&searchKnowledgeTool{m: m, collect: func(rs []aimodels.SearchResult) { hits = append(hits, rs...) }}}
	// Preview is search-only: no custom tools (empty allowed set), no built-in, no side effects.
	answer, err := m.ai.RunAgentWithTools(ctx, buildSystemPrompt(a), history, aiRunMaxSteps, ai.ToolContext{}, []int{}, false, tools)
	if err != nil {
		return "", nil, err
	}
	main, confirm := splitConfirmation(strings.TrimSpace(answer))
	if confirm != "" {
		main += "\n\n" + confirm
	}
	return main, m.previewSources(hits), nil
}

// previewSources dedupes search hits by knowledge base item, keeping each item's best score.
func (m *Manager) previewSources(hits []aimodels.SearchResult) []models.PreviewSource {
	sources := []models.PreviewSource{}
	best := map[int]int{}
	for _, h := range hits {
		if idx, ok := best[h.SourceID]; ok {
			sources[idx].Score = max(sources[idx].Score, h.Score)
			continue
		}
		item, err := m.ai.GetKnowledgeBaseItem(h.SourceID)
		if err != nil {
			continue
		}
		best[h.SourceID] = len(sources)
		sources = append(sources, models.PreviewSource{ID: item.ID, Title: item.Title, Score: h.Score})
	}
	return sources
}

func lastIsInboundContact(msgs []cmodels.Message) bool {
	if len(msgs) == 0 {
		return false
	}
	last := msgs[len(msgs)-1]
	return last.Type == cmodels.MessageIncoming && last.SenderType == cmodels.SenderTypeContact
}

func (m *Manager) buildHistory(msgs []cmodels.Message, contactID int) []aimodels.ChatMessage {
	// Tools act as the primary contact, so a CC'd participant's message must never enter the prompt
	// as a trusted user turn - it could inject instructions that drive those tools under the
	// contact's identity. Keep the contact's own messages and the agent's replies; drop other contacts.
	kept := make([]cmodels.Message, 0, len(msgs))
	for _, msg := range msgs {
		if msg.SenderType == cmodels.SenderTypeContact && msg.SenderID != contactID {
			continue
		}
		kept = append(kept, msg)
	}
	msgs = kept
	if len(msgs) > maxHistoryMessages {
		msgs = msgs[len(msgs)-maxHistoryMessages:]
	}
	vision := m.ai.VisionEnabled()
	m.lo.Debug("ai agent building history", "messages", len(msgs), "vision", vision)
	// Spend the image budget newest-first so the message being answered never loses its
	// attachments to older ones.
	allowedImages := map[string]bool{}
	if vision {
		imagesLeft := maxImagesTotal
		for i := len(msgs) - 1; i >= 0 && imagesLeft > 0; i-- {
			if msgs[i].SenderType != cmodels.SenderTypeContact {
				continue
			}
			perMsg := 0
			for _, att := range msgs[i].Attachments {
				if imagesLeft <= 0 || perMsg >= maxImagesPerMessage {
					break
				}
				if !strings.HasPrefix(att.ContentType, "image/") || att.Size > maxImageBytes {
					continue
				}
				allowedImages[att.UUID] = true
				perMsg++
				imagesLeft--
			}
		}
	}
	history := make([]aimodels.ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		role := aimodels.RoleAssistant
		if msg.SenderType == cmodels.SenderTypeContact {
			role = aimodels.RoleUser
		}
		text := strings.TrimSpace(msg.TextContent)

		var images []aimodels.ChatImage
		var markers []string
		// Only the customer's own attachments are worth showing the assistant.
		if role == aimodels.RoleUser {
			for _, att := range msg.Attachments {
				if !strings.HasPrefix(att.ContentType, "image/") {
					m.lo.Debug("ai agent attachment not an image", "uuid", att.UUID, "content_type", att.ContentType)
					markers = append(markers, unreadableFileMarker(att))
					continue
				}
				if !allowedImages[att.UUID] {
					m.lo.Debug("ai agent image not sent", "uuid", att.UUID, "size", att.Size, "vision", vision)
					markers = append(markers, unreadableImageMarker(att))
					continue
				}
				img, ok := m.encodeAttachmentImage(att)
				if !ok {
					markers = append(markers, unreadableImageMarker(att))
					continue
				}
				m.lo.Debug("ai agent attached image", "uuid", att.UUID, "content_type", att.ContentType, "size", att.Size)
				images = append(images, img)
			}
		}

		if text == "" && len(images) == 0 && len(markers) == 0 {
			continue
		}
		if len(markers) > 0 {
			if text != "" {
				text += "\n"
			}
			text += strings.Join(markers, "\n")
		}
		history = append(history, aimodels.ChatMessage{Role: role, Content: text, Images: images})
	}
	return history
}

// recentContactConversations lists the contact's other recent conversations, only once verified so
// past-conversation PII never reaches an unverified/self-claimed identity.
func (m *Manager) recentContactConversations(conv cmodels.Conversation, verified bool) []models.RecentConversation {
	if !verified {
		return nil
	}
	recent := []models.RecentConversation{}
	if err := m.q.GetRecentContactConvos.Select(&recent, conv.ContactID, conv.ID, recentConversationDays, maxRecentConversations); err != nil {
		m.lo.Error("error fetching recent contact conversations for ai agent", "conversation_uuid", conv.UUID, "error", err)
		return nil
	}
	return recent
}

func (m *Manager) encodeAttachmentImage(att attachment.Attachment) (aimodels.ChatImage, bool) {
	blob, err := m.media.GetBlob(att.UUID)
	if err != nil {
		m.lo.Error("error reading attachment for ai agent", "uuid", att.UUID, "error", err)
		return aimodels.ChatImage{}, false
	}
	data, mediaType, err := imageutil.EncodeForLLM(blob)
	if err != nil {
		m.lo.Warn("could not encode attachment image for ai agent", "uuid", att.UUID, "error", err)
		return aimodels.ChatImage{}, false
	}
	m.lo.Debug("ai agent encoded image", "uuid", att.UUID, "raw_bytes", len(blob), "encoded_b64", len(data))
	return aimodels.ChatImage{MediaType: mediaType, Data: data}, true
}

func unreadableFileMarker(att attachment.Attachment) string {
	return fmt.Sprintf("[The customer attached a file %q (%s) that cannot be read here. Ask them to share the relevant details as text if it matters.]", att.Name, att.ContentType)
}

func unreadableImageMarker(att attachment.Attachment) string {
	return fmt.Sprintf("[The customer attached an image %q that you cannot view. Ask them to describe it if it matters.]", att.Name)
}

func splitConfirmation(answer string) (string, string) {
	idx := strings.LastIndex(answer, confirmMarker)
	if idx == -1 {
		return answer, ""
	}
	main := strings.TrimSpace(strings.ReplaceAll(answer[:idx], confirmMarker, ""))
	confirm := strings.TrimSpace(answer[idx+len(confirmMarker):])
	if main == "" {
		return confirm, ""
	}
	return main, confirm
}
