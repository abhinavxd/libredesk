// Package livechat implements a live chat inbox for handling real-time conversations.
package livechat

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	ErrClientNotConnected = fmt.Errorf("client not connected")
)

const (
	ChannelLiveChat       = "livechat"
	MaxConnectionsPerUser = 10

	HomeAppAnnouncement  = "announcement"
	HomeAppExternalLink  = "external_link"
)

type PreChatFormField struct {
	Key               string `json:"key"`
	Type              string `json:"type"`
	Label             string `json:"label"`
	Placeholder       string `json:"placeholder"`
	Required          bool   `json:"required"`
	Enabled           bool   `json:"enabled"`
	Order             int    `json:"order"`
	IsDefault         bool   `json:"is_default"`
	CustomAttributeID int    `json:"custom_attribute_id"`
}

// ContinuityConfig holds per-inbox conversation continuity settings.
type ContinuityConfig struct {
	OfflineThreshold    string `json:"offline_threshold"`
	MaxMessagesPerEmail int    `json:"max_messages_per_email"`
	MinEmailInterval    string `json:"min_email_interval"`
}

// Config holds the live chat inbox configuration.
type Config struct {
	BrandName        string `json:"brand_name"`
	WebsiteURL       string `json:"website_url"`
	DarkMode         bool   `json:"dark_mode"`
	ShowPoweredBy    bool   `json:"show_powered_by"`
	Language         string `json:"language"`
	FallbackLanguage string `json:"fallback_language"`
	Users            struct {
		AllowStartConversation           bool   `json:"allow_start_conversation"`
		PreventMultipleConversations     bool   `json:"prevent_multiple_conversations"`
		PreventReplyToClosedConversation bool   `json:"prevent_reply_to_closed_conversation"`
		StartConversationButtonText      string `json:"start_conversation_button_text"`
	} `json:"users"`
	Colors struct {
		Primary string `json:"primary"`
	} `json:"colors"`
	HomeScreen struct {
		HeaderTextColor string `json:"header_text_color"`
		Background      struct {
			Type          string `json:"type"`
			Color         string `json:"color"`
			GradientStart string `json:"gradient_start"`
			GradientEnd   string `json:"gradient_end"`
			ImageURL      string `json:"image_url"`
		} `json:"background"`
		FadeBackground bool `json:"fade_background"`
	} `json:"home_screen"`
	Features struct {
		Emoji      bool `json:"emoji"`
		FileUpload bool `json:"file_upload"`
	} `json:"features"`
	Launcher struct {
		Spacing struct {
			Side   int `json:"side"`
			Bottom int `json:"bottom"`
		} `json:"spacing"`
		LogoURL  string `json:"logo_url"`
		Position string `json:"position"`
		Color    string `json:"color"`
	} `json:"launcher"`
	LogoURL  string `json:"logo_url"`
	Visitors struct {
		AllowStartConversation           bool   `json:"allow_start_conversation"`
		PreventMultipleConversations     bool   `json:"prevent_multiple_conversations"`
		PreventReplyToClosedConversation bool   `json:"prevent_reply_to_closed_conversation"`
		StartConversationButtonText      string `json:"start_conversation_button_text"`
	} `json:"visitors"`
	NoticeBanner struct {
		Text    string `json:"text"`
		Enabled bool   `json:"enabled"`
	} `json:"notice_banner"`
	HomeApps []struct {
		Type        string `json:"type"`
		Title       string `json:"title,omitempty"`
		Description string `json:"description,omitempty"`
		ImageURL    string `json:"image_url,omitempty"`
		URL         string `json:"url"`
		Text        string `json:"text,omitempty"`
	} `json:"home_apps"`
	TrustedDomains                 []string         `json:"trusted_domains"`
	BlockedIPs                     []string         `json:"blocked_ips"`
	DirectToConversation           bool             `json:"direct_to_conversation"`
	GreetingMessage                string           `json:"greeting_message"`
	ChatIntroduction               string           `json:"chat_introduction"`
	IntroductionMessage            string           `json:"introduction_message"`
	Continuity                     ContinuityConfig `json:"continuity"`
	ShowOfficeHoursInChat          bool             `json:"show_office_hours_in_chat"`
	ShowOfficeHoursAfterAssignment bool             `json:"show_office_hours_after_assignment"`
	ChatReplyExpectationMessage    string           `json:"chat_reply_expectation_message"`
	SessionDuration                string           `json:"session_duration"`
	PreChatForm                    struct {
		Enabled bool               `json:"enabled"`
		Title   string             `json:"title"`
		Fields  []PreChatFormField `json:"fields"`
	} `json:"prechat_form"`
}

