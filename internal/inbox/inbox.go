// Package inbox provides functionality to manage inboxes in the system.
package inbox

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const (
	ChannelEmail = "email"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	// ErrInboxNotFound is returned when an inbox is not found.
	ErrInboxNotFound = errors.New("inbox not found")
)

type initFn func(imodels.Inbox, MessageStore) (Inbox, error)

// Closer provides a function for closing an inbox.
type Closer interface {
	Close() error
}

// Identifier provides a method for obtaining a unique identifier for the inbox.
type Identifier interface {
	Identifier() int
}

// MessageHandler defines methods for handling message operations.
type MessageHandler interface {
	Receive(context.Context) error
	Send(models.Message) error
}

// Inbox combines the operations of an inbox including its lifecycle, identification, and message handling.
type Inbox interface {
	Closer
	Identifier
	MessageHandler
	FromAddress() string
	Channel() string
}

// MessageStore defines methods for storing and processing messages.
type MessageStore interface {
	MessageExists(string) (bool, error)
	EnqueueIncoming(models.IncomingMessage) error
}

// Opts contains the options for initializing the inbox manager.
type Opts struct {
	QueueSize   int
	Concurrency int
}

// Manager manages the inboxes.
type Manager struct {
	mu        sync.RWMutex
	queries   queries
	inboxes   map[int]Inbox
	lo        *logf.Logger
	i18n      *i18n.I18n
	receivers map[int]context.CancelFunc
	store     MessageStore
	wg        sync.WaitGroup
}

// Prepared queries.
type queries struct {
	GetInbox    *sqlx.Stmt `query:"get-inbox"`
	GetActive   *sqlx.Stmt `query:"get-active-inboxes"`
	GetAll      *sqlx.Stmt `query:"get-all-inboxes"`
	Update      *sqlx.Stmt `query:"update"`
	Toggle      *sqlx.Stmt `query:"toggle"`
	SoftDelete  *sqlx.Stmt `query:"soft-delete"`
	InsertInbox *sqlx.Stmt `query:"insert-inbox"`
}

// New returns a new inbox manager.
func New(lo *logf.Logger, db *sqlx.DB, i18n *i18n.I18n) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	m := &Manager{
		lo:        lo,
		inboxes:   make(map[int]Inbox),
		receivers: make(map[int]context.CancelFunc),
		queries:   q,
		i18n:      i18n,
	}
	return m, nil
}

// SetMessageStore sets the message store for the manager.
func (m *Manager) SetMessageStore(store MessageStore) {
	m.store = store
}

// Register registers the inbox with the manager.
func (m *Manager) Register(i Inbox) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inboxes[i.Identifier()] = i
}

// Get retrieves the initialized inbox instance with the specified ID from memory.
func (m *Manager) Get(id int) (Inbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	i, ok := m.inboxes[id]
	if !ok {
		return nil, ErrInboxNotFound
	}
	return i, nil
}

// GetDBRecord returns the inbox record from the DB.
func (m *Manager) GetDBRecord(id int) (imodels.Inbox, error) {
	var inbox imodels.Inbox
	if err := m.queries.GetInbox.Get(&inbox, id); err != nil {
		if err == sql.ErrNoRows {
			return inbox, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.inbox}"), nil)
		}
		m.lo.Error("error fetching inbox", "error", err)
		return inbox, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.inbox}"), nil)
	}
	return inbox, nil
}

// GetAll returns all inboxes from the DB.
func (m *Manager) GetAll() ([]imodels.Inbox, error) {
	var inboxes = make([]imodels.Inbox, 0)
	if err := m.queries.GetAll.Select(&inboxes); err != nil {
		m.lo.Error("error fetching inboxes", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", m.i18n.P("globals.entities.inbox")), nil)
	}
	return inboxes, nil
}

// Create creates an inbox in the DB.
func (m *Manager) Create(inbox imodels.Inbox) error {
	if _, err := m.queries.InsertInbox.Exec(inbox.Channel, inbox.Config, inbox.Name, inbox.From, inbox.CSATEnabled); err != nil {
		m.lo.Error("error creating inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.entities.inbox}"), nil)
	}
	return nil
}

// InitInboxes initializes and registers active inboxes with the manager.
func (m *Manager) InitInboxes(initFn initFn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inboxRecords, err := m.getActive()
	if err != nil {
		m.lo.Error("error fetching active inboxes", "error", err)
		return fmt.Errorf("fetching active inboxes: %v", err)
	}

	for _, inboxRecord := range inboxRecords {
		inbox, err := initFn(inboxRecord, m.store)
		if err != nil {
			m.lo.Error("error initializing inbox",
				"name", inboxRecord.Name,
				"channel", inboxRecord.Channel,
				"error", err)
			continue
		}
		m.inboxes[inbox.Identifier()] = inbox
	}
	return nil
}

