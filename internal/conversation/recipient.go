package conversation

import (
	"encoding/json"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// makeRecipients computes the recipients for a given conversation ID using the last message in the conversation.
func (m *Manager) makeRecipients(conversationID int, contactEmail, inboxEmail, inboxReplyTo string) (to, cc, bcc []string, err error) {
	lastMessage, err := m.getLatestMessage(conversationID, []string{models.MessageIncoming, models.MessageOutgoing}, []string{models.MessageStatusReceived, models.MessageStatusSent}, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetching message for makeRecipients: %w", err)
	}

	var meta struct {
		From []string `json:"from"`
		To   []string `json:"to"`
		CC   []string `json:"cc"`
		BCC  []string `json:"bcc"`
	}
	if err = json.Unmarshal(lastMessage.Meta, &meta); err != nil {
		return nil, nil, nil, err
	}

	isIncoming := lastMessage.Type == models.MessageIncoming
	to, cc, bcc = stringutil.ComputeRecipients(
		meta.From, meta.To, meta.CC, meta.BCC, contactEmail, inboxEmail, inboxReplyTo, isIncoming,
	)
	return
}
