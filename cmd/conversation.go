package main

import (
	"fmt"
	"mime"
	"time"

	amodels "github.com/jakedolan443/fernmail/internal/auth/models"
	cmodels "github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/zerodha/fastglue"
)

// handleGetSidebarCounts returns unread-message counts for each accessible address.
func handleGetSidebarCounts(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	counts, err := app.conversation.GetSidebarCounts(user.ID, user.Permissions, user.Teams.IDs())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(counts)
}

// handleGetConversation retrieves a single conversation by it's UUID.
func handleGetConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(conv)
}

// handleDownloadConversationTranscript sends the conversation transcript as a text file download.
func handleDownloadConversationTranscript(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	private := false
	messages, err := app.conversation.GetAllConversationMessages(uuid, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := maskActivationKeysFor(app, user, messages); err != nil {
		return sendErrorEnvelope(r, err)
	}

	transcript := app.conversation.BuildTranscript(*conversation, messages, time.Now())
	safeRef := stringutil.SanitizeFilename(conversation.ReferenceNumber)
	filename := fmt.Sprintf("transcript-%s.txt", safeRef)
	r.RequestCtx.Response.Header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	r.RequestCtx.SetContentType("text/plain; charset=utf-8")
	r.RequestCtx.SetBody(transcript)
	return nil
}

// handleUpdateConversationAssigneeLastSeen updates the current user's last seen timestamp for a conversation.
func handleUpdateConversationAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.conversation.UpdateUserLastSeen(uuid, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleMarkConversationAsUnread marks a conversation as unread for the current user.
func handleMarkConversationAsUnread(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.conversation.MarkAsUnread(uuid, auser.ID); err != nil {
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
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
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

// enforceConversationAccess fetches the conversation and checks if the user has access to it.
func enforceConversationAccess(app *App, uuid string, user umodels.User) (*cmodels.Conversation, error) {
	conversation, err := app.conversation.GetConversation(0, uuid, "")
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
