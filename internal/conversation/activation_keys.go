package conversation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strings"

	"github.com/jakedolan443/fernmail/internal/activationkey"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jmoiron/sqlx"
)

// Activation keys travel through the outgoing pipeline as placeholders. The
// stored message, drafts, reviews, webhooks and previews only ever hold the
// placeholder; the key is bound to the message when it is queued and written
// into the email only in the copy handed to SMTP.

type activationKeyStore interface {
	Validate(placeholders []activationkey.Placeholder) error
	AllocateTx(tx *sqlx.Tx, in activationkey.Allocation) error
	MessageKeys(messageID int) (map[string]string, error)
	MaskSent(texts ...*string) error
	AppNames(ids []int) (map[int]string, error)
}

// SetActivationKeys connects Key Distribution. Without it, email that carries
// key placeholders is refused.
func (m *Manager) SetActivationKeys(store activationKeyStore) {
	m.activationKeys = store
}

func errDamagedPlaceholder() error {
	return envelope.NewError(envelope.InputError, "This email has a damaged activation key placeholder. Remove it and attach the key again.", nil)
}

func errKeysUnavailable() error {
	return envelope.NewError(envelope.InputError, "Activation keys aren't available right now.", nil)
}

// ValidateActivationKeys checks the placeholders in an email that is not
// being queued yet, such as a submission for review. Pool stock is checked
// only when the email is actually queued.
func (m *Manager) ValidateActivationKeys(content string) error {
	placeholders, err := activationkey.ParsePlaceholders(content)
	if err != nil {
		return errDamagedPlaceholder()
	}
	if len(placeholders) == 0 {
		return nil
	}
	if m.activationKeys == nil {
		return errKeysUnavailable()
	}
	return m.activationKeys.Validate(placeholders)
}

// renderKeepingPlaceholders applies best-effort template rendering, but only
// if it leaves the placeholders exactly as the author placed them: template
// data such as a contact's name must never add, drop or alter a key.
func (m *Manager) renderKeepingPlaceholders(data map[string]any, content string) (string, error) {
	before, err := activationkey.ParsePlaceholders(content)
	if err != nil {
		return "", errDamagedPlaceholder()
	}
	rendered := m.template.RenderString(data, content)
	after, err := activationkey.ParsePlaceholders(rendered)
	if err != nil || !slices.Equal(before, after) {
		return "", envelope.NewError(envelope.InputError, "Activation key placeholders can't be inside template variables or produced by them.", nil)
	}
	return rendered, nil
}

// allocateActivationKeys runs inside the message insert transaction, so an
// outgoing email and the keys it takes commit or roll back together.
func (m *Manager) allocateActivationKeys(tx *sqlx.Tx, message *models.Message) error {
	if message.Type != models.MessageOutgoing || message.SenderType != models.SenderTypeAgent {
		return nil
	}
	placeholders, err := activationkey.ParsePlaceholders(message.Content)
	if err != nil {
		return errDamagedPlaceholder()
	}
	if len(placeholders) == 0 {
		return nil
	}
	if message.Private {
		return envelope.NewError(envelope.InputError, "Activation keys can only be sent in replies, not in private notes.", nil)
	}
	if m.activationKeys == nil {
		return errKeysUnavailable()
	}
	var meta struct {
		To         []string `json:"to"`
		CC         []string `json:"cc"`
		BCC        []string `json:"bcc"`
		ApprovedBy int      `json:"approved_by"`
	}
	_ = json.Unmarshal(message.Meta, &meta)
	return m.activationKeys.AllocateTx(tx, activationkey.Allocation{
		Placeholders:   placeholders,
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		ApprovedBy:     meta.ApprovedBy,
		Recipients:     slices.Concat(meta.To, meta.CC, meta.BCC),
	})
}

// sendKeys maps one-time markers to the keys they stand for in one delivery.
type sendKeys map[string]string

