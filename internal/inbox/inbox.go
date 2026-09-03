// Package inbox provides functionality to manage inboxes in the system.
package inbox

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	ChannelEmail    = "email"
	ChannelLiveChat = "livechat"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	// ErrInboxNotFound is returned when an inbox is not found.
	ErrInboxNotFound = errors.New("inbox not found")
)

type initFn func(imodels.Inbox, MessageStore, UserStore) (Inbox, error)

type aliasVerificationState struct {
	Email     string         `db:"email"`
	Token     sql.NullString `db:"verification_token"`
	StartedAt sql.NullTime   `db:"verification_started_at"`
}

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
	Send(models.OutboundMessage) error
}

// Inbox combines the operations of an inbox including its lifecycle, identification, and message handling.
type Inbox interface {
	Closer
	Identifier
	MessageHandler
	Name() string
	FromAddress() string
	FromNameTemplate() string
	ReplyToAddress() string
	Channel() string
}

// EmailInbox exposes address ownership needed by email delivery.
type EmailInbox interface {
	Inbox
	UUID() string
	PrimaryAddress() string
	OwnedAddresses() []string
	OwnsAddress(string) bool
	SendsAddress(string) bool
}

// AliasVerificationStarter sends verification through an inbox's existing SMTP pool.
type AliasVerificationStarter interface {
	StartAliasVerification(string, string) error
}

type aliasSendStateUpdater interface {
	SetAliasSendable(string, bool)
}

// MessageStore defines methods for storing and processing messages.
type MessageStore interface {
	MessageExists(string) (bool, error)
	EnqueueIncoming(models.IncomingMessage) error
}

// UserStore defines methods for fetching user information.
type UserStore interface {
	GetAgent(id int, email string) (umodels.User, error)
	IsEmailBlocked(email string) (bool, error)
}

// Opts contains the options for initializing the inbox manager.
type Opts struct {
	QueueSize   int
	Concurrency int
}

// receiverState tracks a *single*s inbox receiver goroutine.
type receiverState struct {
	cancel context.CancelFunc
	done   chan struct{} // closed when the goroutine exits
}

type Manager struct {
	mu            sync.RWMutex
	queries       queries
	inboxes       map[int]Inbox
	lo            *logf.Logger
	i18n          *i18n.I18n
	receivers     map[int]receiverState
	msgStore      MessageStore
	usrStore      UserStore
	wg            sync.WaitGroup
	encryptionKey string
	db            *sqlx.DB
}

// Prepared queries.
type queries struct {
	GetInbox        *sqlx.Stmt `query:"get-inbox"`
	GetInboxByUUID  *sqlx.Stmt `query:"get-inbox-by-uuid"`
	GetActive       *sqlx.Stmt `query:"get-active-inboxes"`
	GetAll          *sqlx.Stmt `query:"get-all-inboxes"`
	Update          *sqlx.Stmt `query:"update"`
	Toggle          *sqlx.Stmt `query:"toggle"`
	SoftDelete      *sqlx.Stmt `query:"soft-delete"`
	InsertInbox     *sqlx.Stmt `query:"insert-inbox"`
	UpdateConfig    *sqlx.Stmt `query:"update-config"`
	DeleteAddresses *sqlx.Stmt `query:"delete-inbox-email-addresses"`
	InsertAddress   *sqlx.Stmt `query:"insert-inbox-email-address"`
	GetAddressOwner *sqlx.Stmt `query:"get-email-address-owner"`
}

// New returns a new inbox manager.
func New(lo *logf.Logger, db *sqlx.DB, i18n *i18n.I18n, encryptionKey string) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	m := &Manager{
		lo:            lo,
		inboxes:       make(map[int]Inbox),
		receivers:     make(map[int]receiverState),
		queries:       q,
		i18n:          i18n,
		encryptionKey: encryptionKey,
		db:            db,
	}
	return m, nil
}

// SetMessageStore sets the message store for the manager.
func (m *Manager) SetMessageStore(store MessageStore) {
	m.msgStore = store
}

