// Package conversation manages conversations and messages.
package conversation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/knadh/go-i18n"

	"io"
	"net/url"
	"slices"

	"strings"
	"sync"
	"time"

	"github.com/jakedolan443/fernmail/internal/authz"
	authzmodels "github.com/jakedolan443/fernmail/internal/authz/models"

	"github.com/jakedolan443/fernmail/internal/conversation/models"

	smodels "github.com/jakedolan443/fernmail/internal/conversation/status/models"

	"github.com/jakedolan443/fernmail/internal/dbutil"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/inbox"
	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"

	"github.com/jakedolan443/fernmail/internal/stringutil"

	"github.com/jakedolan443/fernmail/internal/template"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	wmodels "github.com/jakedolan443/fernmail/internal/webhook/models"
	"github.com/jakedolan443/fernmail/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"

	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs                             embed.FS
	errConversationNotFound         = errors.New("conversation not found")
	ErrConversationAlreadyAssigned  = errors.New("conversation already assigned")
	conversationsAllowedFields      = []string{"status_id", "inbox_id", "address_id", "last_message_at", "last_interaction_at", "last_interaction_sender", "created_at", "waiting_since", "snoozed_until"}
	conversationStatusAllowedFields = []string{"id", "name"}
	usersAllowedFields              = []string{"email", "external_user_id"}
	inboxesAllowedFields            = []string{"channel"}
)

const (
	conversationsListMaxPageSize = 500
)

// ListFilterRenderers remains a stable integration point for generic list and
// search filtering. Address routing is now a database field, not a legacy
// metadata renderer.
var ListFilterRenderers = dbutil.FieldRenderers{}

var ListFilterAllowedFields = dbutil.AllowedFields{
	"conversations":         conversationsAllowedFields,
	"conversation_statuses": conversationStatusAllowedFields,
	"users":                 usersAllowedFields,
	"inboxes":               inboxesAllowedFields,
}

// Manager handles the operations related to conversations
type Manager struct {
	cacheIncomingImages        func(context.Context, int, string) error
	q                          queries
	inboxStore                 inboxStore
	userStore                  userStore
	mediaStore                 mediaStore
	statusStore                statusStore
	settingsStore              settingsStore
	webhookStore               webhookStore
	lo                         *logf.Logger
	db                         *sqlx.DB
	i18n                       *i18n.I18n
	wsHub                      *ws.Hub
	template                   *template.Manager
	incomingMessageQueue       chan models.IncomingMessage
	outgoingMessageQueue       chan models.Message
	outgoingProcessingMessages sync.Map
	stopCh                     chan struct{}
	closed                     bool
	closedMu                   sync.RWMutex
	wg                         sync.WaitGroup
	subjectRefFormat           string
}

type statusStore interface {
	Get(int) (smodels.Status, error)
}

type userStore interface {
	Get(int, string, []string) (umodels.User, error)
	GetAgentCachedOrLoad(int) (umodels.User, error)
	GetSystemUser() (umodels.User, error)
	ResolveEmailSender(user *umodels.User) error
}

type mediaStore interface {
	Get(id int, uuid string) (mmodels.Media, error)
	GetBlob(name string) ([]byte, error)
	GetURL(uuid, contentType, fileName string) string
	GetSignedURL(name string) string
	GetThumbnailURL(uuid string) string
	LinkMessageMediaTx(tx *sqlx.Tx, messageID int, media []mmodels.Media, inlineUUIDs []string, uploadUserID int) error
	GetByModel(id int, model string) ([]mmodels.Media, error)
	GetByContentIDs(contentIDs []string, conversationUUID string) ([]mmodels.Media, error)
	GetDraftInlineMedia(uuid string, conversationID, userID int) (mmodels.Media, error)
	ContentIDExists(contentID, conversationUUID string) (bool, string, error)
	Upload(fileName, contentType string, content io.ReadSeeker) (string, string, error)
	UploadAndInsert(fileName, contentType, contentID string, modelType null.String, modelID null.Int, content io.ReadSeeker, fileSize int, disposition null.String, meta []byte, private bool) (mmodels.Media, error)
}

type inboxStore interface {
	Get(int) (inbox.Inbox, error)
	GetDBRecord(any) (imodels.Inbox, error)
	GetAll() ([]imodels.Inbox, error)
}

type settingsStore interface {
	GetAppRootURL() (string, error)
	GetByPrefix(prefix string) (types.JSONText, error)
	Get(key string) (types.JSONText, error)
}

