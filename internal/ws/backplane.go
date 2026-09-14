package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// publishTimeout bounds a single backplane publish so a stalled backplane can't
// block the goroutine that produced the broadcast (often an HTTP handler).
const publishTimeout = 5 * time.Second

// envelopeKind identifies how a relayed broadcast must be delivered on each instance.
type envelopeKind string

const (
	// envelopeUsers delivers to specific users' clients, or to every client when Users is empty.
	envelopeUsers envelopeKind = "users"
	// envelopeConvs delivers to the subscribers of a set of conversations.
	envelopeConvs envelopeKind = "convs"
	// envelopeKick closes every connection of the given users.
	envelopeKick envelopeKind = "kick"
)

// envelope is the wire format relayed between instances: an instance delivers a
// broadcast locally and publishes it as an envelope, and every instance replays
// envelopes from others against its own client registry.
type envelope struct {
	// Origin is the publisher's instance ID; receivers skip their own envelopes.
	Origin string          `json:"origin"`
	Kind   envelopeKind    `json:"kind"`
	Users  []int           `json:"users,omitempty"`
	Convs  []string        `json:"convs,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// Backplane relays broadcast envelopes between Libredesk instances. Implementations
// must deliver every published payload to all subscribed instances, including the
// publisher (the Hub deduplicates its own envelopes by origin). Redis pub/sub is the
// reference implementation.
type Backplane interface {
	// Publish sends a payload to all instances.
	Publish(ctx context.Context, payload []byte) error
	// Subscribe returns a channel of payloads published by any instance. The
	// channel is closed when ctx is cancelled.
	Subscribe(ctx context.Context) (<-chan []byte, error)
}

// SetBackplane wires a cross-instance backplane. Call ConsumeBackplane afterwards to
// start delivering envelopes from other instances. With no backplane set the Hub stays
// single-instance: broadcasts reach only locally connected clients.
func (h *Hub) SetBackplane(b Backplane) {
	h.backplane = b
}

// publish relays an envelope to other instances. Publish failures are logged and
// swallowed so a backplane outage degrades to local-only delivery rather than
// dropping the broadcast that was already delivered locally.
func (h *Hub) publish(env envelope) {
	if h.backplane == nil {
		return
	}
	env.Origin = h.instanceID
	payload, err := json.Marshal(env)
	if err != nil {
		h.lo.Error("marshalling ws backplane envelope failed", "kind", env.Kind, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()
	if err := h.backplane.Publish(ctx, payload); err != nil {
		h.lo.Error("publishing to ws backplane failed", "kind", env.Kind, "error", err)
	}
}

// ConsumeBackplane blocks and delivers envelopes produced by other instances to
// local clients until ctx is cancelled. Run it in its own goroutine.
func (h *Hub) ConsumeBackplane(ctx context.Context) {
	if h.backplane == nil {
		return
	}
	ch, err := h.backplane.Subscribe(ctx)
	if err != nil {
		h.lo.Error("subscribing to ws backplane failed", "error", err)
		return
	}
	for payload := range ch {
		var env envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			h.lo.Error("unmarshalling ws backplane envelope failed", "error", err)
			continue
		}
		// The origin already delivered this locally.
		if env.Origin == h.instanceID {
			continue
		}
		switch env.Kind {
		case envelopeUsers:
			h.broadcastMessageLocal(models.BroadcastMessage{Data: env.Data, Users: env.Users})
		case envelopeConvs:
			h.broadcastToConversationsLocal(env.Convs, env.Data)
		case envelopeKick:
			for _, userID := range env.Users {
				h.kickUserLocal(userID)
			}
		default:
			h.lo.Warn("unknown ws backplane envelope kind", "kind", env.Kind)
		}
	}
}

// broadcastToConversationsLocal delivers data to every locally-connected subscriber
// of the given conversations, each client at most once.
func (h *Hub) broadcastToConversationsLocal(uuids []string, data []byte) {
	if len(uuids) == 0 {
		return
	}
	seen := make(map[*Client]struct{})
	for _, uuid := range uuids {
		for _, c := range h.ListSubscribers(uuid) {
			seen[c] = struct{}{}
		}
	}
	for c := range seen {
		c.SendMessage(data, websocket.TextMessage)
	}
}