// SetUserStore sets the user store for the manager.
func (m *Manager) SetUserStore(store UserStore) {
	m.usrStore = store
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

// GetDBRecord returns the inbox record from the DB by numeric ID or UUID.
// If the identifier contains a dash, it's treated as a UUID; otherwise as a numeric ID.
func (m *Manager) GetDBRecord(identifier any) (imodels.Inbox, error) {
	var inbox imodels.Inbox

	// If it's a string with dashes, look up by UUID; otherwise by numeric ID.
	str := fmt.Sprintf("%v", identifier)
	if strings.Contains(str, "-") {
		if err := m.queries.GetInboxByUUID.Get(&inbox, str); err != nil {
			if err == sql.ErrNoRows {
				return inbox, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundInbox"), nil)
			}
			m.lo.Error("error fetching inbox", "error", err)
			return inbox, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	} else {
		id, err := strconv.Atoi(str)
		if err != nil {
			return inbox, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundInbox"), nil)
		}
		if err := m.queries.GetInbox.Get(&inbox, id); err != nil {
			if err == sql.ErrNoRows {
				return inbox, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundInbox"), nil)
			}
			m.lo.Error("error fetching inbox", "error", err)
			return inbox, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}

	decryptedConfig, err := m.decryptInboxConfig(inbox.Config)
	if err != nil {
		m.lo.Error("error decrypting inbox config", "identifier", identifier, "error", err)
		return imodels.Inbox{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	inbox.Config = decryptedConfig

	m.decryptInboxSecret(&inbox)

	return inbox, nil
}

// GetAll returns all inboxes from the DB.
func (m *Manager) GetAll() ([]imodels.Inbox, error) {
	var inboxes = make([]imodels.Inbox, 0)
	if err := m.queries.GetAll.Select(&inboxes); err != nil {
		m.lo.Error("error fetching inboxes", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Decrypt sensitive fields in each inbox config
	for i := range inboxes {
		decryptedConfig, err := m.decryptInboxConfig(inboxes[i].Config)
		if err != nil {
			m.lo.Error("error decrypting inbox config", "id", inboxes[i].ID, "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		inboxes[i].Config = decryptedConfig

		// Decrypt secret field
		m.decryptInboxSecret(&inboxes[i])
	}

	return inboxes, nil
}

// Create creates an inbox in the DB.
func (m *Manager) Create(inbox imodels.Inbox) (imodels.Inbox, error) {
	if inbox.Channel == ChannelEmail {
		_, aliases, err := ValidateEmailAddresses(inbox.From, inbox.Aliases)
		if err != nil {
			return imodels.Inbox{}, envelope.NewError(envelope.InputError, err.Error(), nil)
		}
		inbox.Aliases = aliases
		for i := range inbox.Aliases {
			inbox.Aliases[i].VerificationStatus = imodels.AliasVerificationNotVerified
			inbox.Aliases[i].VerifiedAt = nil
		}
	}
	if inbox.Channel == ChannelLiveChat {
		secret := inbox.Secret.String
		if secret == "" {
			generated, err := stringutil.RandomAlphanumeric(32)
			if err != nil {
				return imodels.Inbox{}, fmt.Errorf("generating inbox secret: %w", err)
			}
			secret = generated
		}
		encryptedSecret, err := crypto.Encrypt(secret, m.encryptionKey)
		if err != nil {
			return imodels.Inbox{}, fmt.Errorf("encrypting inbox secret: %w", err)
		}
		inbox.Secret = null.StringFrom(encryptedSecret)
	}

	// Encrypt sensitive fields before saving
	encryptedConfig, err := m.encryptInboxConfig(inbox.Config)
	if err != nil {
		m.lo.Error("error encrypting inbox config", "error", err)
		return imodels.Inbox{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	tx, err := m.db.Beginx()
	if err != nil {
		return imodels.Inbox{}, m.persistenceError("starting inbox creation", err)
	}
	defer func() { _ = tx.Rollback() }()

	var createdInbox imodels.Inbox
	if err := tx.Stmtx(m.queries.InsertInbox).Get(&createdInbox, inbox.Channel, encryptedConfig, inbox.Name, inbox.From, inbox.Enabled, inbox.CSATEnabled, inbox.PromptTagsOnReply, inbox.Secret, inbox.LinkedEmailInboxID, inbox.FromNameTemplate); err != nil {
		m.lo.Error("error creating inbox", "error", err)
		return imodels.Inbox{}, m.persistenceError("creating inbox", err)
	}
	if inbox.Channel == ChannelEmail {
		if err := m.replaceEmailAddresses(tx, createdInbox.ID, inbox.From, inbox.Aliases); err != nil {
			return imodels.Inbox{}, err
		}
		createdInbox.Aliases = inbox.Aliases
	}
	if err := tx.Commit(); err != nil {
		return imodels.Inbox{}, m.persistenceError("committing inbox creation", err)
	}

	// Decrypt before returning
	decryptedConfig, err := m.decryptInboxConfig(createdInbox.Config)
	if err != nil {
		m.lo.Error("error decrypting inbox config after creation", "error", err)
	} else {
		createdInbox.Config = decryptedConfig
	}

	// Decrypt secret field
	m.decryptInboxSecret(&createdInbox)

	return createdInbox, nil
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
		inbox, err := initFn(inboxRecord, m.msgStore, m.usrStore)
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

// ReloadInbox reloads a single inbox by ID. It stops the old receiver,
// fetches the current state from DB, and re-initializes if active.
func (m *Manager) ReloadInbox(ctx context.Context, id int, initFn initFn) error {
	// Stop old receiver and close old inbox.
	m.stopInbox(id)

	// Fetch current inbox state from DB.
	record, err := m.GetDBRecord(id)
	if err != nil {
		// Not found (e.g. deleted) - already removed above.
		return nil
	}

	// Only re-init if enabled.
	if !record.Enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	inbox, err := initFn(record, m.msgStore, m.usrStore)
	if err != nil {
		return fmt.Errorf("initializing inbox %s: %w", record.Name, err)
	}
	m.inboxes[inbox.Identifier()] = inbox
	m.startReceiver(ctx, inbox)
	return nil
}

// Update updates an inbox in the DB.
func (m *Manager) Update(id int, inbox imodels.Inbox) (imodels.Inbox, error) {
	current, err := m.GetDBRecord(id)
	if err != nil {
		return imodels.Inbox{}, err
	}
	if inbox.Channel != current.Channel {
		return imodels.Inbox{}, envelope.NewError(envelope.InputError, "Inbox channel cannot be changed.", nil)
	}
	if current.Channel == ChannelEmail {
		_, aliases, err := ValidateEmailAddresses(inbox.From, inbox.Aliases)
		if err != nil {
			return imodels.Inbox{}, envelope.NewError(envelope.InputError, err.Error(), nil)
		}
		inbox.Aliases = aliases
		for i := range inbox.Aliases {
			inbox.Aliases[i].VerificationStatus = imodels.AliasVerificationNotVerified
			inbox.Aliases[i].VerifiedAt = nil
			for _, previous := range current.Aliases {
				if strings.EqualFold(previous.Email, inbox.Aliases[i].Email) {
					inbox.Aliases[i].VerificationStatus = previous.VerificationStatus
					inbox.Aliases[i].VerifiedAt = previous.VerifiedAt
					break
				}
			}
		}
	}

	// Preserve existing passwords if update has empty password
	switch current.Channel {
	case "email":
		var currentCfg struct {
			AuthType             string            `json:"auth_type"`
			OAuth                map[string]string `json:"oauth"`
			IMAP                 []map[string]any  `json:"imap"`
			SMTP                 []map[string]any  `json:"smtp"`
			ReplyTo              string            `json:"reply_to"`
			EnablePlusAddressing bool              `json:"enable_plus_addressing"`
		}
		var updateCfg struct {
			AuthType             string            `json:"auth_type"`
			OAuth                map[string]string `json:"oauth"`
			IMAP                 []map[string]any  `json:"imap"`
			SMTP                 []map[string]any  `json:"smtp"`
			ReplyTo              string            `json:"reply_to"`
			EnablePlusAddressing bool              `json:"enable_plus_addressing"`
		}

		if err := json.Unmarshal(current.Config, &currentCfg); err != nil {
			m.lo.Error("error unmarshalling current config", "id", id, "error", err)
			return imodels.Inbox{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if len(inbox.Config) == 0 {
			return imodels.Inbox{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "{globals.terms.config}"), nil)
		}
		if err := json.Unmarshal(inbox.Config, &updateCfg); err != nil {
			m.lo.Error("error unmarshalling update config", "id", id, "error", err)
			return imodels.Inbox{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}

		if len(updateCfg.IMAP) == 0 {
			return imodels.Inbox{}, envelope.NewError(envelope.InputError, m.i18n.T("inbox.emptyIMAP"), nil)
		}

		if len(updateCfg.SMTP) == 0 {
			return imodels.Inbox{}, envelope.NewError(envelope.InputError, m.i18n.T("inbox.emptySMTP"), nil)
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

		// Preserve existing OAuth fields if update has empty
		if currentCfg.OAuth != nil {
			if updateCfg.OAuth == nil {
				updateCfg.OAuth = make(map[string]string)
			}
			for k, v := range currentCfg.OAuth {
				if updateCfg.OAuth[k] == "" {
					updateCfg.OAuth[k] = v
				}
			}
		}

		updatedConfig, err := json.Marshal(updateCfg)
		if err != nil {
			m.lo.Error("error marshalling updated config", "id", id, "error", err)
			return imodels.Inbox{}, err
		}
		inbox.Config = updatedConfig
	case "livechat":
		// Preserve existing secret if update contains password dummy
		if inbox.Secret.Valid && strings.Contains(inbox.Secret.String, stringutil.PasswordDummy) {
			inbox.Secret = current.Secret
		} else if inbox.Secret.Valid && inbox.Secret.String != "" {
			// Encrypt new secret
			encryptedSecret, err := crypto.Encrypt(inbox.Secret.String, m.encryptionKey)
			if err != nil {
				return imodels.Inbox{}, fmt.Errorf("encrypting inbox secret: %w", err)
			}
			inbox.Secret = null.StringFrom(encryptedSecret)
		}
	}

	// Encrypt sensitive fields before updating
	encryptedConfig, err := m.encryptInboxConfig(inbox.Config)
	if err != nil {
		m.lo.Error("error encrypting inbox config", "error", err)
		return imodels.Inbox{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	tx, err := m.db.Beginx()
	if err != nil {
		return imodels.Inbox{}, m.persistenceError("starting inbox update", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update the inbox and address ownership atomically.
	var updatedInbox imodels.Inbox
	verificationStates := make(map[string]aliasVerificationState)
	if current.Channel == ChannelEmail {
		var states []aliasVerificationState
		if err := tx.Select(&states, `SELECT email, verification_token, verification_started_at FROM inbox_email_addresses WHERE inbox_id = $1 AND kind = 'alias'`, id); err != nil {
			return imodels.Inbox{}, m.persistenceError("fetching inbox alias verification state", err)
		}
		for _, state := range states {
			verificationStates[strings.ToLower(state.Email)] = state
		}
	}
	if err := tx.Stmtx(m.queries.Update).Get(&updatedInbox, id, inbox.Channel, encryptedConfig, inbox.Name, inbox.From, inbox.CSATEnabled, inbox.PromptTagsOnReply, inbox.Enabled, inbox.Secret, inbox.LinkedEmailInboxID, inbox.FromNameTemplate); err != nil {
		m.lo.Error("error updating inbox", "error", err)
		return imodels.Inbox{}, m.persistenceError("updating inbox", err)
	}
	if _, err := tx.Stmtx(m.queries.DeleteAddresses).Exec(id); err != nil {
		return imodels.Inbox{}, m.persistenceError("clearing inbox addresses", err)
	}
	if current.Channel == ChannelEmail {
		if err := m.insertEmailAddresses(tx, id, inbox.From, inbox.Aliases, verificationStates); err != nil {
			return imodels.Inbox{}, err
		}
		updatedInbox.Aliases = inbox.Aliases
	}
	if err := tx.Commit(); err != nil {
		return imodels.Inbox{}, m.persistenceError("committing inbox update", err)
	}

	// Decrypt before returning
	decryptedConfig, err := m.decryptInboxConfig(updatedInbox.Config)
	if err != nil {
		m.lo.Error("error decrypting inbox config after update", "error", err)
	} else {
		updatedInbox.Config = decryptedConfig
	}

	// Decrypt secret field
	m.decryptInboxSecret(&updatedInbox)

	return updatedInbox, nil
}

// Toggle toggles the status of an inbox in the DB.
func (m *Manager) Toggle(ctx context.Context, id int) (imodels.Inbox, error) {
	if _, err := m.queries.Toggle.ExecContext(ctx, id); err != nil {
		m.lo.Error("error toggling inbox", "error", err)
		return imodels.Inbox{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return m.GetDBRecord(id)
}

// SoftDelete soft deletes an inbox in the DB.
func (m *Manager) SoftDelete(ctx context.Context, id int) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return m.persistenceError("starting inbox deletion", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Stmtx(m.queries.SoftDelete).ExecContext(ctx, id); err != nil {
		m.lo.Error("error deleting inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.queries.DeleteAddresses).ExecContext(ctx, id); err != nil {
		return m.persistenceError("releasing inbox addresses", err)
	}
	return tx.Commit()
}

// GetEmailAddressOwner returns the owning inbox ID, or zero when unowned.
func (m *Manager) GetEmailAddressOwner(address string) (int, error) {
	normalized, err := NormalizeEmailAddress(address)
	if err != nil {
		return 0, err
	}
	var inboxID int
	if err := m.queries.GetAddressOwner.Get(&inboxID, normalized); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return inboxID, nil
}

func (m *Manager) replaceEmailAddresses(tx *sqlx.Tx, inboxID int, from string, aliases imodels.EmailAliases) error {
	if _, err := tx.Stmtx(m.queries.DeleteAddresses).Exec(inboxID); err != nil {
		return m.persistenceError("clearing inbox addresses", err)
	}
	return m.insertEmailAddresses(tx, inboxID, from, aliases, nil)
}

func (m *Manager) insertEmailAddresses(tx *sqlx.Tx, inboxID int, from string, aliases imodels.EmailAliases, verificationStates map[string]aliasVerificationState) error {
	primary, normalizedAliases, err := ValidateEmailAddresses(from, aliases)
	if err != nil {
		return envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	if _, err := tx.Stmtx(m.queries.InsertAddress).Exec(inboxID, primary, "primary", 0, imodels.AliasVerificationVerified, nil, nil, nil); err != nil {
		return m.persistenceError("claiming inbox address", err)
	}
	for position, alias := range normalizedAliases {
		status := alias.VerificationStatus
		if status == "" {
			status = imodels.AliasVerificationNotVerified
		}
		var token, startedAt any
		if state, ok := verificationStates[alias.Email]; ok {
			if state.Token.Valid {
				token = state.Token.String
			}
			if state.StartedAt.Valid {
				startedAt = state.StartedAt.Time
			}
		}
		if _, err := tx.Stmtx(m.queries.InsertAddress).Exec(inboxID, alias.Email, "alias", position+1, status, token, startedAt, alias.VerifiedAt); err != nil {
			return m.persistenceError("claiming inbox address", err)
		}
	}
	return nil
}

// StartAliasVerification marks an alias pending and sends the verification
// message through the already initialized inbox SMTP pool.
func (m *Manager) StartAliasVerification(ctx context.Context, id int, address string) error {
	normalized, err := NormalizeEmailAddress(address)
	if err != nil {
		return envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	inbox, err := m.GetDBRecord(id)
	if err != nil {
		return err
	}
	var found *imodels.EmailAlias
	for i := range inbox.Aliases {
		if strings.EqualFold(inbox.Aliases[i].Email, normalized) {
			found = &inbox.Aliases[i]
			break
		}
	}
	if found == nil {
		return envelope.NewError(envelope.InputError, "Email alias not found.", nil)
	}
	token, err := stringutil.RandomAlphanumeric(48)
	if err != nil {
		return fmt.Errorf("generating alias verification token: %w", err)
	}
	if _, err := m.db.ExecContext(ctx, `
		UPDATE inbox_email_addresses
		SET verification_status = $3, verification_token = $4,
			verification_started_at = NOW(), verified_at = NULL
		WHERE inbox_id = $1 AND LOWER(email) = LOWER($2) AND kind = 'alias'`,
		id, normalized, imodels.AliasVerificationPending, token); err != nil {
		return m.persistenceError("starting alias verification", err)
	}
	runtimeInbox, err := m.Get(id)
	if err != nil {
		_, _ = m.db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = $3, verification_token = NULL WHERE inbox_id = $1 AND LOWER(email) = LOWER($2) AND kind = 'alias'`, id, normalized, imodels.AliasVerificationFailed)
		return err
	}
	verifier, ok := runtimeInbox.(AliasVerificationStarter)
	if !ok {
		_, _ = m.db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = $3, verification_token = NULL WHERE inbox_id = $1 AND LOWER(email) = LOWER($2) AND kind = 'alias'`, id, normalized, imodels.AliasVerificationFailed)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if updater, ok := runtimeInbox.(aliasSendStateUpdater); ok {
		updater.SetAliasSendable(normalized, false)
	}
	if err := verifier.StartAliasVerification(normalized, token); err != nil {
		_, _ = m.db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = $3, verification_token = NULL WHERE inbox_id = $1 AND LOWER(email) = LOWER($2) AND kind = 'alias'`, id, normalized, imodels.AliasVerificationFailed)
		return err
	}
	return nil
}

// CompleteAliasVerification consumes a verification message received by IMAP.
func (m *Manager) CompleteAliasVerification(ctx context.Context, id int, token, from string) error {
	normalized, err := NormalizeEmailAddress(from)
	if err != nil {
		_, _ = m.db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = $2, verification_token = NULL WHERE inbox_id = $1 AND verification_token = $3 AND kind = 'alias' AND verification_status = $4`, id, imodels.AliasVerificationFailed, token, imodels.AliasVerificationPending)
		return err
	}
	result, err := m.db.ExecContext(ctx, `
		UPDATE inbox_email_addresses
		SET verification_status = $4, verification_token = NULL, verified_at = NOW()
		WHERE inbox_id = $1 AND verification_token = $2 AND LOWER(email) = $3
		  AND kind = 'alias' AND verification_status = $5`,
		id, token, normalized, imodels.AliasVerificationVerified, imodels.AliasVerificationPending)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		_, _ = m.db.ExecContext(ctx, `UPDATE inbox_email_addresses SET verification_status = $3, verification_token = NULL WHERE inbox_id = $1 AND verification_token = $2 AND kind = 'alias' AND verification_status = $4`, id, token, imodels.AliasVerificationFailed, imodels.AliasVerificationPending)
		return nil
	}
	if runtimeInbox, err := m.Get(id); err == nil {
		if updater, ok := runtimeInbox.(aliasSendStateUpdater); ok {
			updater.SetAliasSendable(normalized, true)
		}
	}
	return nil
}

func (m *Manager) persistenceError(action string, err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.inboxEmailAddressInUse"), nil)
	}
	m.lo.Error("inbox persistence error", "action", action, "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}

// UpdateConfig updates only the config field of an inbox in the DB.
func (m *Manager) UpdateConfig(id int, config json.RawMessage) error {
	// Encrypt fields before updating
	encryptedConfig, err := m.encryptInboxConfig(config)
	if err != nil {
		m.lo.Error("error encrypting inbox config", "id", id, "error", err)
		return fmt.Errorf("encrypting inbox config: %w", err)
	}

	if _, err := m.queries.UpdateConfig.Exec(id, encryptedConfig); err != nil {
		m.lo.Error("error updating inbox config", "id", id, "error", err)
		return fmt.Errorf("updating inbox config: %w", err)
	}
	return nil
}

// CloseLiveChatClients disconnects widget websocket clients and returns the number of inboxes closed.
func (m *Manager) CloseLiveChatClients() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var n int
	for _, inb := range m.inboxes {
		if inb.Channel() != ChannelLiveChat {
			continue
		}
		if err := inb.Close(); err != nil {
			m.lo.Error("error closing livechat inbox", "error", err)
			continue
		}
		n++
	}
	return n
}

// stopInbox cancels the receiver for a single inbox, waits for its goroutine
// to exit, then closes the inbox. Caller must NOT hold m.mu.
func (m *Manager) stopInbox(id int) {
	m.mu.Lock()
	rs, hasReceiver := m.receivers[id]
	if hasReceiver {
		rs.cancel()
		delete(m.receivers, id)
	}
	m.mu.Unlock()

	// Wait outside lock so the receiver goroutine can finish.
	if hasReceiver {
		<-rs.done
	}

	m.mu.Lock()
	if inb, ok := m.inboxes[id]; ok {
		inb.Close()
		delete(m.inboxes, id)
	}
	m.mu.Unlock()
}

// startReceiver starts a receiver goroutine for the given inbox.
// Caller must hold m.mu.
func (m *Manager) startReceiver(ctx context.Context, inb Inbox) {
	done := make(chan struct{})
	receiverCtx, cancel := context.WithCancel(ctx)
	m.receivers[inb.Identifier()] = receiverState{cancel: cancel, done: done}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(done)
		if err := inb.Receive(receiverCtx); err != nil {
			m.lo.Error("error starting inbox receiver", "error", err)
		}
	}()
}

// Start starts the receiver for each inbox.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inb := range m.inboxes {
		m.startReceiver(ctx, inb)
	}
	return nil
}

// Close closes all inboxes.
func (m *Manager) Close() {
	m.mu.Lock()

	// Cancel all receivers.
	for _, rs := range m.receivers {
		rs.cancel()
	}

	// Close all inboxes.
	for _, inb := range m.inboxes {
		inb.Close()
	}
	m.mu.Unlock()

	// Wait for all receiver goroutines to finish.
	m.wg.Wait()
}

// getActive returns all active inboxes from the DB.
func (m *Manager) getActive() ([]imodels.Inbox, error) {
	var inboxes []imodels.Inbox
	if err := m.queries.GetActive.Select(&inboxes); err != nil {
		m.lo.Error("fetching active inboxes", "error", err)
		return nil, err
	}

	// Decrypt sensitive fields in each inbox config
	for i := range inboxes {
		decryptedConfig, err := m.decryptInboxConfig(inboxes[i].Config)
		if err != nil {
			m.lo.Error("error decrypting inbox config", "id", inboxes[i].ID, "error", err)
			return nil, fmt.Errorf("decrypting inbox config for ID %d: %w", inboxes[i].ID, err)
		}
		inboxes[i].Config = decryptedConfig

		// Decrypt secret field
		m.decryptInboxSecret(&inboxes[i])
	}

	return inboxes, nil
}

// encryptInboxConfig encrypts sensitive fields in the inbox config JSON.
func (m *Manager) encryptInboxConfig(config json.RawMessage) (json.RawMessage, error) {
	if len(config) == 0 {
		return config, nil
	}

	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	// Encrypt SMTP passwords
	if smtpSlice, ok := cfg["smtp"].([]any); ok {
		for i, smtpItem := range smtpSlice {
			if smtpMap, ok := smtpItem.(map[string]any); ok {
				if password, ok := smtpMap["password"].(string); ok && password != "" {
					encrypted, err := crypto.Encrypt(password, m.encryptionKey)
					if err != nil {
						return nil, fmt.Errorf("encrypting SMTP password at index %d: %w", i, err)
					}
					smtpMap["password"] = encrypted
				}
			}
		}
	}

	// Encrypt IMAP passwords
	if imapSlice, ok := cfg["imap"].([]any); ok {
		for i, imapItem := range imapSlice {
			if imapMap, ok := imapItem.(map[string]any); ok {
				if password, ok := imapMap["password"].(string); ok && password != "" {
					encrypted, err := crypto.Encrypt(password, m.encryptionKey)
					if err != nil {
						return nil, fmt.Errorf("encrypting IMAP password at index %d: %w", i, err)
					}
					imapMap["password"] = encrypted
				}
			}
		}
	}

	// Encrypt OAuth fields if present
	if oauthMap, ok := cfg["oauth"].(map[string]any); ok {
		fields := []string{"client_secret", "access_token", "refresh_token"}
		for _, fieldName := range fields {
			if fieldValue, ok := oauthMap[fieldName].(string); ok && fieldValue != "" {
				encrypted, err := crypto.Encrypt(fieldValue, m.encryptionKey)
				if err != nil {
					return nil, fmt.Errorf("encrypting OAuth %s: %w", fieldName, err)
				}
				oauthMap[fieldName] = encrypted
			}
		}
	}

	encrypted, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshalling encrypted config: %w", err)
	}

	return encrypted, nil
}

// Decrypt failures clear the field so the app stays usable across encryption_key rotation.
func (m *Manager) decryptInboxConfig(config json.RawMessage) (json.RawMessage, error) {
	if len(config) == 0 {
		return config, nil
	}

	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	if smtpSlice, ok := cfg["smtp"].([]any); ok {
		for i, smtpItem := range smtpSlice {
			if smtpMap, ok := smtpItem.(map[string]any); ok {
				if password, ok := smtpMap["password"].(string); ok && password != "" {
					decrypted, err := crypto.Decrypt(password, m.encryptionKey)
					if err != nil {
						m.lo.Error("error decrypting SMTP password, clearing field", "index", i, "error", err)
						smtpMap["password"] = ""
						continue
					}
					smtpMap["password"] = decrypted
				}
			}
		}
	}

	if imapSlice, ok := cfg["imap"].([]any); ok {
		for i, imapItem := range imapSlice {
			if imapMap, ok := imapItem.(map[string]any); ok {
				if password, ok := imapMap["password"].(string); ok && password != "" {
					decrypted, err := crypto.Decrypt(password, m.encryptionKey)
					if err != nil {
						m.lo.Error("error decrypting IMAP password, clearing field", "index", i, "error", err)
						imapMap["password"] = ""
						continue
					}
					imapMap["password"] = decrypted
				}
			}
		}
	}

	if oauthMap, ok := cfg["oauth"].(map[string]any); ok {
		fields := []string{"client_secret", "access_token", "refresh_token"}
		for _, fieldName := range fields {
			if fieldValue, ok := oauthMap[fieldName].(string); ok && fieldValue != "" {
				decrypted, err := crypto.Decrypt(fieldValue, m.encryptionKey)
				if err != nil {
					m.lo.Error("error decrypting OAuth field, clearing field", "field", fieldName, "error", err)
					oauthMap[fieldName] = ""
					continue
				}
				oauthMap[fieldName] = decrypted
			}
		}
	}

	decrypted, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshalling decrypted config: %w", err)
	}

	return decrypted, nil
}

// decryptInboxSecret decrypts the inbox secret field if present.
func (m *Manager) decryptInboxSecret(inbox *imodels.Inbox) {
	if inbox.Secret.Valid && inbox.Secret.String != "" {
		decrypted, err := crypto.Decrypt(inbox.Secret.String, m.encryptionKey)
		if err != nil {
			m.lo.Error("error decrypting inbox secret", "inbox_id", inbox.ID, "error", err)
			return
		}
		inbox.Secret = null.StringFrom(decrypted)
	}
}
