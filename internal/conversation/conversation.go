// Package conversation manages conversations and messages.
package conversation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation"
	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	pmodels "github.com/abhinavxd/libredesk/internal/conversation/priority/models"
	smodels "github.com/abhinavxd/libredesk/internal/conversation/status/models"
	csatModels "github.com/abhinavxd/libredesk/internal/csat/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	slaModels "github.com/abhinavxd/libredesk/internal/sla/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/template"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs                             embed.FS
	errConversationNotFound         = errors.New("conversation not found")
	conversationsAllowedFields      = []string{"status_id", "priority_id", "assigned_team_id", "assigned_user_id", "inbox_id", "last_message_at", "created_at", "waiting_since", "next_sla_deadline_at", "priority_id"}
	conversationStatusAllowedFields = []string{"id", "name"}
	csatReplyMessage                = "Please rate your experience with us: <a href=\"%s\">Rate now</a>"
)

const (
	conversationsListMaxPageSize = 100
)

// Manager handles the operations related to conversations
type Manager struct {
	q                          queries
	inboxStore                 inboxStore
	userStore                  userStore
	teamStore                  teamStore
	mediaStore                 mediaStore
	statusStore                statusStore
	priorityStore              priorityStore
	slaStore                   slaStore
	settingsStore              settingsStore
	csatStore                  csatStore
	webhookStore               webhookStore
	notifier                   *notifier.Service
	lo                         *logf.Logger
	db                         *sqlx.DB
	i18n                       *i18n.I18n
	automation                 *automation.Engine
	wsHub                      *ws.Hub
	template                   *template.Manager
	incomingMessageQueue       chan models.IncomingMessage
	outgoingMessageQueue       chan models.Message
	outgoingProcessingMessages sync.Map
	closed                     bool
	closedMu                   sync.RWMutex
	wg                         sync.WaitGroup
}

type slaStore interface {
	ApplySLA(startTime time.Time, conversationID, assignedTeamID, slaID int) (slaModels.SLAPolicy, error)
	CreateNextResponseSLAEvent(conversationID, appliedSLAID, slaPolicyID, assignedTeamID int) (time.Time, error)
	SetLatestSLAEventMetAt(appliedSLAID int, metric string) (time.Time, error)
}

type statusStore interface {
	Get(int) (smodels.Status, error)
}

type priorityStore interface {
	Get(int) (pmodels.Priority, error)
}

type teamStore interface {
	Get(int) (tmodels.Team, error)
	UserBelongsToTeam(userID, teamID int) (bool, error)
}

type userStore interface {
	GetAgent(int, string) (umodels.User, error)
	GetSystemUser() (umodels.User, error)
	CreateContact(user *umodels.User) error
}

type mediaStore interface {
	GetBlob(name string) ([]byte, error)
	Attach(id int, model string, modelID int) error
	GetByModel(id int, model string) ([]mmodels.Media, error)
	ContentIDExists(contentID string) (bool, string, error)
	Upload(fileName, contentType string, content io.ReadSeeker) (string, error)
	UploadAndInsert(fileName, contentType, contentID string, modelType null.String, modelID null.Int, content io.ReadSeeker, fileSize int, disposition null.String, meta []byte) (mmodels.Media, error)
}

type inboxStore interface {
	Get(int) (inbox.Inbox, error)
	GetDBRecord(int) (imodels.Inbox, error)
}

type settingsStore interface {
	GetAppRootURL() (string, error)
}

type csatStore interface {
	Create(conversationID int) (csatModels.CSATResponse, error)
	MakePublicURL(appBaseURL, uuid string) string
}

type webhookStore interface {
	TriggerEvent(event wmodels.WebhookEvent, data any)
}

// Opts holds the options for creating a new Manager.
type Opts struct {
	DB                       *sqlx.DB
	Lo                       *logf.Logger
	OutgoingMessageQueueSize int
	IncomingMessageQueueSize int
}

// New initializes a new conversation Manager.
func New(
	wsHub *ws.Hub,
	i18n *i18n.I18n,
	notifier *notifier.Service,
	slaStore slaStore,
	statusStore statusStore,
	priorityStore priorityStore,
	inboxStore inboxStore,
	userStore userStore,
	teamStore teamStore,
	mediaStore mediaStore,
	settingsStore settingsStore,
	csatStore csatStore,
	automation *automation.Engine,
	template *template.Manager,
	webhook webhookStore,
	opts Opts) (*Manager, error) {

	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	c := &Manager{
		q:                          q,
		wsHub:                      wsHub,
		i18n:                       i18n,
		notifier:                   notifier,
		inboxStore:                 inboxStore,
		userStore:                  userStore,
		teamStore:                  teamStore,
		mediaStore:                 mediaStore,
		settingsStore:              settingsStore,
		csatStore:                  csatStore,
		webhookStore:               webhook,
		slaStore:                   slaStore,
		statusStore:                statusStore,
		priorityStore:              priorityStore,
		automation:                 automation,
		template:                   template,
		db:                         opts.DB,
		lo:                         opts.Lo,
		incomingMessageQueue:       make(chan models.IncomingMessage, opts.IncomingMessageQueueSize),
		outgoingMessageQueue:       make(chan models.Message, opts.OutgoingMessageQueueSize),
		outgoingProcessingMessages: sync.Map{},
	}

	return c, nil
}

