package guidedform

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	statusmodels "github.com/abhinavxd/libredesk/internal/conversation/status/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	gmodels "github.com/abhinavxd/libredesk/internal/guidedform/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

// progressAttrKey is the reserved conversation custom-attribute key guidedform uses to persist
// where a conversation is within its form's flow. It is never surfaced in the UI because it does
// not correspond to any defined custom attribute.
const progressAttrKey = "_guided_form_progress"

// nonActionableCategories mirrors aiagent: a conversation that's snoozed/waiting or already
// resolved should not have the bot post into it.
var nonActionableCategories = map[string]bool{
	statusmodels.CategoryWaiting:  true,
	statusmodels.CategoryResolved: true,
}

// HandleConversationEvent is notified whenever a conversation assigned to a user may need
// attention - a fresh inbound message, or a brand new assignment to that user. If the assignee
// is a guided-form bot, it advances (or starts) that conversation's flow. It recovers from
// panics so a bad form config can never take other channels down with it.
func (m *Manager) HandleConversationEvent(conversationID, assigneeUserID int) {
	if conversationID == 0 || assigneeUserID == 0 || !m.isFormBotUser(assigneeUserID) {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			m.lo.Error("recovered from panic in guided form handler", "conversation_id", conversationID, "panic", r)
		}
	}()
	m.handle(conversationID, assigneeUserID)
}