type webhookStore interface {
	TriggerEvent(event wmodels.WebhookEvent, data any)
	TriggerWebhook(webhookID int, event wmodels.WebhookEvent, data any)
}

// Opts holds the options for creating a new Manager.
type Opts struct {
	CacheIncomingImages      func(context.Context, int, string) error
	DB                       *sqlx.DB
	Lo                       *logf.Logger
	OutgoingMessageQueueSize int
	IncomingMessageQueueSize int
	SubjectRefFormat         string
}

// New initializes a new conversation Manager.
func New(
	wsHub *ws.Hub,
	i18n *i18n.I18n,
	statusStore statusStore,
	inboxStore inboxStore,
	userStore userStore,
	mediaStore mediaStore,
	settingsStore settingsStore,
	template *template.Manager,
	webhook webhookStore,
	opts Opts) (*Manager, error) {

	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	subjectRefFormat := opts.SubjectRefFormat
	if subjectRefFormat == "" {
		subjectRefFormat = "#{ref}"
	} else if !strings.Contains(subjectRefFormat, "{ref}") {
		return nil, fmt.Errorf("conversation.subject_ref_format must contain {ref} placeholder")
	}

	c := &Manager{
		cacheIncomingImages:        opts.CacheIncomingImages,
		q:                          q,
		wsHub:                      wsHub,
		i18n:                       i18n,
		inboxStore:                 inboxStore,
		userStore:                  userStore,
		mediaStore:                 mediaStore,
		settingsStore:              settingsStore,
		webhookStore:               webhook,
		statusStore:                statusStore,
		template:                   template,
		db:                         opts.DB,
		lo:                         opts.Lo,
		stopCh:                     make(chan struct{}),
		incomingMessageQueue:       make(chan models.IncomingMessage, opts.IncomingMessageQueueSize),
		outgoingMessageQueue:       make(chan models.Message, opts.OutgoingMessageQueueSize),
		outgoingProcessingMessages: sync.Map{},
		subjectRefFormat:           subjectRefFormat,
	}

	return c, nil
}

type queries struct {
	// Conversation queries.
	GetConversationUUID               *sqlx.Stmt `query:"get-conversation-uuid"`
	GetConversation                   *sqlx.Stmt `query:"get-conversation"`
	GetConversationListItem           *sqlx.Stmt `query:"get-conversation-list-item"`
	GetConversationsCreatedAfter      *sqlx.Stmt `query:"get-conversations-created-after"`
	GetConversations                  string     `query:"get-conversations"`
	GetConversationParticipants       *sqlx.Stmt `query:"get-conversation-participants"`
	StartConversationWaitingSince     *sqlx.Stmt `query:"start-conversation-waiting-since"`
	UpdateConversationReplyTimestamps *sqlx.Stmt `query:"update-conversation-reply-timestamps"`
	UpdateConversationContactLastSeen *sqlx.Stmt `query:"update-conversation-contact-last-seen"`
	UpsertUserLastSeen                *sqlx.Stmt `query:"upsert-user-last-seen"`
	MarkConversationUnread            *sqlx.Stmt `query:"mark-conversation-unread"`
	UpdateConversationStatus          *sqlx.Stmt `query:"update-conversation-status"`
	UpdateConversationLastMessage     *sqlx.Stmt `query:"update-conversation-last-message"`
	InsertConversationParticipant     *sqlx.Stmt `query:"insert-conversation-participant"`
	InsertConversation                *sqlx.Stmt `query:"insert-conversation"`
	ReOpenConversation                *sqlx.Stmt `query:"re-open-conversation"`
	UnsnoozeAll                       *sqlx.Stmt `query:"unsnooze-all"`
	DeleteConversation                *sqlx.Stmt `query:"delete-conversation"`

	// Draft queries.
	UpsertConversationDraft *sqlx.Stmt `query:"upsert-conversation-draft"`
	GetAllUserDrafts        *sqlx.Stmt `query:"get-all-user-drafts"`
	DeleteConversationDraft *sqlx.Stmt `query:"delete-conversation-draft"`
	DeleteStaleDrafts       *sqlx.Stmt `query:"delete-stale-drafts"`

	// Message queries.
	GetMessage                         *sqlx.Stmt `query:"get-message"`
	GetMessages                        string     `query:"get-messages"`
	GetOutgoingPendingMessages         *sqlx.Stmt `query:"get-outgoing-pending-messages"`
	GetMessageSourceIDs                *sqlx.Stmt `query:"get-message-source-ids"`
	GetConversationUUIDFromMessageUUID *sqlx.Stmt `query:"get-conversation-uuid-from-message-uuid"`
	MessageExistsBySourceID            *sqlx.Stmt `query:"message-exists-by-source-id"`
	GetConversationByMessageID         *sqlx.Stmt `query:"get-conversation-by-message-id"`
	InsertMessage                      *sqlx.Stmt `query:"insert-message"`
	UpdateMessageStatus                *sqlx.Stmt `query:"update-message-status"`
	DeletePrivateMessage               *sqlx.Stmt `query:"delete-private-message"`

	// Conversation continuity queries.

	// Mention queries.
	InsertMention *sqlx.Stmt `query:"insert-mention"`

	// Broadcast queries.

	// WS list-subscribe authz.
	FilterAuthorizedListUUIDs *sqlx.Stmt `query:"filter-authorized-list-uuids"`
}

