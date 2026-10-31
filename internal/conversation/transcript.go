package conversation

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	transcriptTimeFormat = "Jan 02, 2006 03:04 PM MST"
	transcriptSeparator  = "------------------------------------------------------------"
	transcriptPDFMargin  = 36.0
	transcriptLineHeight = 14.0
)

var ErrTranscriptPDFUnsupportedCharacters = errors.New("transcript contains characters missing from the PDF font")

func (m *Manager) BuildTranscript(conversation models.Conversation, messages []models.Message, downloadedAt time.Time) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "%s #%s\n", m.i18n.T("globals.terms.conversation"), conversation.ReferenceNumber)
	if subject := conversation.Subject.String; subject != "" {
		fmt.Fprintf(&b, "%s: %s\n", m.i18n.T("globals.terms.subject"), subject)
	}
	if contact := conversation.Contact.FullName(); contact != "" {
		fmt.Fprintf(&b, "%s: %s\n", m.i18n.T("globals.terms.contact"), contact)
	}
	fmt.Fprintf(&b, "%s: %s\n", m.i18n.T("globals.terms.createdAt"), conversation.CreatedAt.UTC().Format(transcriptTimeFormat))
	fmt.Fprintf(&b, "%s: %s\n\n", m.i18n.T("globals.terms.downloadedAt"), downloadedAt.UTC().Format(transcriptTimeFormat))
	b.WriteString(transcriptSeparator + "\n")

	label := m.i18n.Tc("globals.terms.articleReference", 2 /* n */)
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
			fmt.Fprintf(&b, "%s: %s\n", m.i18n.Tc("globals.terms.attachment", len(message.Attachments)), strings.Join(names, ", "))
		}
	}
	return []byte(b.String())
}

// BuildTranscriptPDF wraps the text transcript across A4 pages with an embedded font.
func BuildTranscriptPDF(transcript []byte) ([]byte, error) {
	var pdf gopdf.GoPdf
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.SetMargins(transcriptPDFMargin, transcriptPDFMargin, transcriptPDFMargin, transcriptPDFMargin)
	missingGlyph := false
	if err := pdf.AddTTFFontDataWithOption("Go", goregular.TTF, gopdf.TtfOption{
		OnGlyphNotFound: func(rune) { missingGlyph = true },
	}); err != nil {
		return nil, err
	}
	if err := pdf.SetFont("Go", "" /* style */, 10 /* size */); err != nil {
		return nil, err
	}
	pdf.AddPage()
	width := gopdf.PageSizeA4.W - 2*transcriptPDFMargin
	text := strings.ReplaceAll(string(transcript), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	for paragraph := range strings.SplitSeq(text, "\n") {
		lines := []string{""}
		if paragraph != "" {
			var err error
			lines, err = pdf.SplitTextWithWordWrap(paragraph, width)
			if err != nil {
				return nil, err
			}
			if missingGlyph {
				return nil, ErrTranscriptPDFUnsupportedCharacters
			}
		}
		for _, line := range lines {
			if pdf.GetY()+transcriptLineHeight > gopdf.PageSizeA4.H-transcriptPDFMargin {
				pdf.AddPage()
			}
			if line != "" {
				if err := pdf.Cell(nil /* rectangle */, line); err != nil {
					return nil, err
				}
			}
			pdf.Br(transcriptLineHeight)
		}
	}
	return pdf.GetBytesPdfReturnErr()
}

func (m *Manager) senderTypeLabel(senderType string) string {
	switch senderType {
	case models.SenderTypeAgent:
		return m.i18n.T("globals.terms.agent")
	case models.SenderTypeContact:
		return m.i18n.T("globals.terms.contact")
	}
	return senderType
}

func transcriptAuthor(message models.Message) string {
	if sender := proactiveSender(message.Meta); sender != "" {
		return sender
	}
	return message.Author.FullName()
}
