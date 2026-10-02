package channels

import (
	"cmp"
	"slices"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/volatiletech/null/v9"
)

type Provider interface {
	Channel() models.NotificationChannel
	Send(Delivery) Result
}

type Delivery struct {
	Recipient      models.Recipient
	Notification   models.Notification
	NotificationID null.Int
}

type Result struct {
	Sent           bool
	NotificationID null.Int
}

type Pipeline struct {
	providers []Provider
}

func NewPipeline(providers ...Provider) Pipeline {
	providers = slices.Clone(providers)
	slices.SortStableFunc(providers, func(a, b Provider) int {
		return cmp.Compare(channelOrder(a.Channel()), channelOrder(b.Channel()))
	})
	return Pipeline{providers: providers}
}

func (p Pipeline) Send(delivery Delivery, enabled []models.NotificationChannel) []models.NotificationChannel {
	var sent []models.NotificationChannel
	for _, provider := range p.providers {
		if !slices.Contains(enabled, provider.Channel()) {
			continue
		}
		result := provider.Send(delivery)
		if result.Sent {
			sent = append(sent, provider.Channel())
		}
		if result.NotificationID.Valid {
			delivery.NotificationID = result.NotificationID
		}
	}
	return sent
}

func channelOrder(channel models.NotificationChannel) int {
	switch channel {
	case models.NotificationChannelInApp:
		return 0
	case models.NotificationChannelEmail:
		return 1
	case models.NotificationChannelPush:
		return 2
	default:
		return 3
	}
}