// CreateConversation creates a new conversation. If maxConversations > 0, the insert is
// atomically rejected when the contact already has >= maxConversations in the given window.
func (c *Manager) CreateConversation(contactID, inboxID, addressID int, lastMessage string, lastMessageAt time.Time, subject string, appendRefNumToSubject bool, meta, customAttributes map[string]any, maxConversations int, rateLimitWindow time.Duration) (int, string, error) {
	var (
		id     int
		uuid   string
		prefix string
	)

	if meta == nil {
		meta = map[string]any{}
	}
	if customAttributes == nil {
		customAttributes = map[string]any{}
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		c.lo.Error("error marshalling conversation meta", "error", err)
		return 0, "", err
	}
	customAttrsJSON, err := json.Marshal(customAttributes)
	if err != nil {
		c.lo.Error("error marshalling conversation custom attributes", "error", err)
		return 0, "", err
	}

	var since time.Time
	if maxConversations > 0 {
		since = time.Now().Add(-rateLimitWindow)
	}

	if err := c.q.InsertConversation.QueryRow(contactID, models.StatusOpen, inboxID, lastMessage, lastMessageAt, subject, prefix, appendRefNumToSubject, metaJSON, customAttrsJSON, since, maxConversations, c.subjectRefFormat, addressID).Scan(&id, &uuid); err != nil {
		if err == sql.ErrNoRows {
			return 0, "", envelope.NewError(envelope.RateLimitError, c.i18n.T("globals.messages.tooManyRequests"), nil)
		}
		c.lo.Error("error inserting new conversation into the DB", "error", err)
		return 0, "", err
	}
	// The first message owns publication: failed/duplicate ingestion may still
	// remove this provisional conversation before any mail has committed.
	return id, uuid, nil
}

