package email

import (
	"math/rand"
	"net/textproto"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email"
	"github.com/abhinavxd/libredesk/internal/inbox/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
)

// Email implements the MessageSender interface for sending emails.
type Email struct {
	lo        *logf.Logger
	from      string
	smtpPools []*smtppool.Pool
}

// Opts contains options for creating a new Email sender.
type Opts struct {
	Lo        *logf.Logger
	FromEmail string
}

// New initializes a new Email sender.
func New(smtpConfig []models.SMTPConfig, opts Opts) (*Email, error) {
	pools, err := email.NewSmtpPool(smtpConfig, nil)
	if err != nil {
		return nil, err
	}
	return &Email{
		lo:        opts.Lo,
		smtpPools: pools,
		from:      opts.FromEmail,
	}, nil
}

// Send sends a notification message via email.
func (e *Email) Send(msg notifier.Message) error {
	emailMessage := e.prepareEmail(msg.Subject, msg.Content, msg.RecipientEmails, msg)
	return e.send(emailMessage)
}

// Name returns the name of the provider.
func (e *Email) Name() string {
	return notifier.ProviderEmail
}

// send sends an email message.
func (e *Email) send(em smtppool.Email) error {
	srv := e.selectSmtpPool()
	return srv.Send(em)
}

// selectSmtpPool selects a random SMTP pool if multiple are available.
func (e *Email) selectSmtpPool() *smtppool.Pool {
	if len(e.smtpPools) > 1 {
		return e.smtpPools[rand.Intn(len(e.smtpPools))]
	}
	return e.smtpPools[0]
}

// prepareEmail prepares the email message with attachments and headers.
func (e *Email) prepareEmail(subject, content string, recipients []string, msg notifier.Message) smtppool.Email {
	var files []smtppool.Attachment
	if len(msg.Attachments) > 0 {
		files = e.prepareAttachments(msg.Attachments)
	}

	em := smtppool.Email{
		From:        e.from,
		To:          recipients,
		Subject:     subject,
		Attachments: files,
		Headers:     textproto.MIMEHeader{},
	}

	// Set content based on provided type
	switch msg.ContentType {
	case "plain":
		em.Text = []byte(content)
	default:
		em.HTML = []byte(content)
		if len(msg.AltContent) > 0 {
			em.Text = []byte(msg.AltContent)
		}
	}

	// Set any additional headers
	for headerKey, headerValue := range msg.Headers {
		em.Headers[headerKey] = headerValue
	}

	return em
}

// prepareAttachments prepares email attachments.
func (e *Email) prepareAttachments(attachments []attachment.Attachment) []smtppool.Attachment {
	files := make([]smtppool.Attachment, len(attachments))
	for i, f := range attachments {
		files[i] = smtppool.Attachment{
			Filename: f.Name,
			Header:   f.Header,
			Content:  append([]byte(nil), f.Content...),
		}
	}
	return files
}
