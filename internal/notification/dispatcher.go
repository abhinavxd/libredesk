package notifier

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// WSHub defines the interface for the Websocket hub.
type WSHub interface {
	BroadcastMessage(msg wsmodels.BroadcastMessage)
}

type NotificationPreferenceStore interface {
	EnabledChannels(recipientIDs []int, nType models.NotificationType) map[int][]models.NotificationChannel
}

type PushDispatcher interface {
	Send(userID int, payload PushPayload)
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
	MessageUUID      string
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
	emailQueue   *EmailQueue
	wsHub        WSHub
	prefs        NotificationPreferenceStore
	push         PushDispatcher
	emailEnabled bool
	lo           *logf.Logger
}

// DispatcherOpts contains options for creating a new Dispatcher.
type DispatcherOpts struct {
	InApp        *UserNotificationManager
	EmailQueue   *EmailQueue
	WSHub        WSHub
	Prefs        NotificationPreferenceStore
	Push         PushDispatcher
	EmailEnabled bool
	Lo           *logf.Logger
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(opts DispatcherOpts) *Dispatcher {
	return &Dispatcher{
		inApp:        opts.InApp,
		emailQueue:   opts.EmailQueue,
		wsHub:        opts.WSHub,
		prefs:        opts.Prefs,
		push:         opts.Push,
		emailEnabled: opts.EmailEnabled,
		lo:           opts.Lo,
	}
}

// EnabledChannels returns the channels each recipient will actually be notified on, accounting for
// their preferences and the globally configured channels.
func (d *Dispatcher) EnabledChannels(recipientIDs []int, nType models.NotificationType) map[int][]models.NotificationChannel {
	enabled := d.prefs.EnabledChannels(recipientIDs, nType)
	if !d.emailEnabled || d.emailQueue == nil {
		for recipientID, channels := range enabled {
			channels = slices.DeleteFunc(channels, func(c models.NotificationChannel) bool {
				return c == models.NotificationChannelEmail
			})
			if len(channels) == 0 {
				delete(enabled, recipientID)
				continue
			}
			enabled[recipientID] = channels
		}
	}
	return enabled
}

// Send sends a notification through all configured channels.
// For each recipient: creates in-app notification (DB), broadcasts via Websocket,
// and sends email if Email field is provided.
func (d *Dispatcher) Send(n Notification) {
	d.dispatch(n, d.expandEmails(n), 0)
}

func (d *Dispatcher) SendAfter(n Notification, emailDelay time.Duration) {
	d.dispatch(n, d.expandEmails(n), emailDelay)
}

func (d *Dispatcher) SendWithEmails(n Notification, emails []EmailNotification) {
	d.dispatch(n, emails, 0)
}

func (d *Dispatcher) SendWithEmailsAfter(n Notification, emails []EmailNotification, emailDelay time.Duration) {
	d.dispatch(n, emails, emailDelay)
}

func (d *Dispatcher) expandEmails(n Notification) []EmailNotification {
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
	return emails
}

func (d *Dispatcher) dispatch(n Notification, emails []EmailNotification, emailDelay time.Duration) {
	if len(n.RecipientIDs) == 0 {
		return
	}
	enabled := d.EnabledChannels(n.RecipientIDs, n.Type)

	for i, recipientID := range n.RecipientIDs {
		var notificationID null.Int
		if slices.Contains(enabled[recipientID], models.NotificationChannelInApp) {
			if created := d.sendToRecipient(recipientID, n); created != nil {
				notificationID = null.IntFrom(created.ID)
			}
		}

		if i < len(emails) && len(emails[i].Recipients) > 0 &&
			slices.Contains(enabled[recipientID], models.NotificationChannelEmail) {
			e := emails[i]
			queued := queuedEmail{
				UserID:         recipientID,
				NotificationID: notificationID,
				Type:           n.Type,
				ConversationID: n.ConversationID,
				Recipient:      e.Recipients[0],
				Subject:        e.Subject,
				Content:        e.Content,
			}
			if emailDelay > 0 {
				d.emailQueue.SendAfter(queued, emailDelay)
			} else {
				d.emailQueue.Send(queued)
			}
		}

		if d.push != nil && slices.Contains(enabled[recipientID], models.NotificationChannelPush) {
			d.push.Send(recipientID, PushPayload{
				Title: n.Title,
				Body:  n.Body.String,
				Tag:   fmt.Sprintf("%s_%s_%s", n.Type, n.ConversationUUID, n.MessageUUID),
				URL:   pushRoute(string(n.Type), n.ConversationUUID, n.MessageUUID),
			})
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
