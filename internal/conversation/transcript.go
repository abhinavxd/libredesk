package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

const (
	transcriptTimeFormat = "Jan 02, 2006 03:04 PM MST"
	transcriptSeparator  = "------------------------------------------------------------"
)

func (m *Manager) BuildTranscript(conversation models.Conversation, messages []models.Message, downloadedAt time.Time) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "%s #%s\n", m.i18n.Get("Conversation"), conversation.ReferenceNumber)
	if subject := conversation.Subject.String; subject != "" {
		fmt.Fprintf(&b, "%s: %s\n", m.i18n.Get("Subject"), subject)
	}
	if contact := conversation.Contact.FullName(); contact != "" {
		fmt.Fprintf(&b, "%s: %s\n", m.i18n.Get("Contact"), contact)
	}
	fmt.Fprintf(&b, "%s: %s\n", m.i18n.Get("Created at"), conversation.CreatedAt.UTC().Format(transcriptTimeFormat))
	fmt.Fprintf(&b, "%s: %s\n\n", m.i18n.Get("Downloaded at"), downloadedAt.UTC().Format(transcriptTimeFormat))
	b.WriteString(transcriptSeparator + "\n")

	label := m.i18n.GetN("Article reference", "Article references", 2 /* n */)
	references := m.lookupArticleReferences(messageMetas(messages)...)
	for i, message := range messages {
		content := message.TextContent
		if content == "" {
			content = stringutil.HTML2Text(message.Content)
		}
		if len(references[i]) > 0 {
			content += "\n\n" + stringutil.HTML2TextMarkdownLinks(articleReferencesHTML(references[i], label))
		}
		fmt.Fprintf(&b, "\n[%s] %s (%s):\n%s\n",
			message.CreatedAt.UTC().Format(transcriptTimeFormat),
			transcriptAuthor(message),
			m.senderTypeLabel(message.SenderType),
			content,
		)
		if len(message.Attachments) > 0 {
			names := make([]string, 0, len(message.Attachments))
			for _, attachment := range message.Attachments {
				names = append(names, attachment.Name)
			}
			fmt.Fprintf(&b, "%s: %s\n", m.i18n.GetN("Attachment",
				"Attachments",
				len(message.Attachments)), strings.Join(names, ", "))
		}
	}
	return []byte(b.String())
}

func (m *Manager) senderTypeLabel(senderType string) string {
	switch senderType {
	case models.SenderTypeAgent:
		return m.i18n.Get("Agent")
	case models.SenderTypeContact:
		return m.i18n.Get("Contact")
	}
	return senderType
}

func transcriptAuthor(message models.Message) string {
	if sender := proactiveSender(message.Meta); sender != "" {
		return sender
	}
	return message.Author.FullName()
}