type queries struct {
	// Conversation queries.
	GetConversationUUID                *sqlx.Stmt `query:"get-conversation-uuid"`
	GetConversation                    *sqlx.Stmt `query:"get-conversation"`
	GetConversationsCreatedAfter       *sqlx.Stmt `query:"get-conversations-created-after"`
	GetUnassignedConversations         *sqlx.Stmt `query:"get-unassigned-conversations"`
	GetConversations                   string     `query:"get-conversations"`
	GetContactConversations            *sqlx.Stmt `query:"get-contact-conversations"`
	GetConversationParticipants        *sqlx.Stmt `query:"get-conversation-participants"`
	GetUserActiveConversationsCount    *sqlx.Stmt `query:"get-user-active-conversations-count"`
	UpdateConversationFirstReplyAt     *sqlx.Stmt `query:"update-conversation-first-reply-at"`
	UpdateConversationLastReplyAt      *sqlx.Stmt `query:"update-conversation-last-reply-at"`
	UpdateConversationWaitingSince     *sqlx.Stmt `query:"update-conversation-waiting-since"`
	UpdateConversationAssigneeLastSeen *sqlx.Stmt `query:"update-conversation-assignee-last-seen"`
	UpdateConversationAssignedUser     *sqlx.Stmt `query:"update-conversation-assigned-user"`
	UpdateConversationAssignedTeam     *sqlx.Stmt `query:"update-conversation-assigned-team"`
	UpdateConversationCustomAttributes *sqlx.Stmt `query:"update-conversation-custom-attributes"`
	UpdateConversationPriority         *sqlx.Stmt `query:"update-conversation-priority"`
	UpdateConversationStatus           *sqlx.Stmt `query:"update-conversation-status"`
	UpdateConversationLastMessage      *sqlx.Stmt `query:"update-conversation-last-message"`
	InsertConversationParticipant      *sqlx.Stmt `query:"insert-conversation-participant"`
	InsertConversation                 *sqlx.Stmt `query:"insert-conversation"`
	AddConversationTags                *sqlx.Stmt `query:"add-conversation-tags"`
	SetConversationTags                *sqlx.Stmt `query:"set-conversation-tags"`
	RemoveConversationTags             *sqlx.Stmt `query:"remove-conversation-tags"`
	GetConversationTags                *sqlx.Stmt `query:"get-conversation-tags"`
	UnassignOpenConversations          *sqlx.Stmt `query:"unassign-open-conversations"`
	ReOpenConversation                 *sqlx.Stmt `query:"re-open-conversation"`
	UnsnoozeAll                        *sqlx.Stmt `query:"unsnooze-all"`
	DeleteConversation                 *sqlx.Stmt `query:"delete-conversation"`
	RemoveConversationAssignee         *sqlx.Stmt `query:"remove-conversation-assignee"`
	GetLatestMessage                   *sqlx.Stmt `query:"get-latest-message"`

	// Message queries.
	GetMessage                         *sqlx.Stmt `query:"get-message"`
	GetMessages                        string     `query:"get-messages"`
	GetPendingMessages                 *sqlx.Stmt `query:"get-pending-messages"`
	GetMessageSourceIDs                *sqlx.Stmt `query:"get-message-source-ids"`
	GetConversationUUIDFromMessageUUID *sqlx.Stmt `query:"get-conversation-uuid-from-message-uuid"`
	InsertMessage                      *sqlx.Stmt `query:"insert-message"`
	UpdateMessageStatus                *sqlx.Stmt `query:"update-message-status"`
	MessageExistsBySourceID            *sqlx.Stmt `query:"message-exists-by-source-id"`
	GetConversationByMessageID         *sqlx.Stmt `query:"get-conversation-by-message-id"`
}

// CreateConversation creates a new conversation and returns its ID and UUID.
func (c *Manager) CreateConversation(contactID, contactChannelID, inboxID int, lastMessage string, lastMessageAt time.Time, subject string, appendRefNumToSubject bool) (int, string, error) {
	var (
		id     int
		uuid   string
		prefix string
	)
	if err := c.q.InsertConversation.QueryRow(contactID, contactChannelID, models.StatusOpen, inboxID, lastMessage, lastMessageAt, subject, prefix, appendRefNumToSubject).Scan(&id, &uuid); err != nil {
		c.lo.Error("error inserting new conversation into the DB", "error", err)
		return id, uuid, err
	}
	return id, uuid, nil
}

