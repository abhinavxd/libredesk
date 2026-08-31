package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/zerodha/fastglue"
)

// handleDeleteMessage soft-deletes a private note in a conversation.
func handleDeleteMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if _, err = enforceConversationAccess(app, cuuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Agents may delete only their own private notes, admins may delete any.
	senderID := user.ID
	if user.HasAdminRole() {
		senderID = 0
	}

	content, err := app.conversation.DeletePrivateMessage(cuuid, uuid, senderID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]any{"content": content})
}
