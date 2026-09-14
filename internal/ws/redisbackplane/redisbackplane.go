// Package redisbackplane implements the ws.Backplane interface on top of Redis
// pub/sub, relaying WebSocket broadcast envelopes between Libredesk instances so
// real-time updates work when more than one instance is running.
package redisbackplane

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

// Backplane relays broadcast envelopes over a single Redis pub/sub channel.
type Backplane struct {
	rdb     *redis.Client
	channel string
	lo      *logf.Logger
}

// New returns a Backplane that publishes to and subscribes from the given channel.
func New(rdb *redis.Client, channel string, lo *logf.Logger) *Backplane {
	return &Backplane{rdb: rdb, channel: channel, lo: lo}
}

// Publish sends payload to every subscribed instance.
func (b *Backplane) Publish(ctx context.Context, payload []byte) error {
	return b.rdb.Publish(ctx, b.channel, payload).Err()
}

// Subscribe returns a channel of payloads published by any instance. go-redis
// transparently reconnects the underlying subscription; the returned channel is
// closed when ctx is cancelled.
func (b *Backplane) Subscribe(ctx context.Context) (<-chan []byte, error) {
	pubsub := b.rdb.Subscribe(ctx, b.channel)
	out := make(chan []byte, 1024)
	go func() {
		defer close(out)
		defer pubsub.Close()
		msgs := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				select {
				case out <- []byte(msg.Payload):
				default:
					// Drop rather than stall the receive loop; a missed envelope
					// costs one instance a live update until the next event.
					b.lo.Warn("ws backplane consumer lagging, dropping message")
				}
			}
		}
	}()
	return out, nil
}