// Reload hot reloads the inboxes with the given init function.
func (m *Manager) Reload(ctx context.Context, initFn initFn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel all existing receivers.
	for _, cancel := range m.receivers {
		cancel()
	}
	m.receivers = make(map[int]context.CancelFunc)

	// Close existing inboxes.
	for _, inb := range m.inboxes {
		inb.Close()
	}

	// Clear and reload inboxes.
	m.inboxes = make(map[int]Inbox)
	inboxRecords, err := m.getActive()
	if err != nil {
		return fmt.Errorf("error fetching active inboxes: %v", err)
	}

	// Initialize new inboxes.
	for _, inboxRecord := range inboxRecords {
		inbox, err := initFn(inboxRecord, m.store)
		if err != nil {
			m.lo.Error("error initializing inbox during reload",
				"name", inboxRecord.Name,
				"channel", inboxRecord.Channel,
				"error", err)
			continue
		}
		m.inboxes[inbox.Identifier()] = inbox
	}

	// Start new receivers.
	for _, inb := range m.inboxes {
		receiverCtx, cancel := context.WithCancel(ctx)
		m.receivers[inb.Identifier()] = cancel

		go func(inbox Inbox) {
			if err := inbox.Receive(receiverCtx); err != nil {
				m.lo.Error("error starting inbox receiver", "error", err)
			}
		}(inb)
	}

	return nil
}

// Update updates an inbox in the DB.
func (m *Manager) Update(id int, inbox imodels.Inbox) error {
	current, err := m.GetDBRecord(id)
	if err != nil {
		return err
	}

	// Preserve existing passwords if update has empty password
	switch current.Channel {
	case "email":
		var currentCfg struct {
			IMAP []map[string]interface{} `json:"imap"`
			SMTP []map[string]interface{} `json:"smtp"`
		}
		var updateCfg struct {
			IMAP []map[string]interface{} `json:"imap"`
			SMTP []map[string]interface{} `json:"smtp"`
		}

		if err := json.Unmarshal(current.Config, &currentCfg); err != nil {
			m.lo.Error("error unmarshalling current config", "id", id, "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.config}"), nil)
		}
		if len(inbox.Config) == 0 {
			return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "{globals.entities.config}"), nil)
		}
		if err := json.Unmarshal(inbox.Config, &updateCfg); err != nil {
			m.lo.Error("error unmarshalling update config", "id", id, "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.config}"), nil)
		}

		if len(updateCfg.IMAP) == 0 {
			return envelope.NewError(envelope.InputError, m.i18n.T("inbox.emptyIMAP"), nil)
		}

		if len(updateCfg.SMTP) == 0 {
			return envelope.NewError(envelope.InputError, m.i18n.T("inbox.emptySMTP"), nil)
		}

		// Preserve existing IMAP passwords if update has empty password
		for i := range updateCfg.IMAP {
			if updateCfg.IMAP[i]["password"] == "" && i < len(currentCfg.IMAP) {
				updateCfg.IMAP[i]["password"] = currentCfg.IMAP[i]["password"]
			}
		}

		// Preserve existing SMTP passwords if update has empty password
		for i := range updateCfg.SMTP {
			if updateCfg.SMTP[i]["password"] == "" && i < len(currentCfg.SMTP) {
				updateCfg.SMTP[i]["password"] = currentCfg.SMTP[i]["password"]
			}
		}
		updatedConfig, err := json.Marshal(updateCfg)
		if err != nil {
			m.lo.Error("error marshalling updated config", "id", id, "error", err)
			return err
		}
		inbox.Config = updatedConfig
	}

	// Update the inbox in the DB.
	if _, err := m.queries.Update.Exec(id, inbox.Channel, inbox.Config, inbox.Name, inbox.From, inbox.CSATEnabled, inbox.Enabled); err != nil {
		m.lo.Error("error updating inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.inbox}"), nil)
	}

	return nil
}

// Toggle toggles the status of an inbox in the DB.
func (m *Manager) Toggle(id int) error {
	if _, err := m.queries.Toggle.Exec(id); err != nil {
		m.lo.Error("error toggling inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.entities.inbox}"), nil)
	}
	return nil
}

// SoftDelete soft deletes an inbox in the DB.
func (m *Manager) SoftDelete(id int) error {
	if _, err := m.queries.SoftDelete.Exec(id); err != nil {
		m.lo.Error("error deleting inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.entities.inbox}"), nil)
	}
	return nil
}

// Start starts the receiver for each inbox.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inb := range m.inboxes {
		receiverCtx, cancel := context.WithCancel(ctx)
		m.receivers[inb.Identifier()] = cancel

		m.wg.Add(1)
		go func(inbox Inbox) {
			defer m.wg.Done()
			if err := inbox.Receive(receiverCtx); err != nil {
				m.lo.Error("error starting inbox receiver", "error", err)
			}
		}(inb)
	}
	return nil
}

// Close closes all inboxes.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel all receivers.
	for _, cancel := range m.receivers {
		cancel()
	}

	// Close all inboxes.
	for _, inb := range m.inboxes {
		inb.Close()
	}

	// Wait for all workers to finish.
	m.wg.Wait()
}

// getActive returns all active inboxes from the DB.
func (m *Manager) getActive() ([]imodels.Inbox, error) {
	var inboxes []imodels.Inbox
	if err := m.queries.GetActive.Select(&inboxes); err != nil {
		m.lo.Error("fetching active inboxes", "error", err)
		return nil, err
	}
	return inboxes, nil
}