// prepareActivationKeys swaps each placeholder for a random marker before the
// email template renders. Nothing in the message or template can produce a
// marker, and any mismatch between placeholders and bound keys fails the
// delivery rather than sending a bare placeholder or a stray key.
func (m *Manager) prepareActivationKeys(message *models.Message) (sendKeys, error) {
	placeholders, err := activationkey.ParsePlaceholders(message.Content)
	if err != nil {
		return nil, fmt.Errorf("reading activation key placeholders: %w", err)
	}
	var bound map[string]string
	if m.activationKeys != nil {
		if bound, err = m.activationKeys.MessageKeys(message.ID); err != nil {
			return nil, fmt.Errorf("loading activation keys: %w", err)
		}
	}
	if len(placeholders) == 0 && len(bound) == 0 {
		return nil, nil
	}
	if len(placeholders) != len(bound) {
		return nil, fmt.Errorf("email has %d activation key placeholders but %d keys", len(placeholders), len(bound))
	}
	keys := make(sendKeys, len(placeholders))
	markers := make(map[string]string, len(placeholders))
	for _, p := range placeholders {
		key, ok := bound[p.ID]
		if !ok {
			return nil, fmt.Errorf("activation key placeholder %s has no key", p.ID)
		}
		nonce := make([]byte, 16)
		if _, err := rand.Read(nonce); err != nil {
			return nil, err
		}
		marker := "LDKEY" + hex.EncodeToString(nonce)
		keys[marker] = key
		markers[p.ID] = marker
	}
	message.Content, err = activationkey.ReplacePlaceholders(message.Content, func(p activationkey.Placeholder) string {
		return markers[p.ID]
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// apply writes each key, in bold, where its marker landed after rendering.
func (k sendKeys) apply(message *models.Message) error {
	for marker, key := range k {
		if n := strings.Count(message.Content, marker); n != 1 {
			return fmt.Errorf("activation key appears %d times in the rendered email", n)
		}
		message.Content = strings.Replace(message.Content, marker, "<strong>"+html.EscapeString(key)+"</strong>", 1)
	}
	return nil
}

// maskIncomingKeys returns an incoming message's content and text as they
// may be shared beyond the thread, in the conversation preview and webhooks:
// a customer's reply often quotes the email that carried an activation key.
// If the keys can't be checked, the content is withheld rather than risk
// sharing one.
func (m *Manager) maskIncomingKeys(message models.Message) (content, text string) {
	content, text = message.Content, message.TextContent
	if message.Type != models.MessageIncoming || m.activationKeys == nil {
		return content, text
	}
	if err := m.activationKeys.MaskSent(&content, &text); err != nil {
		m.lo.Error("error masking activation keys", "message_id", message.ID, "error", err)
		return "", ""
	}
	return content, text
}

// MaskActivationKeys hides sent keys quoted in messages, for viewers who may
// not see them.
func (m *Manager) MaskActivationKeys(messages []models.Message) error {
	if m.activationKeys == nil || len(messages) == 0 {
		return nil
	}
	texts := make([]*string, 0, 2*len(messages))
	for i := range messages {
		texts = append(texts, &messages[i].Content, &messages[i].TextContent)
	}
	return m.activationKeys.MaskSent(texts...)
}

// summarizeReviewKeys labels submissions that carry activation keys.
func (m *Manager) summarizeReviewKeys(reviews []*models.Review) {
	appIDs := []int{}
	for _, review := range reviews {
		placeholders, err := activationkey.ParsePlaceholders(review.Content)
		if err != nil || len(placeholders) == 0 {
			continue
		}
		review.ActivationKeys = &models.ReviewKeySummary{Count: len(placeholders), AppID: placeholders[0].AppID}
		if !slices.Contains(appIDs, placeholders[0].AppID) {
			appIDs = append(appIDs, placeholders[0].AppID)
		}
	}
	if len(appIDs) == 0 || m.activationKeys == nil {
		return
	}
	names, err := m.activationKeys.AppNames(appIDs)
	if err != nil {
		m.lo.Error("error loading activation key app names", "error", err)
		return
	}
	for _, review := range reviews {
		if review.ActivationKeys != nil {
			review.ActivationKeys.AppName = names[review.ActivationKeys.AppID]
		}
	}
}