// GetConversation retrieves a conversation by its ID or UUID.
func (c *Manager) GetConversation(id int, uuid, refNum string) (models.Conversation, error) {
	var conversation models.Conversation
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	if err := c.q.GetConversation.Get(&conversation, id, uuidParam, refNum); err != nil {
		if err == sql.ErrNoRows {
			return conversation, envelope.NewError(envelope.NotFoundError,
				c.i18n.T("validation.notFoundConversation"), nil)
		}
		c.lo.Error("error fetching conversation", "error", err)
		return conversation, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Strip name and extract plain email from "Name <email>"
	if conversation.InboxMail != "" {
		var err error
		conversation.InboxMail, err = stringutil.ExtractEmail(conversation.InboxMail)
		if err != nil {
			c.lo.Error("error extracting email from inbox mail", "inbox_mail", conversation.InboxMail, "error", err)
		}
	}

	return conversation, nil
}

// SignAvatarURL converts a raw /uploads/ avatar path to a signed URL.
func (c *Manager) SignAvatarURL(avatarURL *null.String) {
	if avatarURL == nil || !avatarURL.Valid || avatarURL.String == "" {
		return
	}
	if strings.HasPrefix(avatarURL.String, "/uploads/") {
		parsed, err := url.Parse(avatarURL.String)
		if err != nil {
			*avatarURL = null.String{}
			return
		}
		*avatarURL = null.StringFrom(c.mediaStore.GetSignedURL(strings.TrimPrefix(parsed.Path, "/uploads/")))
	}
}

// GetConversationsCreatedAfter retrieves a batch of conversation refs created after the given time, keyset-paged by id.
func (c *Manager) GetConversationsCreatedAfter(after time.Time, afterID, limit int) ([]models.ConversationRef, error) {
	var refs = make([]models.ConversationRef, 0, limit)
	if err := c.q.GetConversationsCreatedAfter.Select(&refs, after, afterID, limit); err != nil {
		c.lo.Error("error fetching conversation refs", "error", err)
		return refs, err
	}
	return refs, nil
}

// UpdateUserLastSeen updates the last seen timestamp for a specific user on a conversation.
func (c *Manager) UpdateUserLastSeen(uuid string, userID int) error {
	if _, err := c.q.UpsertUserLastSeen.Exec(userID, uuid); err != nil {
		c.lo.Error("error upserting user last seen", "user_id", userID, "conversation_uuid", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// MarkAsUnread marks a conversation as unread for a specific user by setting last_seen to before the last message.
func (c *Manager) MarkAsUnread(uuid string, userID int) error {
	if _, err := c.q.MarkConversationUnread.Exec(userID, uuid); err != nil {
		c.lo.Error("error marking conversation as unread", "user_id", userID, "conversation_uuid", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// UpdateContactLastSeen updates the last seen timestamp of the contact in the conversation.
func (c *Manager) UpdateConversationContactLastSeen(uuid string) error {
	var lastSeenAt time.Time
	if err := c.q.UpdateConversationContactLastSeen.Get(&lastSeenAt, uuid); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		c.lo.Error("error updating contact last seen timestamp", "conversation_id", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Broadcast the property update to all subscribers.
	c.BroadcastConversationUpdate(uuid, map[string]any{"contact_last_seen_at": lastSeenAt.Format(time.RFC3339Nano)})
	return nil
}

// GetConversationParticipants retrieves the participants of a conversation.
func (c *Manager) GetConversationParticipants(uuid string) ([]models.ConversationParticipant, error) {
	conv := make([]models.ConversationParticipant, 0)
	if err := c.q.GetConversationParticipants.Select(&conv, uuid); err != nil {
		c.lo.Error("error fetching conversation", "error", err)
		return conv, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return conv, nil
}

// GetConversationUUID retrieves the UUID of a conversation by its ID.
func (c *Manager) GetConversationUUID(id int) (string, error) {
	var uuid string
	if err := c.q.GetConversationUUID.QueryRow(id).Scan(&uuid); err != nil {
		if err == sql.ErrNoRows {
			return uuid, err
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return uuid, err
	}
	return uuid, nil
}

// InsertMentions inserts mentions for a message.
func (c *Manager) InsertMentions(conversationID, messageID, mentionedByUserID int, mentions []models.MentionInput) error {
	for _, mention := range mentions {
		var userID, teamID any
		switch mention.Type {
		case models.MentionTypeAgent:
			userID = mention.ID
		case models.MentionTypeTeam:
			teamID = mention.ID
		default:
			c.lo.Warn("invalid mention type, skipping", "type", mention.Type)
			continue
		}

		if _, err := c.q.InsertMention.Exec(conversationID, messageID, userID, teamID, mentionedByUserID); err != nil {
			c.lo.Error("error inserting mention", "error", err)
		}
	}
	return nil
}

// GetConversations retrieves conversations list based on user ID, type, and optional filtering, ordering, and pagination.
// viewingUserID is used to calculate per-agent unread counts.
func (c *Manager) GetConversations(viewingUserID, userID int, teamIDs []int, listTypes []string, order, orderBy, filters string, page, pageSize int) ([]models.ConversationListItem, error) {
	var conversations = make([]models.ConversationListItem, 0)

	// Make the query.
	query, qArgs, err := c.makeConversationsListQuery(viewingUserID, userID, teamIDs, listTypes, c.q.GetConversations, order, orderBy, page, pageSize, filters)
	if err != nil {
		c.lo.Error("error making conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		c.lo.Error("error preparing get conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	defer tx.Rollback()

	if err := tx.Select(&conversations, query, qArgs...); err != nil {
		c.lo.Error("error fetching conversations", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return conversations, nil
}

// ReOpenConversation reopens a conversation if it's snoozed, resolved or closed.
func (c *Manager) ReOpenConversation(conversationUUID string, actor umodels.User) (bool, error) {
	var conversationID int
	if err := c.q.ReOpenConversation.QueryRow(conversationUUID).Scan(&conversationID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		c.lo.Error("error reopening conversation", "uuid", conversationUUID, "error", err)
		return false, envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Reopening revives any first response or resolution deadline the resolved status had nulled.

	c.BroadcastConversationUpdate(conversationUUID, map[string]any{"status": models.StatusOpen})

	if err := c.RecordStatusChange(models.StatusOpen, conversationUUID, actor); err != nil {
		return true, err
	}
	return true, nil
}

// UpdateConversationLastMessage updates the last message details for a conversation.
// Also conditionally updates last_interaction fields if messageType != 'activity' and !private.
func (c *Manager) UpdateConversationLastMessage(conversation int, conversationUUID, lastMessage, lastMessageSenderType, messageType string, private bool, lastMessageAt time.Time, senderID int) error {
	if _, err := c.q.UpdateConversationLastMessage.Exec(conversation, conversationUUID, lastMessage, lastMessageSenderType, lastMessageAt, messageType, private, senderID); err != nil {
		c.lo.Error("error updating conversation last message", "error", err)
		return err
	}
	return nil
}

// StartConversationWaitingSince stamps the waiting since timestamp only if the conversation isn't already waiting.
func (c *Manager) StartConversationWaitingSince(conversationUUID string, at time.Time) error {
	res, err := c.q.StartConversationWaitingSince.Exec(conversationUUID, at)
	if err != nil {
		c.lo.Error("error updating conversation waiting since", "error", err)
		return err
	}

	if rows, _ := res.RowsAffected(); rows > 0 {
		c.BroadcastConversationUpdate(conversationUUID, map[string]any{"waiting_since": at.Format(time.RFC3339)})
	}
	return nil
}

// UpdateConversationStatus updates the status of a conversation.
func (c *Manager) UpdateConversationStatus(uuid string, statusID int, status, snoozeDur string, actor umodels.User) error {
	// Fetch the status name if status ID is provided.
	if statusID > 0 {
		s, err := c.statusStore.Get(statusID)
		if err != nil {
			return envelope.NewError(envelope.InputError, err.Error(), nil)
		}
		status = s.Name
	}

	if status == models.StatusSnoozed && snoozeDur == "" {
		return envelope.NewError(envelope.InputError, c.i18n.T("validation.invalidSnoozeDuration"), nil)
	}

	// Parse the snooze duration if status is snoozed.
	snoozeUntil := time.Time{}
	if status == models.StatusSnoozed {
		duration, err := time.ParseDuration(snoozeDur)
		if err != nil || duration <= 0 {
			c.lo.Error("error parsing snooze duration", "duration", snoozeDur, "error", err)
			return envelope.NewError(envelope.InputError, c.i18n.T("validation.invalidSnoozeDuration"), nil)
		}
		snoozeUntil = time.Now().Add(duration)
	}

	conversationBeforeChange, err := c.GetConversation(0, uuid, "")
	if err != nil {
		c.lo.Error("error fetching conversation before status change", "uuid", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	oldStatus := conversationBeforeChange.Status.String

	// Status not changed and not snoozed. Return early.
	if oldStatus == status && status != models.StatusSnoozed {
		c.lo.Debug("no status update: conversation status unchanged and not snoozed", "uuid", uuid, "old_status", oldStatus, "new_status", status)
		return nil
	}

	// Update the conversation status.
	if _, err := c.q.UpdateConversationStatus.Exec(uuid, status, snoozeUntil); err != nil {
		c.lo.Error("error updating conversation status", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Stamps a just-resolved resolution SLA immediately and recomputes the cached deadline.

	// Fetch conversation for webhook and automation rules.
	conversation, err := c.GetConversation(0, uuid, "")
	if err != nil {
		c.lo.Error("error fetching conversation after status change", "uuid", uuid, "error", err)
	}

	// Trigger webhook for conversation status change
	var snoozeUntilStr string
	if !snoozeUntil.IsZero() {
		snoozeUntilStr = snoozeUntil.UTC().Format(time.RFC3339)
	}
	c.webhookStore.TriggerEvent(wmodels.EventConversationStatusChanged, map[string]any{
		"conversation_uuid": uuid,
		"previous_status":   oldStatus,
		"new_status":        status,
		"snooze_until":      snoozeUntilStr,
		"actor_id":          actor.ID,
		"conversation":      conversation,
	})

	// Record the status change as an activity.
	if err := c.RecordStatusChange(status, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	agentData := map[string]any{"status": status}
	if oldStatus != models.StatusResolved && status == models.StatusResolved {
		resolvedAt := conversationBeforeChange.ResolvedAt.Time
		if resolvedAt.IsZero() {
			resolvedAt = time.Now()
		}
		agentData["resolved_at"] = resolvedAt.Format(time.RFC3339)

	}
	if status == models.StatusClosed {
		closedAt := time.Now()
		if conversation.ID != 0 && !conversation.ClosedAt.Time.IsZero() {
			closedAt = conversation.ClosedAt.Time
		}
		agentData["closed_at"] = closedAt.Format(time.RFC3339)
	}
	if status == models.StatusSnoozed {
		agentData["snoozed_until"] = snoozeUntil.Format(time.RFC3339)
	} else if oldStatus == models.StatusSnoozed {
		agentData["snoozed_until"] = nil
	}
	c.BroadcastConversationUpdate(uuid, agentData)

	if conversation.ID != 0 {
	}

	// Broadcast conversation update to widget clients.

	return nil
}

// GetMessageSourceIDs retrieves source IDs for messages in a conversation in descending order.
// So the oldest message will be the last in the list.
func (m *Manager) GetMessageSourceIDs(conversationID, limit int) ([]string, error) {
	var refs []string
	if err := m.q.GetMessageSourceIDs.Select(&refs, conversationID, limit); err != nil {
		m.lo.Error("error fetching message source IDs", "conversation_id", conversationID, "error", err)
		return refs, err
	}
	return refs, nil
}

// BuildEmailThreadingHeaders builds References and In-Reply-To headers for an outgoing email,
// excluding the message's own source ID.
func (m *Manager) BuildEmailThreadingHeaders(conversationID int, selfSourceID string) ([]string, string) {
	var current struct {
		ID     int    `db:"id"`
		Parent string `db:"reply_to_source_id"`
	}
	if err := m.db.Get(&current, `SELECT id,reply_to_source_id FROM conversation_messages WHERE conversation_id=$1 AND source_id=$2 ORDER BY id DESC LIMIT 1`, conversationID, selfSourceID); err != nil {
		return nil, ""
	}
	var references []string
	// A reply's parent is fixed at queue time. Never point at a future pending
	// or failed reply simply because another agent typed at the same time.
	if err := m.db.Select(&references, `SELECT source_id FROM conversation_messages WHERE conversation_id=$1 AND id<$2 AND NOT private AND type IN ('incoming','outgoing') AND status IN ('received','sent') AND source_id>'' AND ($3='' OR id<=(SELECT id FROM conversation_messages WHERE conversation_id=$1 AND source_id=$3 ORDER BY id DESC LIMIT 1)) ORDER BY id DESC LIMIT 20`, conversationID, current.ID, current.Parent); err != nil {
		return nil, ""
	}
	slices.Reverse(references)
	parent := current.Parent
	if parent == "" && len(references) > 0 {
		parent = references[len(references)-1]
	}
	return references, parent
}

// DeleteConversation deletes a conversation.
func (m *Manager) DeleteConversation(uuid string) error {
	res, err := m.q.DeleteConversation.Exec(uuid)
	if err != nil {
		m.lo.Error("error deleting conversation", "uuid", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	rows, _ := res.RowsAffected()
	m.lo.Info("deleted conversation", "uuid", uuid, "rows_affected", rows)
	return nil
}

// addConversationParticipant adds a user as participant to a conversation.
func (c *Manager) addConversationParticipant(userID int, conversationUUID string) error {
	res, err := c.q.InsertConversationParticipant.Exec(userID, conversationUUID)
	if err != nil {
		c.lo.Error("error adding conversation participant", "user_id", userID, "conversation_uuid", conversationUUID, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return nil
	}

	// New participant added - log activity only for contacts with a different email than the conversation contact.
	conversation, convErr := c.GetConversation(0, conversationUUID, "")
	if convErr == nil {
		user, userErr := c.userStore.Get(userID, "", []string{})
		if userErr == nil && user.Type == umodels.UserTypeContact && !strings.EqualFold(user.Email.String, conversation.Contact.Email.String) {
			participantName := user.Email.String
			if user.FirstName != "" {
				participantName = user.FirstName
				if user.LastName != "" {
					participantName += " " + user.LastName
				}
				participantName += " (" + user.Email.String + ")"
			}
			systemUser, sysErr := c.userStore.GetSystemUser()
			if sysErr == nil {
				c.InsertConversationActivity(models.ActivityParticipantAdded, conversationUUID, participantName, systemUser)
			}
		}
	}

	return nil
}

// makeConversationsListQuery prepares a SQL query string for conversations list
// viewingUserID is used as $1 for per-agent unread count calculation
// $2 is includeMentions bool for conditional mentioned_message_uuid column
func (c *Manager) makeConversationsListQuery(viewingUserID, userID int, teamIDs []int, listTypes []string, baseQuery, order, orderBy string, page, pageSize int, filtersJSON string) (string, []interface{}, error) {
	includeMentions := slices.Contains(listTypes, models.MentionedConversations)
	qArgs := []any{viewingUserID, includeMentions}

	// Set defaults
	if orderBy == "" {
		orderBy = "conversations.last_message_at"
	}
	if order == "" {
		order = "DESC"
	}
	if filtersJSON == "" {
		filtersJSON = "[]"
	}

	// Validate inputs
	if pageSize > conversationsListMaxPageSize {
		pageSize = conversationsListMaxPageSize
	}

	if len(listTypes) == 0 {
		return "", nil, fmt.Errorf("no conversation list types specified")
	}

	// Prepare the conditions based on the list types.
	conditions, err := appendListTypeConditions(listTypes, viewingUserID, userID, teamIDs, &qArgs)
	if err != nil {
		return "", nil, err
	}

	// Build the base query with list type conditions
	whereClause := listTypeWhereClause(conditions)

	baseQuery = fmt.Sprintf(baseQuery, whereClause)

	return dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
		Location: c.FilterLocation(),
	}, filtersJSON, ListFilterAllowedFields, ListFilterRenderers)
}

func (c *Manager) GetConversationListItem(uuid string) (models.ConversationListItem, error) {
	var item models.ConversationListItem
	if err := c.q.GetConversationListItem.Get(&item, uuid); err != nil {
		return item, err
	}
	return item, nil
}

func (c *Manager) AuthorizedConnectedAgentIDs(assignedUserID, assignedTeamID, addressID null.Int, inboxID int) []int {
	connected := c.wsHub.ConnectedUserIDs()
	if len(connected) == 0 {
		return nil
	}
	out := make([]int, 0, len(connected))
	for _, id := range connected {
		agent, err := c.userStore.GetAgentCachedOrLoad(id)
		if err != nil {
			continue
		}
		if !agent.Enabled {
			continue
		}
		var allowed bool
		if addressID.Valid {
			err = c.db.Get(&allowed, `SELECT can_access_email_address($1,$2)`, addressID.Int, id)
		} else {
			err = c.db.Get(&allowed, `SELECT can_access_inbox($1,$2)`, inboxID, id)
		}
		if err != nil || !allowed {
			continue
		}
		if authz.CanReadAssignment(agent, assignedUserID, assignedTeamID) {
			out = append(out, id)
		}
	}
	return out
}

// FilterAuthorizedListUUIDs returns the subset of UUIDs the agent can read, mirroring the conversation read-permission chain.
func (c *Manager) FilterAuthorizedListUUIDs(agentID int, uuids []string) ([]string, error) {
	if len(uuids) == 0 {
		return nil, nil
	}
	user, err := c.userStore.GetAgentCachedOrLoad(agentID)
	if err != nil {
		return nil, err
	}
	if !user.Enabled {
		return nil, nil
	}
	var authorized []string
	err = c.q.FilterAuthorizedListUUIDs.Select(&authorized,
		pq.Array(uuids),
		user.ID,
		pq.Array(user.Teams.IDs()),
		slices.Contains(user.Permissions, authzmodels.PermConversationsRead),
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadAll),
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadAssigned),
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadTeamAll),
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadTeamInbox),
		slices.Contains(user.Permissions, authzmodels.PermConversationsReadUnassigned),
	)
	if err != nil {
		c.lo.Error("error filtering authorized list uuids", "agent_id", agentID, "error", err)
		return nil, err
	}
	return authorized, nil
}

// filterLocation returns the configured app timezone for resolving date filters. The builder normalizes invalid/empty values to UTC.
func (c *Manager) FilterLocation() string {
	b, err := c.settingsStore.Get("app.timezone")
	if err != nil {
		return ""
	}
	var tz string
	if err := json.Unmarshal(b, &tz); err != nil {
		return ""
	}
	return tz
}
