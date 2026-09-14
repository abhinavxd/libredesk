// Package redisbackplane implements the ws.Backplane interface on top of Redis
// pub/sub, relaying WebSocket broadcast envelopes between Libredesk instances so
// real-time updates work when more than one instance is running.
package redisbackplane

import (
	"context"
	"time"

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

// backplaneChannelSize buffers delivered payloads to absorb short bursts.
const backplaneChannelSize = 1024

// receiveErrorBackoff paces retries after a transient receive error so a
// persistent failure can't hot-loop.
const receiveErrorBackoff = time.Second

// Subscribe returns a channel of payloads published by any instance. The returned
// channel is closed when ctx is cancelled.
//
// It reads with PubSub.ReceiveMessage rather than PubSub.Channel so slow delivery
// backpressures the Redis socket read directly. PubSub.Channel interposes its own
// buffered goroutine that silently drops messages after a send timeout when the
// consumer lags — unacceptable for control envelopes (e.g. KickUser). The Hub's
// consumer delivers to local clients without blocking, so blocking here only slows
// the reader under sustained overload instead of losing messages.
//
// Redis pub/sub is not durable: envelopes published while this subscriber is
// disconnected/reconnecting are not redelivered. A durable transport (e.g. Redis
// Streams) would be needed if control envelopes must survive a disconnect.
func (b *Backplane) Subscribe(ctx context.Context) (<-chan []byte, error) {
	pubsub := b.rdb.Subscribe(ctx, b.channel)
	out := make(chan []byte, backplaneChannelSize)
	b.lo.Debug("ws backplane subscribed", "channel", b.channel)
	go func() {
		defer close(out)
		defer pubsub.Close()
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				b.lo.Error("ws backplane receive failed, retrying", "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(receiveErrorBackoff):
				}
				continue
			}
			select {
			case out <- []byte(msg.Payload):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
