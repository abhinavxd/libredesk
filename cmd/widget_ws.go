package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	realip "github.com/ferluci/fast-realip"

	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/fasthttp/websocket"
	"github.com/zerodha/fastglue"
)

const (
	WidgetMsgTypeJoin      = "join"
	WidgetMsgTypeTyping    = "typing"
	WidgetMsgTypePing      = "ping"
	WidgetMsgTypePong      = "pong"
	WidgetMsgTypeError     = "error"
	WidgetMsgTypeJoined    = "joined"
	WidgetMsgTypePageVisit = "page_visit"

	pageVisitRedisKeyPrefix = "page_visits:"
	maxPageVisits           = 20
	pageVisitTTL            = 24 * time.Hour
	wsReadDeadline          = 20 * time.Second
	wsWriteDeadline         = 10 * time.Second
	wsReadLimitBytes        = 64 * 1024

	// Per-connection minimum intervals between inbound frames of each kind.
	// The HTTP upgrade is rate-limited, but inbound frames aren't, so a single
	// connection can otherwise drive unbounded DB/Redis work and agent fan-out.
	// Values are chosen to be just loose enough that no legitimate frontend
	// cadence is ever throttled.
	wsMinIntervalTyping    = 50 * time.Millisecond
	wsMinIntervalPageVisit = 1 * time.Second
	wsMinIntervalPing      = 1 * time.Second
)

type WidgetMessage struct {
	Type  string          `json:"type"`
	Token string          `json:"token,omitempty"`
	Data  json.RawMessage `json:"data"`
}

type WidgetInboxJoinRequest struct {
	InboxID string `json:"inbox_id"`
}

type WidgetTypingData struct {
	ConversationUUID string `json:"conversation_uuid"`
	IsTyping         bool   `json:"is_typing"`
}

type WidgetPageVisitData struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// safeConn wraps a WebSocket connection with a mutex for concurrent-safe writes
// and a per-connection rate tracker for inbound frames.
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex

	rateMu sync.Mutex
	lastAt map[string]time.Time
}

// WriteJSON and WriteMessage set a deadline first; a peer that stops reading would otherwise block the caller while holding sc.mu.
func (sc *safeConn) WriteJSON(v any) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))
	if err := sc.conn.WriteJSON(v); err != nil {
		sc.conn.Close()
		return err
	}
	return nil
}

func (sc *safeConn) WriteMessage(msgType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))
	if err := sc.conn.WriteMessage(msgType, data); err != nil {
		sc.conn.Close()
		return err
	}
	return nil
}

// allow throttles abusive clients that flood typing/page_visit/ping frames.
func (sc *safeConn) allow(kind string, minInterval time.Duration) bool {
	sc.rateMu.Lock()
	defer sc.rateMu.Unlock()
	if sc.lastAt == nil {
		sc.lastAt = make(map[string]time.Time)
	}
	now := time.Now()
	if last, ok := sc.lastAt[kind]; ok && now.Sub(last) < minInterval {
		return false
	}
	sc.lastAt[kind] = now
	return true
}

