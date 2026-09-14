package notifier

import (
	"fmt"

	"github.com/abhinavxd/libredesk/internal/notification/channels"
	"github.com/abhinavxd/libredesk/internal/notification/models"
)

type NotificationPreferenceStore interface {
	EnabledChannels(recipientIDs []int, nType models.NotificationType) (map[int][]models.NotificationChannel, error)
}

type Dispatcher struct {
	pipeline channels.Pipeline
	prefs    NotificationPreferenceStore
}

// DispatcherOpts contains options for creating a new Dispatcher.
type DispatcherOpts struct {
	Pipeline channels.Pipeline
	Prefs    NotificationPreferenceStore
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(opts DispatcherOpts) *Dispatcher {
	return &Dispatcher{pipeline: opts.Pipeline, prefs: opts.Prefs}
}

func (d *Dispatcher) Send(n models.Notification) ([]models.DeliveryResult, error) {
	recipientIDs := make([]int, len(n.Recipients))
	for i, recipient := range n.Recipients {
		recipientIDs[i] = recipient.UserID
	}
	enabled, err := d.prefs.EnabledChannels(recipientIDs, n.Type)
	if err != nil {
		return nil, fmt.Errorf("fetching notification preferences: %w", err)
	}

	results := make([]models.DeliveryResult, 0, len(n.Recipients))
	for _, recipient := range n.Recipients {
		results = append(results, models.DeliveryResult{
			RecipientID: recipient.UserID,
			Channels: d.pipeline.Send(
				channels.Delivery{Recipient: recipient, Notification: n},
				enabled[recipient.UserID],
			),
		})
	}
	return results, nil
}