// SkipToHuman lets a visitor bail out of an in-progress guided form and reach a human directly,
// instead of being stuck typing until something matches a branch. It reuses the same handoff
// path as an error mid-flow: fall back to the form's team if set, else the unassigned queue.
func (m *Manager) SkipToHuman(conversationID int) error {
	conv, err := m.convo.GetConversation(conversationID, "", "")
	if err != nil {
		m.lo.Error("error fetching conversation for guided form skip", "conversation_id", conversationID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if !conv.AssignedUserID.Valid || !m.isFormBotUser(int(conv.AssignedUserID.Int)) {
		// Nothing to skip - already past the guided form (or never in one). Not an error.
		return nil
	}
	form, err := m.GetFormByUserID(int(conv.AssignedUserID.Int))
	if err != nil {
		m.lo.Error("error fetching guided form for skip", "conversation_id", conversationID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	attrs := decodeAttrs(conv.CustomAttributes)
	m.handoff(conv, form, attrs, m.i18n.T("admin.guidedForms.skippedByVisitor"))
	return nil
}

func (m *Manager) handle(conversationID, assigneeUserID int) {
	conv, err := m.convo.GetConversation(conversationID, "", "")
	if err != nil {
		m.lo.Error("error fetching conversation for guided form", "conversation_id", conversationID, "error", err)
		return
	}
	if !conv.AssignedUserID.Valid || int(conv.AssignedUserID.Int) != assigneeUserID {
		return
	}
	form, err := m.GetFormByUserID(assigneeUserID)
	if err != nil {
		if err != sql.ErrNoRows {
			m.lo.Error("error fetching guided form", "conversation_id", conversationID, "user_id", assigneeUserID, "error", err)
		}
		return
	}
	if !form.Enabled || nonActionableCategories[conv.StatusCategory.String] {
		return
	}

	attrs := decodeAttrs(conv.CustomAttributes)
	progress, hasProgress := decodeProgress(attrs)

	// Brand new to the flow: ask the first question and stop, there is nothing to match yet.
	if !hasProgress {
		step, ok := form.StepByID(form.StartStepID)
		if !ok {
			m.lo.Error("guided form has no valid start step", "form_id", form.ID)
			return
		}
		m.askStep(conv, form, step)
		m.saveProgress(conv, attrs, gmodels.Progress{FormID: form.ID, StepID: step.ID, Answers: map[string]any{}})
		return
	}

	// Progress belongs to a different (e.g. since-replaced) form; nothing sane to do but stop.
	if progress.FormID != form.ID {
		return
	}

	step, ok := form.StepByID(progress.StepID)
	if !ok {
		m.lo.Error("guided form progress points at unknown step, handing off", "form_id", form.ID, "step_id", progress.StepID)
		m.handoff(conv, form, attrs, m.i18n.T("globals.messages.somethingWentWrong"))
		return
	}

	// Only a fresh inbound message from the conversation's own contact advances the flow -
	// never the bot's own question, and never a CC'd/other participant's message.
	// GetConversationMessages returns newest-first, so reverse to chronological order before
	// scanning for "the latest one" - mirrors internal/aiagent/worker.go.
	private := false
	msgs, _, err := m.convo.GetConversationMessages(conv.UUID, 1, 20, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing})
	if err != nil {
		m.lo.Error("error fetching messages for guided form", "conversation_uuid", conv.UUID, "error", err)
		return
	}
	slices.Reverse(msgs)
	inbound := latestInboundContact(msgs)
	if inbound == nil || inbound.SenderID != conv.ContactID {
		return
	}
	// Guards against reprocessing the same answer twice if this handler is invoked more than
	// once for the same message (e.g. an assignment event and a message event landing together).
	if inbound.ID <= progress.LastMessageID {
		return
	}

	answer := strings.TrimSpace(messageText(inbound))
	if answer == "" {
		return
	}

	// A required step must satisfy its type's format before the flow advances; an optional one
	// accepts whatever was sent. Marks the message processed either way so it's never re-evaluated.
	if step.Required && !validAnswerFormat(step.Type, answer) {
		progress.LastMessageID = inbound.ID
		m.saveProgress(conv, attrs, progress)
		m.askInvalidAnswer(conv, form, step)
		return
	}

	if progress.Answers == nil {
		progress.Answers = map[string]any{}
	}
	saveKey := step.SaveAs
	if saveKey == "" {
		saveKey = step.ID
	}
	progress.Answers[saveKey] = answer
	progress.LastMessageID = inbound.ID
	// Mutates attrs in place for a conversation-scoped answer; a contact-scoped one is saved
	// separately on the contact's own row, so it never touches this conversation's attributes.
	m.applyAnswer(attrs, conv.ContactID, step, answer)

	nextStepID := matchBranch(step, answer)
	// No branch matched and no explicit default: fall through to the next step in order
	// (a plain linear form needs no branch config at all), unless the step explicitly ends
	// the form here, or there is no next step to fall through to.
	if nextStepID == "" && !step.EndsForm {
		if next, ok := form.NextStepInOrder(step.ID); ok {
			nextStepID = next.ID
		}
	}
	if nextStepID == "" {
		m.complete(conv, form, attrs)
		return
	}
	nextStep, ok := form.StepByID(nextStepID)
	if !ok {
		m.lo.Error("guided form branch points at unknown step", "form_id", form.ID, "next_step_id", nextStepID)
		m.handoff(conv, form, attrs, m.i18n.T("globals.messages.somethingWentWrong"))
		return
	}

	m.askStep(conv, form, nextStep)
	progress.StepID = nextStep.ID
	m.saveProgress(conv, attrs, progress)
}

// matchBranch returns the id of the next step for the given answer: the first branch whose
// pattern matches (case-insensitively), or the step's default, or "" if the step is terminal.
func matchBranch(step gmodels.Step, answer string) string {
	for _, b := range step.Branches {
		re, err := compileBranchPattern(b.Pattern)
		if err != nil {
			continue
		}
		if re.MatchString(answer) {
			return b.NextStepID
		}
	}
	return step.DefaultNextStepID
}

func compileBranchPattern(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i)" + pattern)
}

// askStep posts a step's question as an ordinary outgoing chat message from the bot identity.
// For a choice step, the options travel in meta so the widget can render them as quick-reply
// buttons (see guided_form_options); the question text also lists them inline as a fallback for
// any client that doesn't render the buttons.
func (m *Manager) askStep(conv cmodels.Conversation, form gmodels.Form, step gmodels.Step) {
	m.postQuestion(conv, form, step, "")
}

// postQuestion sends a step's question, optionally prefixed with a validation hint (used to
// re-prompt on an invalid required answer instead of advancing).
func (m *Manager) postQuestion(conv cmodels.Conversation, form gmodels.Form, step gmodels.Step, hint string) {
	question := step.Question
	if hint != "" {
		question = hint + "\n\n" + question
	}
	meta := map[string]any{"is_guided_form": true}
	if step.Type == gmodels.StepTypeChoice && len(step.Options) > 0 {
		question += "\n\n" + strings.Join(step.Options, " / ")
		meta["guided_form_options"] = step.Options
	}
	if _, err := m.convo.QueueReply(nil, conv.InboxID, form.UserID, conv.ContactID, conv.UUID, stringutil.Markdown2HTML(question), nil, nil, nil, meta); err != nil {
		m.lo.Error("error posting guided form question", "conversation_uuid", conv.UUID, "step_id", step.ID, "error", err)
	}
}

var phonePattern = regexp.MustCompile(`^[0-9+()\-\s]{6,20}$`)

// validAnswerFormat reports whether answer satisfies step type's format. text and choice accept
// anything non-empty (already guaranteed by the caller); email/phone/number are format-checked.
func validAnswerFormat(stepType, answer string) bool {
	switch stepType {
	case gmodels.StepTypeEmail:
		return stringutil.ValidEmail(answer)
	case gmodels.StepTypePhone:
		return phonePattern.MatchString(answer) && strings.ContainsAny(answer, "0123456789")
	case gmodels.StepTypeNumber:
		_, err := strconv.ParseFloat(strings.TrimSpace(answer), 64)
		return err == nil
	default:
		return true
	}
}

// invalidAnswerHintKey maps a step type to the i18n key for its re-prompt hint.
func invalidAnswerHintKey(stepType string) string {
	switch stepType {
	case gmodels.StepTypeEmail:
		return "admin.guidedForms.invalidEmail"
	case gmodels.StepTypePhone:
		return "admin.guidedForms.invalidPhone"
	case gmodels.StepTypeNumber:
		return "admin.guidedForms.invalidNumber"
	default:
		return "admin.guidedForms.invalidAnswer"
	}
}

// askInvalidAnswer re-prompts the same step with a short validation hint instead of advancing.
func (m *Manager) askInvalidAnswer(conv cmodels.Conversation, form gmodels.Form, step gmodels.Step) {
	m.postQuestion(conv, form, step, m.i18n.T(invalidAnswerHintKey(step.Type)))
}

// applyAnswer records a step's answer onto the custom attribute it was configured against, so it
// shows up wherever the equivalent pre-chat form field would (sidebar custom attributes). A
// conversation-scoped answer is written into attrs in place (the caller persists it once,
// alongside the progress state, to avoid two independent read-modify-writes racing on the same
// JSONB column); a contact-scoped answer is saved directly since it lives on a different row.
func (m *Manager) applyAnswer(attrs map[string]any, contactID int, step gmodels.Step, answer string) {
	if step.CustomAttributeID == 0 {
		return
	}
	attr, err := m.customAttribute.Get(step.CustomAttributeID)
	if err != nil {
		m.lo.Warn("guided form custom attribute not found", "custom_attribute_id", step.CustomAttributeID, "error", err)
		return
	}
	if attr.AppliesTo == gmodels.AppliesToConversation {
		attrs[attr.Key] = answer
		return
	}
	if err := m.user.SaveCustomAttributes(contactID, map[string]any{attr.Key: answer}, false); err != nil {
		m.lo.Error("error saving guided form answer to contact attributes", "contact_id", contactID, "error", err)
	}
}

// complete runs when the flow reaches a terminal step: it clears the progress marker (merging
// any conversation-scoped answer the caller already applied to attrs into the same write), posts
// the optional completion message, hands off per the form's configured action, and records the event.
func (m *Manager) complete(conv cmodels.Conversation, form gmodels.Form, attrs map[string]any) {
	delete(attrs, progressAttrKey)
	if err := m.convo.UpdateConversationCustomAttributes(conv.UUID, attrs); err != nil {
		m.lo.Error("error clearing guided form progress on completion", "conversation_uuid", conv.UUID, "error", err)
	}
	if form.CompletionMessage != "" {
		meta := map[string]any{"is_guided_form": true}
		if _, err := m.convo.QueueReply(nil, conv.InboxID, form.UserID, conv.ContactID, conv.UUID, stringutil.Markdown2HTML(form.CompletionMessage), nil, nil, nil, meta); err != nil {
			m.lo.Error("error posting guided form completion message", "conversation_uuid", conv.UUID, "error", err)
		}
	}

	actor := umodels.User{ID: form.UserID, FirstName: form.Name, Type: umodels.UserTypeGuidedFormBot}
	switch form.OnCompleteAction {
	case gmodels.CompleteActionAssistant:
		var assistantUserID int
		if err := m.db.Get(&assistantUserID, `SELECT user_id FROM ai_assistants WHERE id = $1`, form.OnCompleteAssistantID.Int); err != nil {
			m.lo.Error("error resolving ai assistant for guided form handoff", "form_id", form.ID, "error", err)
			m.recordEvent(form.ID, conv.ID, "completed")
			return
		}
		if err := m.convo.UpdateConversationUserAssignee(conv.UUID, assistantUserID, actor); err != nil {
			m.lo.Error("error assigning conversation to ai assistant after guided form", "conversation_uuid", conv.UUID, "error", err)
		}
	case gmodels.CompleteActionTeam:
		if err := m.convo.UpdateConversationTeamAssignee(conv.UUID, int(form.OnCompleteTeamID.Int), actor); err != nil {
			m.lo.Error("error assigning conversation to team after guided form", "conversation_uuid", conv.UUID, "error", err)
		}
		if err := m.convo.RemoveConversationAssignee(conv.UUID, cmodels.AssigneeTypeUser, actor); err != nil {
			m.lo.Error("error unassigning guided form bot after handoff", "conversation_uuid", conv.UUID, "error", err)
		}
	case gmodels.CompleteActionUnassign:
		if err := m.convo.RemoveConversationAssignee(conv.UUID, cmodels.AssigneeTypeUser, actor); err != nil {
			m.lo.Error("error unassigning guided form bot on completion", "conversation_uuid", conv.UUID, "error", err)
		}
	}
	m.recordEvent(form.ID, conv.ID, "completed")
}

// handoff is used for error paths mid-flow (e.g. a misconfigured form): fall back to the form's
// team if set, else drop the assignment so a human picks it up from the unassigned queue.
func (m *Manager) handoff(conv cmodels.Conversation, form gmodels.Form, attrs map[string]any, reason string) {
	delete(attrs, progressAttrKey)
	if err := m.convo.UpdateConversationCustomAttributes(conv.UUID, attrs); err != nil {
		m.lo.Error("error clearing guided form progress on handoff", "conversation_uuid", conv.UUID, "error", err)
	}
	actor := umodels.User{ID: form.UserID, FirstName: form.Name, Type: umodels.UserTypeGuidedFormBot}
	if _, err := m.convo.SendPrivateNote(nil, form.UserID, conv.UUID, reason, nil); err != nil {
		m.lo.Error("error posting guided form handoff note", "conversation_uuid", conv.UUID, "error", err)
	}
	if form.OnCompleteTeamID.Valid {
		if err := m.convo.UpdateConversationTeamAssignee(conv.UUID, int(form.OnCompleteTeamID.Int), actor); err != nil {
			m.lo.Error("error assigning fallback team on guided form error", "conversation_uuid", conv.UUID, "error", err)
		}
	}
	if err := m.convo.RemoveConversationAssignee(conv.UUID, cmodels.AssigneeTypeUser, actor); err != nil {
		m.lo.Error("error unassigning guided form bot on error", "conversation_uuid", conv.UUID, "error", err)
	}
	m.recordEvent(form.ID, conv.ID, "handoff")
}

func (m *Manager) saveProgress(conv cmodels.Conversation, attrs map[string]any, progress gmodels.Progress) {
	attrs[progressAttrKey] = progress
	if err := m.convo.UpdateConversationCustomAttributes(conv.UUID, attrs); err != nil {
		m.lo.Error("error saving guided form progress", "conversation_uuid", conv.UUID, "error", err)
	}
}

func decodeAttrs(raw json.RawMessage) map[string]any {
	attrs := map[string]any{}
	if len(raw) == 0 {
		return attrs
	}
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return map[string]any{}
	}
	return attrs
}

func decodeProgress(attrs map[string]any) (gmodels.Progress, bool) {
	raw, ok := attrs[progressAttrKey]
	if !ok {
		return gmodels.Progress{}, false
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return gmodels.Progress{}, false
	}
	var p gmodels.Progress
	if err := json.Unmarshal(b, &p); err != nil {
		return gmodels.Progress{}, false
	}
	return p, true
}

func latestInboundContact(msgs []cmodels.Message) *cmodels.Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == cmodels.MessageIncoming && msgs[i].SenderType == cmodels.SenderTypeContact {
			return &msgs[i]
		}
	}
	return nil
}

// messageText returns a message's plain-text content, stripping quoted reply chains.
func messageText(msg *cmodels.Message) string {
	if msg.ContentType == cmodels.ContentTypeHTML {
		if t := stringutil.HTML2TextNoQuotes(msg.Content); t != "" {
			return t
		}
		return stringutil.HTML2TextMarkdownLinks(msg.Content)
	}
	if t := stringutil.TrimPlainTextQuotes(msg.TextContent); t != "" {
		return t
	}
	return msg.TextContent
}