func handleWidgetWS(r *fastglue.Request) error {
	var app = r.Context.(*App)

	clientIP := realip.FromRequest(r.RequestCtx)

	if err := widgetUpgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		conn.SetReadLimit(wsReadLimitBytes)
		sc := &safeConn{conn: conn}

		var (
			client    *livechat.Client
			liveChat  *livechat.LiveChat
			inboxUUID string
			userID    int
		)

		defer func() {
			conn.Close()
			if client != nil && liveChat != nil {
				liveChat.RemoveClient(client)
				client.CloseChannel()
			}
		}()

		for {
			conn.SetReadDeadline(time.Now().Add(wsReadDeadline))

			// Checked after the refresh: CloseChannel marks the client before disconnect() expires the deadline, so
			// either this sees it or the expiry landed after the refresh and ReadJSON returns immediately.
			if client != nil && client.IsClosed() {
				break
			}

			var msg WidgetMessage
			if err := conn.ReadJSON(&msg); err != nil {
				app.lo.Debug("widget websocket connection closed", "error", err)
				break
			}

			switch msg.Type {
			case WidgetMsgTypeJoin:
				// Clean up previous client on re-join.
				if client != nil && liveChat != nil {
					liveChat.RemoveClient(client)
					client.CloseChannel()
				}
				client, liveChat, inboxUUID, userID = nil, nil, "", 0

				joinedClient, joinedLiveChat, joinedInboxUUID, joinedUserID, err := handleInboxJoin(app, sc, msg.Data, msg.Token, clientIP)
				if err != nil {
					app.lo.Error("error handling widget join", "error", err)
					sendWidgetError(sc, "Failed to join conversation")
					continue
				}
				client = joinedClient
				liveChat = joinedLiveChat
				inboxUUID = joinedInboxUUID
				userID = joinedUserID

			case WidgetMsgTypeTyping:
				if userID == 0 || inboxUUID == "" {
					continue
				}
				if !sc.allow(WidgetMsgTypeTyping, wsMinIntervalTyping) {
					continue
				}
				handleWidgetTyping(app, msg.Data, userID)

			case WidgetMsgTypePageVisit:
				if userID > 0 && sc.allow(WidgetMsgTypePageVisit, wsMinIntervalPageVisit) {
					handleWidgetPageVisit(app, msg.Data, userID)
				}

			case WidgetMsgTypePing:
				if !sc.allow(WidgetMsgTypePing, wsMinIntervalPing) {
					continue
				}
				if userID > 0 {
					wasOffline, err := app.user.UpdateLastActive(userID)
					if err != nil {
						app.lo.Error("error updating user last active timestamp", "user_id", userID, "error", err)
					} else if wasOffline {
						app.conversation.BroadcastContactUpdate(userID, map[string]any{"availability_status": "online"})
					}
				}

				if err := sc.WriteJSON(WidgetMessage{Type: WidgetMsgTypePong}); err != nil {
					app.lo.Error("error writing pong to widget client", "error", err)
				}
			}
		}
	}); err != nil {
		app.lo.Error("error upgrading widget websocket connection", "error", err)
	}
	return nil
}

func handleInboxJoin(app *App, sc *safeConn, data json.RawMessage, token, clientIP string) (*livechat.Client, *livechat.LiveChat, string, int, error) {
	var joinData WidgetInboxJoinRequest
	if err := json.Unmarshal(data, &joinData); err != nil {
		return nil, nil, "", 0, fmt.Errorf("invalid join data: %w", err)
	}

	inbox, err := app.inbox.GetDBRecord(joinData.InboxID)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("inbox not found: %w", err)
	}
	if !inbox.Enabled {
		return nil, nil, "", 0, fmt.Errorf("inbox is not enabled")
	}

	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err == nil {
		if len(config.BlockedIPs) > 0 && httputil.IsIPBlocked(clientIP, config.BlockedIPs) {
			return nil, nil, "", 0, fmt.Errorf("IP address is blocked")
		}
	}

	session, err := loadSession(app, token, config)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("session token validation failed: %w", err)
	}
	if session.InboxID != inbox.ID {
		return nil, nil, "", 0, fmt.Errorf("session does not belong to this inbox")
	}

	// Verify user exists and is enabled.
	user, err := app.user.Get(session.UserID, "", []string{})
	if err != nil || !user.Enabled {
		return nil, nil, "", 0, fmt.Errorf("user not found or disabled")
	}

	lcInbox, err := app.inbox.Get(inbox.ID)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("live chat inbox not found: %w", err)
	}

	liveChat, ok := lcInbox.(*livechat.LiveChat)
	if !ok {
		return nil, nil, "", 0, fmt.Errorf("inbox is not a live chat inbox")
	}

	userIDStr := fmt.Sprintf("%d", user.ID)
	// fasthttp makes Close a no-op on a hijacked conn; expiring the read deadline is what drops it.
	client, err := liveChat.AddClient(userIDStr, func() {
		sc.conn.SetReadDeadline(time.Now())
	})
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("adding client to live chat: %w", err)
	}

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				app.lo.Error("panic in widget ws forwarder", "panic", rec)
			}
		}()
		for msgData := range client.Channel {
			if err := sc.WriteMessage(websocket.TextMessage, msgData); err != nil {
				app.lo.Error("error forwarding message to widget client", "error", err)
				return
			}
		}
	}()

	if err := sc.WriteJSON(WidgetMessage{
		Type: WidgetMsgTypeJoined,
		Data: json.RawMessage(`{"message":"namaste!"}`),
	}); err != nil {
		liveChat.RemoveClient(client)
		client.CloseChannel()
		return nil, nil, "", 0, err
	}

	app.lo.Debug("widget client joined live chat", "user_id", userIDStr, "inbox_uuid", joinData.InboxID)

	return client, liveChat, joinData.InboxID, user.ID, nil
}