// Client represents a connected chat client
type Client struct {
	ID        string
	Channel   chan []byte
	closed    atomic.Bool
	closeOnce sync.Once
}

// CloseChannel closes the client's channel exactly once. Safe to call multiple times.
func (c *Client) CloseChannel() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.Channel)
	})
}

// LiveChat represents the live chat inbox.
type LiveChat struct {
	id            int
	config        Config
	from          string
	lo            *logf.Logger
	messageStore  inbox.MessageStore
	userStore     inbox.UserStore
	signAvatarURL func(*null.String)   // Signs a raw /uploads/ avatar path into a signed URL.
	clients       map[string][]*Client // Maps user IDs to slices of clients (to handle multiple devices)
	clientsMutex  sync.RWMutex
}

// Opts holds the options required for the live chat inbox.
type Opts struct {
	ID            int
	Config        Config
	From          string
	Lo            *logf.Logger
	SignAvatarURL func(*null.String)
}

// New returns a new instance of the live chat inbox.
func New(store inbox.MessageStore, userStore inbox.UserStore, opts Opts) (*LiveChat, error) {
	lc := &LiveChat{
		id:            opts.ID,
		config:        opts.Config,
		from:          opts.From,
		lo:            opts.Lo,
		messageStore:  store,
		userStore:     userStore,
		signAvatarURL: opts.SignAvatarURL,
		clients:       make(map[string][]*Client),
	}
	return lc, nil
}

// Identifier returns the unique identifier of the inbox which is the database ID.
func (lc *LiveChat) Identifier() int {
	return lc.id
}

// Receive is no-op as messages received via api.
func (lc *LiveChat) Receive(ctx context.Context) error {
	return nil
}

// Send sends the passed message to the message receiver if they are connected to the live chat.
func (lc *LiveChat) Send(message models.OutboundMessage) error {
	if message.MessageReceiverID <= 0 {
		lc.lo.Warn("received empty receiver_id for live chat message", "message_id", message.UUID)
		return nil
	}

	msgReceiverStr := strconv.Itoa(message.MessageReceiverID)
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()

	clients, exists := lc.clients[msgReceiverStr]
	if !exists {
		lc.lo.Debug("websocket client not connected for live chat message", "receiver_id", msgReceiverStr, "message_id", message.UUID)
		return ErrClientNotConnected
	}

	sender, err := lc.userStore.GetAgent(message.SenderID, "")
	if err != nil {
		return fmt.Errorf("failed to get sender name: %w", err)
	}

	for i := range message.Attachments {
		message.Attachments[i].Content = nil
	}

	avatarURL := sender.AvatarURL
	if lc.signAvatarURL != nil {
		lc.signAvatarURL(&avatarURL)
	}

	messageData := map[string]any{
		"type": "new_message",
		"data": models.ChatMessage{
			UUID:             message.UUID,
			ConversationUUID: message.ConversationUUID,
			CreatedAt:        message.CreatedAt,
			Content:          message.Content,
			TextContent:      message.TextContent,
			Meta:             message.Meta,
			Author: models.MessageAuthor{
				ID:                 message.SenderID,
				FirstName:          sender.FirstName,
				LastName:           sender.LastName,
				AvatarURL:          avatarURL,
				AvailabilityStatus: sender.AvailabilityStatus,
				Type:               sender.Type,
				LastActiveAt:       sender.LastActiveAt,
			},
			Attachments: message.Attachments,
		},
	}

	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		return fmt.Errorf("failed to marshal message data: %w", err)
	}

	for _, client := range clients {
		if client.closed.Load() {
			continue
		}
		select {
		case client.Channel <- messageJSON:
			lc.lo.Info("message sent to live chat client", "client_id", client.ID, "message_id", message.UUID)
		default:
			lc.lo.Warn("client channel full, dropping message", "client_id", client.ID, "message_id", message.UUID)
		}
	}

	return nil
}

// Close closes all connected client channels and clears the client map.
func (lc *LiveChat) Close() error {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()
	for _, clients := range lc.clients {
		for _, c := range clients {
			c.CloseChannel()
		}
	}
	lc.clients = make(map[string][]*Client)
	return nil
}

// FromAddress returns the from address for this inbox.
func (lc *LiveChat) FromAddress() string {
	return lc.from
}

// ReplyToAddress is not applicable to livechat and always returns empty.
func (lc *LiveChat) ReplyToAddress() string {
	return ""
}

