package main

import (
	"encoding/json"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	medModels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type createConversationRequest struct {
	InboxID         int    `json:"inbox_id" form:"inbox_id"`
	AssignedAgentID int    `json:"agent_id" form:"agent_id"`
	AssignedTeamID  int    `json:"team_id" form:"team_id"`
	Email           string `json:"contact_email" form:"contact_email"`
	FirstName       string `json:"first_name" form:"first_name"`
	LastName        string `json:"last_name" form:"last_name"`
	Subject         string `json:"subject" form:"subject"`
	Content         string `json:"content" form:"content"`
	Attachments     []int  `json:"attachments" form:"attachments"`
}

// handleGetAllConversations retrieves all conversations.
func handleGetAllConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total       = 0
	)

	conversations, err := app.conversation.GetAllConversationsList(order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetAssignedConversations retrieves conversations assigned to the current user.
func handleGetAssignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(amodels.User)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	conversations, err := app.conversation.GetAssignedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetUnassignedConversations retrieves unassigned conversations.
func handleGetUnassignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)

	conversations, err := app.conversation.GetUnassignedConversationsList(order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetViewConversations retrieves conversations for a view.
func handleGetViewConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		auser       = r.RequestCtx.UserValue("user").(amodels.User)
		viewID, _   = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	if viewID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`view_id`"), nil, envelope.InputError)
	}

	// Check if user has access to the view.
	view, err := app.view.Get(viewID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if view.UserID != auser.ID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("conversation.viewPermissionDenied"), nil, envelope.PermissionError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Prepare lists user has access to based on user permissions, internally this prepares the SQL query.
	lists := []string{}
	for _, perm := range user.Permissions {
		if perm == authzModels.PermConversationsReadAll {
			// No further lists required as user has access to all conversations.
			lists = []string{cmodels.AllConversations}
			break
		}
		if perm == authzModels.PermConversationsReadUnassigned {
			lists = append(lists, cmodels.UnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadAssigned {
			lists = append(lists, cmodels.AssignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamInbox {
			lists = append(lists, cmodels.TeamUnassignedConversations)
		}
	}

	// No lists found, user doesn't have access to any conversations.
	if len(lists) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	conversations, err := app.conversation.GetViewConversationsList(user.ID, user.Teams.IDs(), lists, order, orderBy, string(view.Filters), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetTeamUnassignedConversations returns conversations assigned to a team but not to any user.
func handleGetTeamUnassignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		auser       = r.RequestCtx.UserValue("user").(amodels.User)
		teamIDStr   = r.RequestCtx.UserValue("id").(string)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	teamID, _ := strconv.Atoi(teamIDStr)
	if teamID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`team_id`"), nil, envelope.InputError)
	}

	// Check if user belongs to the team.
	exists, err := app.team.UserBelongsToTeam(teamID, auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if !exists {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("conversation.notMemberOfTeam"), nil))
	}

	conversations, err := app.conversation.GetTeamUnassignedConversationsList(teamID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetConversation retrieves a single conversation by it's UUID.
func handleGetConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	prev, _ := app.conversation.GetContactConversations(conv.ContactID)
	conv.PreviousConversations = filterCurrentConv(prev, conv.UUID)
	return r.SendEnvelope(conv)
}

// handleUpdateConversationAssigneeLastSeen updates the assignee's last seen timestamp for a conversation.
func handleUpdateConversationAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.conversation.UpdateConversationAssigneeLastSeen(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetConversationParticipants retrieves participants of a conversation.
func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	p, err := app.conversation.GetConversationParticipants(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

// handleUpdateUserAssignee updates the user assigned to a conversation.
func handleUpdateUserAssignee(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
		auser      = r.RequestCtx.UserValue("user").(amodels.User)
		assigneeID = r.RequestCtx.PostArgs().GetUintOrZero("assignee_id")
	)
	if assigneeID == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`assignee_id`"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.UpdateConversationUserAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateTeamAssignee updates the team assigned to a conversation.
func handleUpdateTeamAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	assigneeID, err := r.RequestCtx.PostArgs().GetUint("assignee_id")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`assignee_id`"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	_, err = app.team.Get(assigneeID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.UpdateConversationTeamAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateConversationPriority updates the priority of a conversation.
func handleUpdateConversationPriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		auser    = r.RequestCtx.UserValue("user").(amodels.User)
		priority = string(r.RequestCtx.PostArgs().Peek("priority"))
	)
	if priority == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`priority`"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.UpdateConversationPriority(uuid, 0 /**priority_id**/, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateConversationStatus updates the status of a conversation.
func handleUpdateConversationStatus(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		status       = string(r.RequestCtx.PostArgs().Peek("status"))
		snoozedUntil = string(r.RequestCtx.PostArgs().Peek("snoozed_until"))
		uuid         = r.RequestCtx.UserValue("uuid").(string)
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
	)

	// Validate inputs
	if status == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`status`"), nil, envelope.InputError)
	}
	if snoozedUntil == "" && status == cmodels.StatusSnoozed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`snoozed_until`"), nil, envelope.InputError)
	}
	if status == cmodels.StatusSnoozed {
		_, err := time.ParseDuration(snoozedUntil)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`snoozed_until`"), nil, envelope.InputError)
		}
	}

	// Enforce conversation access.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Make sure a user is assigned before resolving conversation.
	if status == cmodels.StatusResolved && conversation.AssignedUserID.Int == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("conversation.resolveWithoutAssignee"), nil))
	}

	// Update conversation status.
	if err := app.conversation.UpdateConversationStatus(uuid, 0 /**status_id**/, status, snoozedUntil, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// If status is `Resolved`, send CSAT survey if enabled on inbox.
	if status == cmodels.StatusResolved {
		// Check if CSAT is enabled on the inbox and send CSAT survey message.
		inbox, err := app.inbox.GetDBRecord(conversation.InboxID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if inbox.CSATEnabled {
			if err := app.conversation.SendCSATReply(user.ID, *conversation); err != nil {
				return sendErrorEnvelope(r, err)
			}
		}
	}
	return r.SendEnvelope(true)
}

// handleUpdateConversationtags updates conversation tags.
func handleUpdateConversationtags(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		tagNames = []string{}
		tagJSON  = r.RequestCtx.PostArgs().Peek("tags")
		auser    = r.RequestCtx.UserValue("user").(amodels.User)
		uuid     = r.RequestCtx.UserValue("uuid").(string)
	)

	if err := json.Unmarshal(tagJSON, &tagNames); err != nil {
		app.lo.Error("error unmarshalling tags JSON", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.SetConversationTags(uuid, models.ActionSetTags, tagNames, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateConversationCustomAttributes updates custom attributes of a conversation.
func handleUpdateConversationCustomAttributes(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		attributes = map[string]any{}
		auser      = r.RequestCtx.UserValue("user").(amodels.User)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
	)
	if err := r.Decode(&attributes, ""); err != nil {
		app.lo.Error("error unmarshalling custom attributes JSON", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Enforce conversation access.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update custom attributes.
	if err := app.conversation.UpdateConversationCustomAttributes(uuid, attributes); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateContactCustomAttributes updates custom attributes of a contact.
func handleUpdateContactCustomAttributes(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		attributes = map[string]any{}
		auser      = r.RequestCtx.UserValue("user").(amodels.User)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
	)
	if err := r.Decode(&attributes, ""); err != nil {
		app.lo.Error("error unmarshalling custom attributes JSON", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Enforce conversation access.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.user.UpdateCustomAttributes(conversation.ContactID, attributes); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Broadcast update.
	app.conversation.BroadcastConversationUpdate(conversation.UUID, "contact.custom_attributes", attributes)
	return r.SendEnvelope(true)
}

// enforceConversationAccess fetches the conversation and checks if the user has access to it.
func enforceConversationAccess(app *App, uuid string, user umodels.User) (*cmodels.Conversation, error) {
	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return nil, err
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, envelope.NewError(envelope.PermissionError, "Permission denied", nil)
	}
	return &conversation, nil
}

// handleRemoveUserAssignee removes the user assigned to a conversation.
func handleRemoveUserAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.conversation.RemoveConversationAssignee(uuid, "user", user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleRemoveTeamAssignee removes the team assigned to a conversation.
func handleRemoveTeamAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.conversation.RemoveConversationAssignee(uuid, "team", user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// filterCurrentConv removes the current conversation from the list of conversations.
func filterCurrentConv(convs []cmodels.Conversation, uuid string) []cmodels.Conversation {
	for i, c := range convs {
		if c.UUID == uuid {
			return append(convs[:i], convs[i+1:]...)
		}
	}
	return []cmodels.Conversation{}
}

// handleCreateConversation creates a new conversation and sends a message to it.
func handleCreateConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = createConversationRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding create conversation request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	to := []string{req.Email}

	// Validate required fields
	if req.InboxID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "`inbox_id`"), nil, envelope.InputError)
	}
	if req.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "`content`"), nil, envelope.InputError)
	}
	if req.Email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "`contact_email`"), nil, envelope.InputError)
	}
	if req.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "`first_name`"), nil, envelope.InputError)
	}
	if !stringutil.ValidEmail(req.Email) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`contact_email`"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if inbox exists and is enabled.
	inbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "inbox"), nil, envelope.InputError)
	}

	// Find or create contact.
	contact := umodels.User{
		Email:           null.StringFrom(req.Email),
		SourceChannelID: null.StringFrom(req.Email),
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		InboxID:         req.InboxID,
	}
	if err := app.user.CreateContact(&contact); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.contact}"), nil))
	}

	// Create conversation
	conversationID, conversationUUID, err := app.conversation.CreateConversation(
		contact.ID,
		contact.ContactChannelID,
		req.InboxID,
		"",         /** last_message **/
		time.Now(), /** last_message_at **/
		req.Subject,
		true, /** append reference number to subject **/
	)
	if err != nil {
		app.lo.Error("error creating conversation", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.conversation}"), nil))
	}

	// Prepare attachments.
	var media = make([]medModels.Media, 0, len(req.Attachments))
	for _, id := range req.Attachments {
		m, err := app.media.Get(id, "")
		if err != nil {
			app.lo.Error("error fetching media", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.media}"), nil, envelope.GeneralError)
		}
		media = append(media, m)
	}

	// Send reply to the created conversation.
	if err := app.conversation.SendReply(media, req.InboxID, auser.ID /**sender_id**/, conversationUUID, req.Content, to, nil /**cc**/, nil /**bcc**/, map[string]any{} /**meta**/); err != nil {
		// Delete the conversation if reply fails.
		if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
			app.lo.Error("error deleting conversation", "error", err)
		}
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil))
	}

	// Assign the conversation to the agent or team.
	if req.AssignedAgentID > 0 {
		app.conversation.UpdateConversationUserAssignee(conversationUUID, req.AssignedAgentID, user)
	}
	if req.AssignedTeamID > 0 {
		app.conversation.UpdateConversationTeamAssignee(conversationUUID, req.AssignedTeamID, user)
	}

	// Trigger webhook event for conversation created.
	conversation, err := app.conversation.GetConversation(conversationID, "")
	if err == nil {
		app.webhook.TriggerEvent(wmodels.EventConversationCreated, conversation)
	}

	return r.SendEnvelope(conversation)
}
