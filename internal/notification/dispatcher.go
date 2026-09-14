package notifier

import (
	"github.com/abhinavxd/libredesk/internal/notification/channels"
	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/zerodha/logf"
)

type NotificationPreferenceStore interface {
	EnabledChannels(recipientIDs []int, nType models.NotificationType) (map[int][]models.NotificationChannel, error)
}

type Dispatcher struct {
	pipeline channels.Pipeline
	prefs    NotificationPreferenceStore
	lo       *logf.Logger
}

// DispatcherOpts contains options for creating a new Dispatcher.
type DispatcherOpts struct {
	Pipeline channels.Pipeline
	Prefs    NotificationPreferenceStore
	Lo       *logf.Logger
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(opts DispatcherOpts) *Dispatcher {
	return &Dispatcher{pipeline: opts.Pipeline, prefs: opts.Prefs, lo: opts.Lo}
}

func (d *Dispatcher) Send(n models.Notification) []models.DeliveryResult {
	recipientIDs := make([]int, len(n.Recipients))
	for i, recipient := range n.Recipients {
		recipientIDs[i] = recipient.UserID
	}
	enabled, err := d.prefs.EnabledChannels(recipientIDs, n.Type)
	if err != nil {
		d.lo.Error("error fetching notification preferences", "type", n.Type, "error", err)
		results := make([]models.DeliveryResult, len(recipientIDs))
		for i, recipientID := range recipientIDs {
			results[i].RecipientID = recipientID
		}
		return results
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
	return results
}