// Channel returns the channel name for this inbox.
func (lc *LiveChat) Channel() string {
	return ChannelLiveChat
}

// AddClient adds a new client to the live chat session.
func (lc *LiveChat) AddClient(userID string) (*Client, error) {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()

	// Check if the user already has the maximum allowed connections.
	if clients, exists := lc.clients[userID]; exists && len(clients) >= MaxConnectionsPerUser {
		lc.lo.Warn("maximum connections reached for user", "client_id", userID, "max_connections", MaxConnectionsPerUser)
		return nil, fmt.Errorf("maximum connections reached")
	}

	client := &Client{
		ID:      userID,
		Channel: make(chan []byte, 1000),
	}

	// Add the client to the clients map.
	lc.clients[userID] = append(lc.clients[userID], client)
	return client, nil
}

// RemoveClient removes a client from the live chat session.
func (lc *LiveChat) RemoveClient(c *Client) {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()
	if clients, exists := lc.clients[c.ID]; exists {
		for i, client := range clients {
			if client == c {
				// Remove the client from the slice
				lc.clients[c.ID] = append(clients[:i], clients[i+1:]...)

				// If no more clients for this user, remove the entry entirely
				if len(lc.clients[c.ID]) == 0 {
					delete(lc.clients, c.ID)
				}

				lc.lo.Debug("client removed from live chat", "client_id", c.ID)
				return
			}
		}
	}
}

// BroadcastTypingToClients broadcasts typing status to specific widget clients for a conversation.
func (lc *LiveChat) BroadcastTypingToClients(conversationUUID string, contactID int, isTyping bool) {
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()

	// Create typing status message for widget clients
	typingMessage := map[string]interface{}{
		"type": "typing",
		"data": map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"is_typing":         isTyping,
		},
	}

	messageJSON, err := json.Marshal(typingMessage)
	if err != nil {
		lc.lo.Error("failed to marshal typing message", "error", err)
		return
	}

	// Only send to the specific contact's clients
	contactIDStr := strconv.Itoa(contactID)
	if clients, exists := lc.clients[contactIDStr]; exists {
		for _, client := range clients {
			if client.closed.Load() {
				continue
			}
			select {
			case client.Channel <- messageJSON:
				lc.lo.Debug("typing status sent to widget client", "contact_id", contactID, "client_id", client.ID, "conversation_uuid", conversationUUID, "is_typing", isTyping)
			default:
				lc.lo.Warn("client channel full, dropping typing message", "contact_id", contactID, "client_id", client.ID)
			}
		}
	}
}

// BroadcastMessageToClients broadcasts a new message to specific widget clients.
func (lc *LiveChat) BroadcastMessageToClients(conversationUUID string, contactID int, messageData any) {
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()

	msg := map[string]any{
		"type": "new_message",
		"data": messageData,
	}

	messageJSON, err := json.Marshal(msg)
	if err != nil {
		lc.lo.Error("failed to marshal new message for widget broadcast", "error", err)
		return
	}

	contactIDStr := strconv.Itoa(contactID)
	if clients, exists := lc.clients[contactIDStr]; exists {
		for _, client := range clients {
			if client.closed.Load() {
				continue
			}
			select {
			case client.Channel <- messageJSON:
			default:
				lc.lo.Warn("client channel full, dropping message broadcast", "contact_id", contactID, "client_id", client.ID)
			}
		}
	}
}

// BroadcastConversationToClients broadcasts conversation updates to specific widget clients.
func (lc *LiveChat) BroadcastConversationToClients(conversationUUID string, contactID int, conversationData interface{}) {
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()

	conversationMessage := map[string]any{
		"type": "conversation_update",
		"data": conversationData,
	}

	messageJSON, err := json.Marshal(conversationMessage)
	if err != nil {
		lc.lo.Error("failed to marshal conversation update message", "error", err)
		return
	}

	// Only send to the specific contact's clients
	contactIDStr := strconv.Itoa(contactID)
	if clients, exists := lc.clients[contactIDStr]; exists {
		for _, client := range clients {
			if client.closed.Load() {
				continue
			}
			select {
			case client.Channel <- messageJSON:
				lc.lo.Debug("conversation update sent to widget client", "contact_id", contactID, "client_id", client.ID, "conversation_uuid", conversationUUID)
			default:
				lc.lo.Warn("client channel full, dropping conversation update", "contact_id", contactID, "client_id", client.ID)
			}
		}
	}
}
