package notifier

import (
	"encoding/json"
	"slices"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// WSHub defines the interface for the Websocket hub.
type WSHub interface {
	BroadcastMessage(msg wsmodels.BroadcastMessage)
}

// Notification represents a notification to be sent through all channels.
type Notification struct {
	// Core notification fields
	Type           models.NotificationType
	RecipientIDs   []int
	Title          string
	Body           null.String
	ConversationID null.Int
	MessageID      null.Int
	ActorID        null.Int
	Meta           json.RawMessage

	// For Websocket broadcast
	ConversationUUID string
	ActorFirstName   string
	ActorLastName    string

	// Email fields (optional - if empty, no email sent)
	Email *EmailNotification
}

// EmailNotification holds email channel notification details.
type EmailNotification struct {
	Recipients []string
	Subject    string
	Content    string
}

// Dispatcher coordinates sending notifications through multiple channels: WS, DB, email.
type Dispatcher struct {
	inApp        *UserNotificationManager
	outbound     *Service
	wsHub        WSHub
	prefs        *PreferenceManager
	emailEnabled bool
	lo           *logf.Logger
}

// DispatcherOpts contains options for creating a new Dispatcher.
type DispatcherOpts struct {
	InApp        *UserNotificationManager
	Outbound     *Service
	WSHub        WSHub
	Prefs        *PreferenceManager
	EmailEnabled bool
	Lo           *logf.Logger
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(opts DispatcherOpts) *Dispatcher {
	return &Dispatcher{
		inApp:        opts.InApp,
		outbound:     opts.Outbound,
		wsHub:        opts.WSHub,
		prefs:        opts.Prefs,
		emailEnabled: opts.EmailEnabled,
		lo:           opts.Lo,
	}
}

// disabledChannels resolves the opted-out channels of every recipient. On a lookup failure email
// is suppressed for all recipients, an unreadable preference must not override an email opt-out.
func (d *Dispatcher) disabledChannels(n Notification) map[int][]models.NotificationChannel {
	disabled, err := d.prefs.DisabledChannels(n.RecipientIDs, n.Type)
	if err != nil {
		disabled = make(map[int][]models.NotificationChannel, len(n.RecipientIDs))
		for _, recipientID := range n.RecipientIDs {
			disabled[recipientID] = []models.NotificationChannel{models.NotificationChannelEmail}
		}
	}
	return disabled
}

// Send sends a notification through all configured channels.
// For each recipient: creates in-app notification (DB), broadcasts via Websocket,
// and sends email if Email field is provided.
func (d *Dispatcher) Send(n Notification) {
	var emails []EmailNotification
	if n.Email != nil {
		emails = make([]EmailNotification, len(n.RecipientIDs))
		for i := range n.RecipientIDs {
			var email string
			if i < len(n.Email.Recipients) {
				email = n.Email.Recipients[i]
			} else if len(n.Email.Recipients) == 1 {
				email = n.Email.Recipients[0] // Broadcast mode
			}
			if email == "" {
				continue
			}
			emails[i] = EmailNotification{
				Recipients: []string{email},
				Subject:    n.Email.Subject,
				Content:    n.Email.Content,
			}
		}
	}
	d.SendWithEmails(n, emails)
}

// SendWithEmails sends notifications where each recipient has their own email content.
// This is useful when email content is personalized per recipient.
func (d *Dispatcher) SendWithEmails(n Notification, emails []EmailNotification) {
	if len(n.RecipientIDs) == 0 {
		return
	}
	disabled := d.disabledChannels(n)

	for i, recipientID := range n.RecipientIDs {
		if !slices.Contains(disabled[recipientID], models.NotificationChannelInApp) {
			d.sendToRecipient(recipientID, n)
		}

		if d.outbound != nil && i < len(emails) && len(emails[i].Recipients) > 0 && d.emailEnabled &&
			!slices.Contains(disabled[recipientID], models.NotificationChannelEmail) {
			e := emails[i]
			d.sendEmail(recipientID, e.Recipients[0], e.Subject, e.Content, n.Type)
		}
	}
}

// sendToRecipient creates in-app notification and broadcasts via Websocket.
// Returns the created notification or nil if creation failed.
func (d *Dispatcher) sendToRecipient(recipientID int, n Notification) *models.UserNotification {
	notification, err := d.inApp.Create(
		recipientID,
		n.Type,
		n.Title,
		n.Body,
		n.ConversationID,
		n.MessageID,
		n.ActorID,
		n.Meta,
	)
	if err != nil {
		d.lo.Error("error creating in-app notification",
			"recipient_id", recipientID,
			"type", n.Type,
			"error", err)
		return nil
	}
	notification.ConversationUUID = null.StringFrom(n.ConversationUUID)
	notification.ActorFirstName = null.StringFrom(n.ActorFirstName)
	notification.ActorLastName = null.StringFrom(n.ActorLastName)
	d.broadcastNotification([]int{recipientID}, notification)
	return &notification
}

// sendEmail sends an email notification through the outbound service.
func (d *Dispatcher) sendEmail(recipientID int, email, subject, content string, nType models.NotificationType) {
	if err := d.outbound.Send(Message{
		RecipientEmails: []string{email},
		Subject:         subject,
		Content:         content,
		Provider:        ProviderEmail,
	}); err != nil {
		d.lo.Error("error sending email notification",
			"recipient_id", recipientID,
			"email", email,
			"type", nType,
			"error", err)
	}
}

// broadcastNotification broadcasts a notification via Websocket to specified users.
func (d *Dispatcher) broadcastNotification(userIDs []int, notification any) {
	if d.wsHub == nil {
		return
	}
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeNewNotification,
		Data: notification,
	}
	msgB, err := json.Marshal(message)
	if err != nil {
		d.lo.Error("error marshalling notification for Websocket", "error", err)
		return
	}
	d.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{
		Data:  msgB,
		Users: userIDs,
	})
}
