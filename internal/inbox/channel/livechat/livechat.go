// Package livechat implements a live chat inbox for handling real-time conversations.
package livechat

import (
	"context"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/zerodha/logf"
)

const (
	ChannelLiveChat = "livechat"
)

// Config holds the live chat inbox configuration.
type Config struct {
	Users struct {
		AllowStartConversation       bool   `json:"allow_start_conversation"`
		PreventMultipleConversations bool   `json:"prevent_multiple_conversations"`
		StartConversationButtonText  string `json:"start_conversation_button_text"`
	} `json:"users"`
	Colors struct {
		Primary    string `json:"primary"`
		Background string `json:"background"`
	} `json:"colors"`
	Features struct {
		Emoji                  bool `json:"emoji"`
		FileUpload             bool `json:"file_upload"`
		AllowCloseConversation bool `json:"allow_close_conversation"`
	} `json:"features"`
	Launcher struct {
		Spacing struct {
			Side   int `json:"side"`
			Bottom int `json:"bottom"`
		} `json:"spacing"`
		LogoURL  string `json:"logo_url"`
		Position string `json:"position"`
	} `json:"launcher"`
	LogoURL  string `json:"logo_url"`
	Visitors struct {
		AllowStartConversation       bool   `json:"allow_start_conversation"`
		PreventMultipleConversations bool   `json:"prevent_multiple_conversations"`
		StartConversationButtonText  string `json:"start_conversation_button_text"`
	} `json:"visitors"`
	SecretKey    string `json:"secret_key"`
	NoticeBanner struct {
		Text    string `json:"text"`
		Enabled bool   `json:"enabled"`
	} `json:"notice_banner"`
	ExternalLinks []struct {
		URL  string `json:"url"`
		Text string `json:"text"`
	} `json:"external_links"`
	TrustedDomains                 []string `json:"trusted_domains"`
	GreetingMessage                string   `json:"greeting_message"`
	ChatIntroduction               string   `json:"chat_introduction"`
	IntroductionMessage            string   `json:"introduction_message"`
	ShowOfficeHoursInChat          bool     `json:"show_office_hours_in_chat"`
	ShowOfficeHoursAfterAssignment bool     `json:"show_office_hours_after_assignment"`
}

// Client represents a connected chat client
type Client struct {
	ID             string
	ConversationID string
	Channel        chan []byte
	LastActivity   time.Time
	mutex          sync.RWMutex
}

// LiveChat represents the live chat inbox.
type LiveChat struct {
	id           int
	config       Config
	from         string
	lo           *logf.Logger
	messageStore inbox.MessageStore
	userStore    inbox.UserStore
	clients      map[string]*Client
	// conversationClients maps conversation IDs to client IDs.
	conversationClients map[string]map[string]*Client
	clientsMutex        sync.RWMutex
	wg                  sync.WaitGroup
}

// Opts holds the options required for the live chat inbox.
type Opts struct {
	ID     int
	Config Config
	From   string
	Lo     *logf.Logger
}

// New returns a new instance of the live chat inbox.
func New(store inbox.MessageStore, userStore inbox.UserStore, opts Opts) (*LiveChat, error) {
	lc := &LiveChat{
		id:                  opts.ID,
		config:              opts.Config,
		from:                opts.From,
		lo:                  opts.Lo,
		messageStore:        store,
		userStore:           userStore,
		clients:             make(map[string]*Client),
		conversationClients: make(map[string]map[string]*Client),
	}
	return lc, nil
}

// Identifier returns the unique identifier of the inbox which is the database ID.
func (lc *LiveChat) Identifier() int {
	return lc.id
}

// Receive handles incoming messages for the live chat channel.
// For live chat, this is a no-op as messages come through WebSocket connections.
func (lc *LiveChat) Receive(ctx context.Context) error {
	lc.lo.Info("live chat receiver started", "inbox_id", lc.id)

	// Start a cleanup routine for inactive clients
	lc.wg.Add(1)
	go func() {
		defer lc.wg.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lc.cleanupInactiveClients()
			}
		}
	}()

	<-ctx.Done()
	lc.wg.Wait()
	return nil
}

// Send sends a message through the live chat channel.
func (lc *LiveChat) Send(message models.Message) error {
	lc.lo.Info("sending live chat message",
		"conversation_id", message.ConversationUUID,
		"message_id", message.UUID)
	return nil
}

