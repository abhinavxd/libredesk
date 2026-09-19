package main

import (
	"fmt"
	"mime"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/zerodha/fastglue"
)

func handleWidgetTranscript(r *fastglue.Request) error {
	app := r.Context.(*App)
	config, err := getWidgetConfig(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !config.Features.Transcript {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("status.deniedPermission"), nil))
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	_, conversation, err := getContactConversation(r, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	private := false
	messages, err := app.conversation.GetAllConversationMessages(uuid, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	filename := fmt.Sprintf("transcript-%s.txt", stringutil.SanitizeFilename(conversation.ReferenceNumber))
	r.RequestCtx.Response.Header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, no-store")
	r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	r.RequestCtx.SetContentType("text/plain; charset=utf-8")
	r.RequestCtx.SetBody(app.conversation.BuildTranscript(conversation, messages, time.Now()))
	return nil
}