// GetConversation retrieves a conversation by its ID or UUID.
func (c *Manager) GetConversation(id int, uuid string) (models.Conversation, error) {
	var conversation models.Conversation
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	if err := c.q.GetConversation.Get(&conversation, id, uuidParam); err != nil {
		if err == sql.ErrNoRows {
			return conversation, envelope.NewError(envelope.InputError,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.conversation}"), nil)
		}
		c.lo.Error("error fetching conversation", "error", err)
		return conversation, envelope.NewError(envelope.GeneralError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}

	// Strip name and extract plain email from "Name <email>"
	var err error
	conversation.InboxMail, err = stringutil.ExtractEmail(conversation.InboxMail)
	if err != nil {
		c.lo.Error("error extracting email from inbox mail", "inbox_mail", conversation.InboxMail, "error", err)
	}

	return conversation, nil
}

// GetContactConversations retrieves conversations for a contact.
func (c *Manager) GetContactConversations(contactID int) ([]models.Conversation, error) {
	var conversations = make([]models.Conversation, 0)
	if err := c.q.GetContactConversations.Select(&conversations, contactID); err != nil {
		c.lo.Error("error fetching conversations", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}
	return conversations, nil
}

// GetConversationsCreatedAfter retrieves conversations created after the specified time.
func (c *Manager) GetConversationsCreatedAfter(time time.Time) ([]models.Conversation, error) {
	var conversations = make([]models.Conversation, 0)
	if err := c.q.GetConversationsCreatedAfter.Select(&conversations, time); err != nil {
		c.lo.Error("error fetching conversation", "error", err)
		return conversations, err
	}
	return conversations, nil
}

// UpdateConversationAssigneeLastSeen updates the last seen timestamp of assignee.
func (c *Manager) UpdateConversationAssigneeLastSeen(uuid string) error {
	if _, err := c.q.UpdateConversationAssigneeLastSeen.Exec(uuid); err != nil {
		c.lo.Error("error updating conversation", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	// Broadcast the property update to all subscribers.
	c.BroadcastConversationUpdate(uuid, "assignee_last_seen_at", time.Now().Format(time.RFC3339))
	return nil
}

// GetConversationParticipants retrieves the participants of a conversation.
func (c *Manager) GetConversationParticipants(uuid string) ([]models.ConversationParticipant, error) {
	conv := make([]models.ConversationParticipant, 0)
	if err := c.q.GetConversationParticipants.Select(&conv, uuid); err != nil {
		c.lo.Error("error fetching conversation", "error", err)
		return conv, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}
	return conv, nil
}

// GetUnassignedConversations retrieves unassigned conversations.
func (c *Manager) GetUnassignedConversations() ([]models.Conversation, error) {
	var conv []models.Conversation
	if err := c.q.GetUnassignedConversations.Select(&conv); err != nil {
		if err != sql.ErrNoRows {
			c.lo.Error("error fetching conversations", "error", err)
			return conv, err
		}
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

// GetAllConversationsList retrieves all conversations with optional filtering, ordering, and pagination.
func (c *Manager) GetAllConversationsList(order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(0, []int{}, []string{models.AllConversations}, order, orderBy, filters, page, pageSize)
}

// GetAssignedConversationsList retrieves conversations assigned to a specific user with optional filtering, ordering, and pagination.
func (c *Manager) GetAssignedConversationsList(userID int, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(userID, []int{}, []string{models.AssignedConversations}, order, orderBy, filters, page, pageSize)
}

// GetUnassignedConversationsList retrieves conversations assigned to a team the user is part of with optional filtering, ordering, and pagination.
func (c *Manager) GetUnassignedConversationsList(order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(0, []int{}, []string{models.UnassignedConversations}, order, orderBy, filters, page, pageSize)
}

// GetTeamUnassignedConversationsList retrieves conversations assigned to a team with optional filtering, ordering, and pagination.
func (c *Manager) GetTeamUnassignedConversationsList(teamID int, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(0, []int{teamID}, []string{models.TeamUnassignedConversations}, order, orderBy, filters, page, pageSize)
}

func (c *Manager) GetViewConversationsList(userID int, teamIDs []int, listType []string, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(userID, teamIDs, listType, order, orderBy, filters, page, pageSize)
}

// GetConversations retrieves conversations list based on user ID, type, and optional filtering, ordering, and pagination.
func (c *Manager) GetConversations(userID int, teamIDs []int, listTypes []string, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	var conversations = make([]models.Conversation, 0)

	// Make the query.
	query, qArgs, err := c.makeConversationsListQuery(userID, teamIDs, listTypes, c.q.GetConversations, order, orderBy, page, pageSize, filters)
	if err != nil {
		c.lo.Error("error making conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}

	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	defer tx.Rollback()
	if err != nil {
		c.lo.Error("error preparing get conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}

	if err := tx.Select(&conversations, query, qArgs...); err != nil {
		c.lo.Error("error fetching conversations", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}
	return conversations, nil
}

// ReOpenConversation reopens a conversation if it's snoozed, resolved or closed.
func (c *Manager) ReOpenConversation(conversationUUID string, actor umodels.User) error {
	rows, err := c.q.ReOpenConversation.Exec(conversationUUID)
	if err != nil {
		c.lo.Error("error reopening conversation", "uuid", conversationUUID, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	// Record the status change as an activity if the conversation was reopened.
	count, _ := rows.RowsAffected()
	if count > 0 {
		// Broadcast update using WS
		c.BroadcastConversationUpdate(conversationUUID, "status", models.StatusOpen)

		// Record the status change as an activity.
		if err := c.RecordStatusChange(models.StatusOpen, conversationUUID, actor); err != nil {
			return err
		}
	}
	return nil
}

// ActiveUserConversationsCount returns the count of active conversations for a user. i.e. conversations not closed or resolved status.
func (c *Manager) ActiveUserConversationsCount(userID int) (int, error) {
	var count int
	if err := c.q.GetUserActiveConversationsCount.Get(&count, userID); err != nil {
		c.lo.Error("error fetching active conversation count", "error", err)
		return count, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}
	return count, nil
}

// UpdateConversationLastMessage updates the last message details for a conversation.
func (c *Manager) UpdateConversationLastMessage(conversation int, conversationUUID, lastMessage, lastMessageSenderType string, lastMessageAt time.Time) error {
	if _, err := c.q.UpdateConversationLastMessage.Exec(conversation, conversationUUID, lastMessage, lastMessageSenderType, lastMessageAt); err != nil {
		c.lo.Error("error updating conversation last message", "error", err)
		return err
	}
	return nil
}

// UpdateConversationFirstReplyAt updates the first reply timestamp for a conversation.
func (c *Manager) UpdateConversationFirstReplyAt(conversationUUID string, conversationID int, at time.Time) error {
	res, err := c.q.UpdateConversationFirstReplyAt.Exec(conversationID, at)
	if err != nil {
		c.lo.Error("error updating conversation first reply at", "error", err)
		return err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		c.BroadcastConversationUpdate(conversationUUID, "first_reply_at", at.Format(time.RFC3339))
	}
	return nil
}

// UpdateConversationLastReplyAt updates the last reply timestamp for a conversation.
func (c *Manager) UpdateConversationLastReplyAt(conversationUUID string, conversationID int, at time.Time) error {
	res, err := c.q.UpdateConversationLastReplyAt.Exec(conversationID, at)
	if err != nil {
		c.lo.Error("error updating conversation last reply at", "error", err)
		return err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		c.BroadcastConversationUpdate(conversationUUID, "last_reply_at", at.Format(time.RFC3339))
	}
	return nil
}

// UpdateConversationWaitingSince updates the waiting since timestamp for a conversation.
func (c *Manager) UpdateConversationWaitingSince(conversationUUID string, at *time.Time) error {
	res, err := c.q.UpdateConversationWaitingSince.Exec(conversationUUID, at)
	if err != nil {
		c.lo.Error("error updating conversation waiting since", "error", err)
		return err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		if at != nil {
			c.BroadcastConversationUpdate(conversationUUID, "waiting_since", at.Format(time.RFC3339))
		} else {
			c.BroadcastConversationUpdate(conversationUUID, "waiting_since", nil)
		}
	}
	return nil
}

// UpdateConversationUserAssignee sets the assignee of a conversation to a specifc user.
func (c *Manager) UpdateConversationUserAssignee(uuid string, assigneeID int, actor umodels.User) error {
	if err := c.UpdateAssignee(uuid, assigneeID, models.AssigneeTypeUser); err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	c.webhookStore.TriggerEvent(wmodels.EventConversationAssigned, map[string]any{
		"conversation_uuid": uuid,
		"assigned_to":       assigneeID,
		"actor_id":          actor.ID,
	})

	// Refetch the conversation to get the updated details.
	conversation, err := c.GetConversation(0, uuid)
	if err != nil {
		return err
	}

	// Evaluate automation rules.
	c.automation.EvaluateConversationUpdateRules(conversation, amodels.EventConversationUserAssigned)

	// Send email to assignee.
	if err := c.SendAssignedConversationEmail([]int{assigneeID}, conversation); err != nil {
		c.lo.Error("error sending assigned conversation email", "error", err)
	}

	if err := c.RecordAssigneeUserChange(uuid, assigneeID, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	return nil
}

// UpdateConversationTeamAssignee sets the assignee of a conversation to a specific team and sets the assigned user id to NULL.
func (c *Manager) UpdateConversationTeamAssignee(uuid string, teamID int, actor umodels.User) error {
	// Store previously assigned team ID to apply SLA policy if team has changed.
	conversation, err := c.GetConversation(0, uuid)
	if err != nil {
		return err
	}
	previousAssignedTeamID := conversation.AssignedTeamID.Int

	if err := c.UpdateAssignee(uuid, teamID, models.AssigneeTypeTeam); err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	// Assignment successful, any errors now are non-critical and can be ignored by returning nil.
	if err := c.RecordAssigneeTeamChange(uuid, teamID, actor); err != nil {
		return nil
	}

	// Team changed?
	if previousAssignedTeamID != teamID {
		team, err := c.teamStore.Get(teamID)
		if err != nil {
			return nil
		}
		// Fetch the conversation again to get the updated details.
		conversation, err := c.GetConversation(0, uuid)
		if err != nil {
			return nil
		}
		if team.SLAPolicyID.Int > 0 {
			systemUser, err := c.userStore.GetSystemUser()
			if err != nil {
				return nil
			}
			if err := c.ApplySLA(conversation, team.SLAPolicyID.Int, systemUser); err != nil {
				return nil
			}
		}

		// Evaluate automation rules for conversation team assignment.
		c.automation.EvaluateConversationUpdateRules(conversation, amodels.EventConversationTeamAssigned)
	}
	return nil
}

// UpdateAssignee updates the assignee of a conversation.
func (c *Manager) UpdateAssignee(uuid string, assigneeID int, assigneeType string) error {
	var prop string
	switch assigneeType {
	case models.AssigneeTypeUser:
		prop = "assigned_user_id"
		if _, err := c.q.UpdateConversationAssignedUser.Exec(uuid, assigneeID); err != nil {
			c.lo.Error("error updating conversation assignee", "error", err)
			return fmt.Errorf("updating assignee: %w", err)
		}
	case models.AssigneeTypeTeam:
		prop = "assigned_team_id"
		if _, err := c.q.UpdateConversationAssignedTeam.Exec(uuid, assigneeID); err != nil {
			c.lo.Error("error updating conversation assignee", "error", err)
			return fmt.Errorf("updating assignee: %w", err)
		}
	default:
		return fmt.Errorf("invalid assignee type: %s", assigneeType)
	}
	// Broadcast update to all subscribers.
	c.BroadcastConversationUpdate(uuid, prop, assigneeID)
	return nil
}

// UpdateConversationPriority updates the priority of a conversation.
func (c *Manager) UpdateConversationPriority(uuid string, priorityID int, priority string, actor umodels.User) error {
	// Fetch the priority name if priority ID is provided.
	if priorityID > 0 {
		p, err := c.priorityStore.Get(priorityID)
		if err != nil {
			return envelope.NewError(envelope.InputError, err.Error(), nil)
		}
		priority = p.Name
	}
	if _, err := c.q.UpdateConversationPriority.Exec(uuid, priority); err != nil {
		c.lo.Error("error updating conversation priority", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	// Evaluate automation rules for conversation priority change.
	conversation, err := c.GetConversation(0, uuid)
	if err == nil {
		c.automation.EvaluateConversationUpdateRules(conversation, amodels.EventConversationPriorityChange)
	}

	// Record activity.
	if err := c.RecordPriorityChange(priority, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}
	c.BroadcastConversationUpdate(uuid, "priority", priority)
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
		return envelope.NewError(envelope.InputError, c.i18n.T("conversation.invalidSnoozeDuration"), nil)
	}

	// Parse the snooze duration if status is snoozed.
	snoozeUntil := time.Time{}
	if status == models.StatusSnoozed {
		duration, err := time.ParseDuration(snoozeDur)
		if err != nil {
			c.lo.Error("error parsing snooze duration", "error", err)
			return envelope.NewError(envelope.InputError, c.i18n.T("conversation.invalidSnoozeDuration"), nil)
		}
		snoozeUntil = time.Now().Add(duration)
	}

	conversationBeforeChange, err := c.GetConversation(0, uuid)
	if err != nil {
		c.lo.Error("error fetching conversation before status change", "uuid", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
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
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
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
	})

	// Record the status change as an activity.
	if err := c.RecordStatusChange(status, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}

	// Broadcast updates using websocket.
	c.BroadcastConversationUpdate(uuid, "status", status)

	// Evaluate automation rules.
	conversation, err := c.GetConversation(0, uuid)
	if err != nil {
		c.lo.Error("error fetching conversation after status change", "uuid", uuid, "error", err)
	} else {
		c.automation.EvaluateConversationUpdateRules(conversation, amodels.EventConversationStatusChange)
	}

	// Broadcast `resolved_at` if the status is changed to resolved, `resolved_at` is set only once when the conversation is resolved for the first time.
	// Subsequent status changes to resolved will not update the `resolved_at` field.
	if oldStatus != models.StatusResolved && status == models.StatusResolved {
		resolvedAt := conversationBeforeChange.ResolvedAt.Time
		if resolvedAt.IsZero() {
			resolvedAt = time.Now()
		}
		c.BroadcastConversationUpdate(uuid, "resolved_at", resolvedAt.Format(time.RFC3339))
	}
	return nil
}

// SetConversationTags sets the tags associated with a conversation.
func (c *Manager) SetConversationTags(uuid string, action string, tagNames []string, actor umodels.User) error {
	// Get current tags list.
	prevTags, err := c.getConversationTags(uuid)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.tag}"), nil)
	}
	if prevTags == nil {
		prevTags = []string{}
	}

	// Add specified tags, ignore existing ones.
	if action == amodels.ActionAddTags {
		if _, err := c.q.AddConversationTags.Exec(uuid, pq.Array(tagNames)); err != nil {
			c.lo.Error("error adding conversation tags", "error", err)
			return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.tag}"), nil)
		}
	}

	// Set specified tags and remove all other existing ones.
	if action == amodels.ActionSetTags {
		if _, err := c.q.SetConversationTags.Exec(uuid, pq.Array(tagNames)); err != nil {
			c.lo.Error("error setting conversation tags", "error", err)
			return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.tag}"), nil)
		}
	}

	// Delete specified tags, ignore all others.
	if action == amodels.ActionRemoveTags {
		if _, err := c.q.RemoveConversationTags.Exec(uuid, pq.Array(tagNames)); err != nil {
			c.lo.Error("error removing conversation tags", "error", err)
			return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.tag}"), nil)
		}
	}

	// Get updated tags list.
	newTags, err := c.getConversationTags(uuid)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.tag}"), nil)
	}

	// Trigger webhook for conversation tags changed.
	if newTags == nil {
		newTags = []string{}
	}
	c.webhookStore.TriggerEvent(wmodels.EventConversationTagsChanged, map[string]any{
		"conversation_uuid": uuid,
		"previous_tags":     prevTags,
		"new_tags":          newTags,
		"actor_id":          actor.ID,
	})

	// Find actually removed tags.
	for _, tag := range prevTags {
		if slices.Contains(newTags, tag) {
			continue
		}
		// Record the removed tags as activities.
		if err := c.RecordTagRemoval(uuid, tag, actor); err != nil {
			return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.tag}"), nil)
		}
	}

	// Find actually added tags.
	for _, tag := range newTags {
		if slices.Contains(prevTags, tag) {
			continue
		}
		// Record the added tags as activities.
		if err := c.RecordTagAddition(uuid, tag, actor); err != nil {
			return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.tag}"), nil)
		}
	}

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

// SendAssignedConversationEmail sends a email for an assigned conversation to the passed user ids.
func (m *Manager) SendAssignedConversationEmail(userIDs []int, conversation models.Conversation) error {
	agent, err := m.userStore.GetAgent(userIDs[0], "")
	if err != nil {
		m.lo.Error("error fetching agent", "user_id", userIDs[0], "error", err)
		return fmt.Errorf("fetching agent: %w", err)
	}

	content, subject, err := m.template.RenderStoredEmailTemplate(template.TmplConversationAssigned,
		map[string]any{
			"Conversation": map[string]any{
				"ReferenceNumber": conversation.ReferenceNumber,
				"Subject":         conversation.Subject.String,
				"Priority":        conversation.Priority.String,
				"UUID":            conversation.UUID,
			},
			"Contact": map[string]any{
				"FirstName": conversation.Contact.FirstName,
				"LastName":  conversation.Contact.LastName,
				"FullName":  conversation.Contact.FullName(),
				"Email":     conversation.Contact.Email.String,
			},
			"Recipient": map[string]any{
				"FirstName": agent.FirstName,
				"LastName":  agent.LastName,
				"FullName":  agent.FullName(),
				"Email":     agent.Email.String,
			},
			// Automated messages do not have an author.
			"Author": map[string]any{
				"FirstName": "",
				"LastName":  "",
				"FullName":  "",
				"Email":     "",
			},
		})
	if err != nil {
		m.lo.Error("error rendering template", "template", template.TmplConversationAssigned, "conversation_uuid", conversation.UUID, "error", err)
		return fmt.Errorf("rendering template: %w", err)
	}
	nm := notifier.Message{
		RecipientEmails: []string{agent.Email.String},
		Subject:         subject,
		Content:         content,
		Provider:        notifier.ProviderEmail,
	}
	if err := m.notifier.Send(nm); err != nil {
		m.lo.Error("error sending notification message", "template", template.TmplConversationAssigned, "conversation_uuid", conversation.UUID, "error", err)
		return fmt.Errorf("sending notification message with template %s: %w", template.TmplConversationAssigned, err)
	}
	return nil
}

// UnassignOpen unassigns all open conversations belonging to a user.
// i.e conversations without status `Closed` and `Resolved`.
func (m *Manager) UnassignOpen(userID int) error {
	if _, err := m.q.UnassignOpenConversations.Exec(userID); err != nil {
		m.lo.Error("error unassigning open conversations", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("conversation.errorUnassigningOpenConversations"), nil)
	}
	return nil
}

// ApplySLA applies the SLA policy to a conversation.
func (m *Manager) ApplySLA(conversation models.Conversation, policyID int, actor umodels.User) error {
	policy, err := m.slaStore.ApplySLA(conversation.CreatedAt, conversation.ID, conversation.AssignedTeamID.Int, policyID)
	if err != nil {
		m.lo.Error("error applying SLA to conversation", "conversation_id", conversation.ID, "policy_id", policyID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorApplying", "name", m.i18n.Ts("globals.terms.sla")), nil)
	}

	// Record the SLA application as an activity.
	if err := m.RecordSLASet(conversation.UUID, policy.Name, actor); err != nil {
		return err
	}
	return nil
}

// ApplyAction applies an action to a conversation, this can be called from multiple packages across the app to perform actions on conversations.
// all actions are executed on behalf of the provided user if the user is not provided, system user is used.
func (m *Manager) ApplyAction(action amodels.RuleAction, conv models.Conversation, user umodels.User) error {
	// CSAT action does not require a value.
	if len(action.Value) == 0 && action.Type != amodels.ActionSendCSAT {
		return fmt.Errorf("empty value for action %s", action.Type)
	}

	// Fall back to system user if user is not provided.
	if user.ID == 0 {
		var err error
		if user, err = m.userStore.GetSystemUser(); err != nil {
			return fmt.Errorf("get system user: %w", err)
		}
	}

	m.lo.Debug("executing action",
		"type", action.Type,
		"value", action.Value,
		"conv_uuid", conv.UUID,
		"user_id", user.ID,
	)

	switch action.Type {
	case amodels.ActionAssignTeam:
		teamID, err := strconv.Atoi(action.Value[0])
		if err != nil {
			return fmt.Errorf("invalid team ID %q: %w", action.Value[0], err)
		}
		return m.UpdateConversationTeamAssignee(conv.UUID, teamID, user)
	case amodels.ActionAssignUser:
		agentID, err := strconv.Atoi(action.Value[0])
		if err != nil {
			return fmt.Errorf("invalid agent ID %q: %w", action.Value[0], err)
		}
		return m.UpdateConversationUserAssignee(conv.UUID, agentID, user)
	case amodels.ActionSetPriority:
		priorityID, err := strconv.Atoi(action.Value[0])
		if err != nil {
			return fmt.Errorf("invalid priority ID %q: %w", action.Value[0], err)
		}
		return m.UpdateConversationPriority(conv.UUID, priorityID, "", user)
	case amodels.ActionSetStatus:
		statusID, err := strconv.Atoi(action.Value[0])
		if err != nil {
			return fmt.Errorf("invalid status ID %q: %w", action.Value[0], err)
		}
		return m.UpdateConversationStatus(conv.UUID, statusID, "", "", user)
	case amodels.ActionSendPrivateNote:
		return m.SendPrivateNote([]mmodels.Media{}, user.ID, conv.UUID, action.Value[0])
	case amodels.ActionReply:
		// Make recipient list.
		to, cc, bcc, err := m.makeRecipients(conv.ID, conv.Contact.Email.String, conv.InboxMail)
		if err != nil {
			return fmt.Errorf("making recipients for reply action: %w", err)
		}
		return m.SendReply(
			[]mmodels.Media{},
			conv.InboxID,
			user.ID,
			conv.UUID,
			action.Value[0],
			to,
			cc,
			bcc,
			map[string]any{}, /**meta**/
		)
	case amodels.ActionSetSLA:
		slaID, err := strconv.Atoi(action.Value[0])
		if err != nil {
			return fmt.Errorf("invalid SLA ID %q: %w", action.Value[0], err)
		}
		return m.ApplySLA(conv, slaID, user)
	case amodels.ActionAddTags, amodels.ActionSetTags, amodels.ActionRemoveTags:
		return m.SetConversationTags(conv.UUID, action.Type, action.Value, user)
	case amodels.ActionSendCSAT:
		return m.SendCSATReply(user.ID, conv)
	default:
		return fmt.Errorf("unknown action: %s", action.Type)
	}
}

// RemoveConversationAssignee removes the assignee from the conversation.
func (m *Manager) RemoveConversationAssignee(uuid, typ string, actor umodels.User) error {
	if _, err := m.q.RemoveConversationAssignee.Exec(uuid, typ); err != nil {
		m.lo.Error("error removing conversation assignee", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("conversation.errorRemovingConversationAssignee"), nil)
	}

	// Trigger webhook for conversation unassigned from user.
	if typ == models.AssigneeTypeUser {
		m.webhookStore.TriggerEvent(wmodels.EventConversationUnassigned, map[string]any{
			"conversation_uuid": uuid,
			"actor_id":          actor.ID,
		})
	}

	return nil
}

// SendCSATReply sends a CSAT reply message to a conversation.
func (m *Manager) SendCSATReply(actorUserID int, conversation models.Conversation) error {
	appRootURL, err := m.settingsStore.GetAppRootURL()
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.appRootURL}"), nil)
	}
	csat, err := m.csatStore.Create(conversation.ID)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.csat}"), nil)
	}
	csatPublicURL := m.csatStore.MakePublicURL(appRootURL, csat.UUID)
	message := fmt.Sprintf(csatReplyMessage, csatPublicURL)
	// Store `is_csat` meta to identify and filter CSAT public url from the message.
	meta := map[string]interface{}{
		"is_csat": true,
	}

	// Make recipient list.
	to, cc, bcc, err := m.makeRecipients(conversation.ID, conversation.Contact.Email.String, conversation.InboxMail)
	if err != nil {
		return fmt.Errorf("making recipients for CSAT reply: %w", err)
	}

	return m.SendReply(nil /**media**/, conversation.InboxID, actorUserID, conversation.UUID, message, to, cc, bcc, meta)
}

// DeleteConversation deletes a conversation.
func (m *Manager) DeleteConversation(uuid string) error {
	if _, err := m.q.DeleteConversation.Exec(uuid); err != nil {
		m.lo.Error("error deleting conversation", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", m.i18n.Ts("globals.terms.conversation")), nil)
	}
	return nil
}

// UpdateConversationCustomAttributes updates the custom attributes of a conversation.
func (c *Manager) UpdateConversationCustomAttributes(uuid string, customAttributes map[string]any) error {
	jsonb, err := json.Marshal(customAttributes)
	if err != nil {
		c.lo.Error("error marshalling custom attributes", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}
	if _, err := c.q.UpdateConversationCustomAttributes.Exec(uuid, jsonb); err != nil {
		c.lo.Error("error updating conversation custom attributes", "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.conversation}"), nil)
	}
	// Broadcast the custom attributes update.
	c.BroadcastConversationUpdate(uuid, "custom_attributes", customAttributes)
	return nil
}

// addConversationParticipant adds a user as participant to a conversation.
func (c *Manager) addConversationParticipant(userID int, conversationUUID string) error {
	if _, err := c.q.InsertConversationParticipant.Exec(userID, conversationUUID); err != nil && !dbutil.IsUniqueViolationError(err) {
		c.lo.Error("error adding conversation participant", "user_id", userID, "conversation_uuid", conversationUUID, "error", err)
		return envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.conversationParticipant}"), nil)
	}
	return nil
}

// getConversationTags retrieves the tags associated with a conversation.
func (c *Manager) getConversationTags(uuid string) ([]string, error) {
	var tags []string
	if err := c.q.GetConversationTags.Select(&tags, uuid); err != nil {
		c.lo.Error("error fetching conversation tags", "error", err)
		return tags, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.tag}"), nil)
	}
	return tags, nil
}

// makeConversationsListQuery prepares a SQL query string for conversations list
func (c *Manager) makeConversationsListQuery(userID int, teamIDs []int, listTypes []string, baseQuery, order, orderBy string, page, pageSize int, filtersJSON string) (string, []interface{}, error) {
	var qArgs []interface{}

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
	if pageSize > conversationsListMaxPageSize || pageSize < 1 {
		return "", nil, fmt.Errorf("invalid page size: must be between 1 and %d", conversationsListMaxPageSize)
	}
	if page < 1 {
		return "", nil, fmt.Errorf("page must be greater than 0")
	}

	if len(listTypes) == 0 {
		return "", nil, fmt.Errorf("no conversation list types specified")
	}

	// Prepare the conditions based on the list types.
	conditions := []string{}
	for _, lt := range listTypes {
		switch lt {
		case models.AssignedConversations:
			conditions = append(conditions, fmt.Sprintf("conversations.assigned_user_id = $%d", len(qArgs)+1))
			qArgs = append(qArgs, userID)
		case models.UnassignedConversations:
			conditions = append(conditions, "conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL")
		case models.TeamUnassignedConversations:
			placeholders := make([]string, len(teamIDs))
			for i := range teamIDs {
				placeholders[i] = fmt.Sprintf("$%d", len(qArgs)+i+1)
			}
			conditions = append(conditions, fmt.Sprintf("(conversations.assigned_team_id IN (%s) AND conversations.assigned_user_id IS NULL)", strings.Join(placeholders, ",")))
			for _, id := range teamIDs {
				qArgs = append(qArgs, id)
			}
		case models.AllConversations:
			// No conditions needed for all conversations.
		default:
			return "", nil, fmt.Errorf("unknown conversation type: %s", lt)
		}
	}

	if len(conditions) > 0 {
		baseQuery = fmt.Sprintf(baseQuery, "AND ("+strings.Join(conditions, " OR ")+")")
	} else {
		// Replace the `%s` in the base query with an empty string.
		baseQuery = fmt.Sprintf(baseQuery, "")
	}

	return dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
	}, filtersJSON, dbutil.AllowedFields{
		"conversations":         conversationsAllowedFields,
		"conversation_statuses": conversationStatusAllowedFields,
	})
}