// Close closes the live chat channel.
func (lc *LiveChat) Close() error {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()

	// Close all client channels
	for _, client := range lc.clients {
		close(client.Channel)
	}

	lc.clients = make(map[string]*Client)
	lc.conversationClients = make(map[string]map[string]*Client)
	return nil
}

// FromAddress returns the from address for this inbox.
func (lc *LiveChat) FromAddress() string {
	return lc.from
}

// Channel returns the channel name for this inbox.
func (lc *LiveChat) Channel() string {
	return ChannelLiveChat
}

// AddClient adds a new client to the live chat session.
// TODO: Limit the number of clients that can be connected for a `clientID`.
func (lc *LiveChat) AddClient(clientID, conversationID string) *Client {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()

	client := &Client{
		ID:             clientID,
		ConversationID: conversationID,
		Channel:        make(chan []byte, 256),
		LastActivity:   time.Now(),
	}

	lc.clients[clientID] = client
	
	// Add to conversation mapping for faster lookup
	if lc.conversationClients[conversationID] == nil {
		lc.conversationClients[conversationID] = make(map[string]*Client)
	}
	lc.conversationClients[conversationID][clientID] = client
	
	lc.lo.Info("client added to live chat", "client_id", clientID, "conversation_id", conversationID)
	return client
}

// RemoveClient removes a client from the live chat session.
func (lc *LiveChat) RemoveClient(clientID string) {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()

	if client, exists := lc.clients[clientID]; exists {
		close(client.Channel)
		delete(lc.clients, clientID)
		
		// Remove from conversation mapping
		if conversationClients, exists := lc.conversationClients[client.ConversationID]; exists {
			delete(conversationClients, clientID)
			// Clean up empty conversation mapping
			if len(conversationClients) == 0 {
				delete(lc.conversationClients, client.ConversationID)
			}
		}
		
		lc.lo.Info("client removed from live chat", "client_id", clientID)
	}
}

// GetClient returns a client by ID.
func (lc *LiveChat) GetClient(clientID string) (*Client, bool) {
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()

	client, exists := lc.clients[clientID]
	return client, exists
}

// BroadcastToConversation broadcasts a message to all clients in a conversation.
func (lc *LiveChat) BroadcastToConversation(conversationID string, message []byte) {
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()

	// Use the conversation mapping for O(1) lookup instead of iterating all clients
	if conversationClients, exists := lc.conversationClients[conversationID]; exists {
		for _, client := range conversationClients {
			select {
			case client.Channel <- message:
			default:
				lc.lo.Warn("client channel full, dropping message", "client_id", client.ID)
			}
		}
	}
}

// GetConfig returns the live chat configuration.
func (lc *LiveChat) GetConfig() Config {
	return lc.config
}

// UpdateClientActivity updates the last activity time for a client.
func (lc *LiveChat) UpdateClientActivity(clientID string) {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()

	if client, exists := lc.clients[clientID]; exists {
		client.mutex.Lock()
		client.LastActivity = time.Now()
		client.mutex.Unlock()
	}
}

// cleanupInactiveClients removes clients that have been inactive for too long.
func (lc *LiveChat) cleanupInactiveClients() {
	lc.clientsMutex.Lock()
	defer lc.clientsMutex.Unlock()

	cutoff := time.Now().Add(-30 * time.Minute)

	for clientID, client := range lc.clients {
		client.mutex.RLock()
		lastActivity := client.LastActivity
		client.mutex.RUnlock()

		if lastActivity.Before(cutoff) {
			close(client.Channel)
			delete(lc.clients, clientID)
			
			// Remove from conversation mapping
			if conversationClients, exists := lc.conversationClients[client.ConversationID]; exists {
				delete(conversationClients, clientID)
				// Clean up empty conversation mapping
				if len(conversationClients) == 0 {
					delete(lc.conversationClients, client.ConversationID)
				}
			}
			
			lc.lo.Info("cleaned up inactive client", "client_id", clientID)
		}
	}
}

// GetActiveClients returns the number of active clients.
func (lc *LiveChat) GetActiveClients() int {
	lc.clientsMutex.RLock()
	defer lc.clientsMutex.RUnlock()
	return len(lc.clients)
}
