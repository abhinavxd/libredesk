package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/zerodha/fastglue"
)

// handleDeleteMessage deletes a private note from a conversation. Only private
// notes can be removed (enforced by the SQL query); sent/incoming messages are
// protected. Returns 404 if the message doesn't exist or isn't private.
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

	// Ensure the agent has access to the conversation.
	if _, err = enforceConversationAccess(app, cuuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.DeletePrivateMessage(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