func handleWidgetTyping(app *App, data json.RawMessage, userID int) {
	var typingData WidgetTypingData
	if err := json.Unmarshal(data, &typingData); err != nil || typingData.ConversationUUID == "" {
		return
	}

	// userID was already validated during WS join.
	conversation, err := app.conversation.GetConversation(0, typingData.ConversationUUID, "")
	if err != nil || conversation.ContactID != userID {
		return
	}

	app.conversation.BroadcastTypingToConversation(typingData.ConversationUUID, typingData.IsTyping, false)
}

func sendWidgetError(sc *safeConn, message string) {
	data, _ := json.Marshal(map[string]string{"message": message})
	sc.WriteJSON(WidgetMessage{
		Type: WidgetMsgTypeError,
		Data: data,
	})
}

func handleWidgetPageVisit(app *App, data json.RawMessage, contactID int) {
	var visit WidgetPageVisitData
	if err := json.Unmarshal(data, &visit); err != nil || visit.URL == "" {
		return
	}

	if len(visit.URL) > 2048 {
		visit.URL = visit.URL[:2048]
	}
	if len(visit.Title) > 256 {
		visit.Title = visit.Title[:256]
	}

	parsedURL, err := url.Parse(visit.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return
	}

	redisCtx := context.Background()
	key := fmt.Sprintf("%s%d", pageVisitRedisKeyPrefix, contactID)

	// Skip if the most recent page visit has the same URL.
	if latest, err := app.redis.LIndex(redisCtx, key, 0).Result(); err == nil {
		var lastVisit map[string]string
		if json.Unmarshal([]byte(latest), &lastVisit) == nil && lastVisit["url"] == visit.URL {
			return
		}
	}

	entry, _ := json.Marshal(map[string]string{
		"url":   visit.URL,
		"title": visit.Title,
		"time":  time.Now().UTC().Format(time.RFC3339),
	})

	pipe := app.redis.Pipeline()
	pipe.LPush(redisCtx, key, string(entry))
	pipe.LTrim(redisCtx, key, 0, maxPageVisits-1)
	pipe.Expire(redisCtx, key, pageVisitTTL)
	lrangeCmd := pipe.LRange(redisCtx, key, 0, maxPageVisits-1)
	pipe.Exec(redisCtx)

	entries, err := lrangeCmd.Result()
	if err != nil {
		return
	}
	pages := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		var p map[string]string
		if err := json.Unmarshal([]byte(e), &p); err == nil {
			pages = append(pages, p)
		}
	}
	app.conversation.BroadcastContactUpdate(contactID, map[string]any{"page_visits": pages})
}

func getPageVisitsFromRedis(app *App, contactID int) []map[string]string {
	redisCtx := context.Background()
	key := fmt.Sprintf("%s%d", pageVisitRedisKeyPrefix, contactID)
	entries, err := app.redis.LRange(redisCtx, key, 0, maxPageVisits-1).Result()
	if err != nil {
		return nil
	}
	pages := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		var p map[string]string
		if err := json.Unmarshal([]byte(e), &p); err == nil {
			pages = append(pages, p)
		}
	}
	return pages
}
