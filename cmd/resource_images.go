package main

import (
	"context"
	"encoding/json"
	"net/mail"
	"slices"
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/resourceimage"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/fastglue"
)

func imageSourceID(source string) string {
	return resourceimage.SourceID(source)
}

func imageSender(msg cmodels.Message) string {
	if msg.SenderType != cmodels.SenderTypeContact || msg.Type != cmodels.MessageIncoming {
		return ""
	}
	var meta struct {
		From []string `json:"from"`
	}
	if err := json.Unmarshal(msg.Meta, &meta); err != nil || len(meta.From) != 1 {
		return ""
	}
	address, err := mail.ParseAddress(meta.From[0])
	if err != nil || len(address.Address) > 254 {
		return ""
	}
	parts := strings.SplitN(address.Address, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0] + "@" + strings.ToLower(parts[1])
}

func imagePermission(app *App, userID int, msg cmodels.Message) (bool, bool) {
	return imagePermissionTx(app, nil, userID, msg)
}

func imagePermissionTx(app *App, tx *sqlx.Tx, userID int, msg cmodels.Message) (bool, bool) {
	trusted := false
	if sender := imageSender(msg); sender != "" {
		senders, err := app.user.GetImageSendersTx(tx, userID)
		trusted = err == nil && slices.Contains(senders, sender)
	}
	if trusted {
		return true, true
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	allowed, err := app.resourceImages.MessageAllowedTx(ctx, tx, userID, msg.ID, imageSourceID(msg.Content))
	return err == nil && allowed, false
}

func authorizedImageMessage(r *fastglue.Request) (cmodels.Message, int, error) {
	app := r.Context.(*App)
	userID := r.RequestCtx.UserValue("user").(amodels.User).ID
	user, err := app.user.GetAgentCachedOrLoad(userID)
	if err != nil {
		return cmodels.Message{}, userID, err
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	if _, err := enforceConversationAccess(app, cuuid, user); err != nil {
		return cmodels.Message{}, userID, err
	}
	msg, err := app.conversation.GetMessage(r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return msg, userID, err
	}
	if msg.ConversationUUID != cuuid || msg.ContentType != cmodels.ContentTypeHTML {
		return msg, userID, envelope.NewError(envelope.PermissionError, "Permission denied", nil)
	}
	return msg, userID, nil
}

func messageImageSources(msg cmodels.Message) map[string]string {
	return resourceimage.Sources(msg.Content)
}

func handleResourceImage(r *fastglue.Request) error {
	setResourceImageHeaders(r)
	app := r.Context.(*App)
	msg, userID, err := authorizedImageMessage(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	policy := loadResourcePolicy(app)
	sourceID := r.RequestCtx.UserValue("image").(string)
	source := messageImageSources(msg)[sourceID]
	allowed, _ := imagePermission(app, userID, msg)
	if source == "" || !imageLoadingAllowed(policy, source, allowed) {
		return r.SendErrorEnvelope(403, "Image loading is not permitted", nil, envelope.PermissionError)
	}
	body, err := app.resourceImages.Get(r.RequestCtx, msg.ID, sourceID, source, func(tx *sqlx.Tx) error {
		policy, err := resourceImagePolicyTx(app, tx)
		if err != nil {
			return err
		}
		allowed, _ := imagePermissionTx(app, tx, userID, msg)
		if !imageLoadingAllowed(policy, source, allowed) {
			return resourceimage.ErrImage
		}
		return nil
	})
	if err != nil {
		return r.SendErrorEnvelope(502, "Image unavailable or unsupported", nil, envelope.GeneralError)
	}
	policy = loadResourcePolicy(app)
	allowed, _ = imagePermission(app, userID, msg)
	if !imageLoadingAllowed(policy, source, allowed) {
		return r.SendErrorEnvelope(403, "Image loading is not permitted", nil, envelope.PermissionError)
	}
	r.RequestCtx.SetContentType("image/png")
	r.RequestCtx.SetBody(body)
	return nil
}

func handleAllowResourceImages(r *fastglue.Request) error {
	app := r.Context.(*App)
	msg, userID, err := authorizedImageMessage(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope := r.RequestCtx.UserValue("scope").(string)
	if scope != "revoke-sender" && !loadResourcePolicy(app).PermitsConsent() {
		return r.SendErrorEnvelope(403, "External images are disabled by the administrator", nil, envelope.PermissionError)
	}
	switch scope {
	case "message":
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = app.resourceImages.AllowMessage(ctx, userID, msg.ID, imageSourceID(msg.Content))
	case "sender":
		sender := imageSender(msg)
		if sender == "" {
			return r.SendErrorEnvelope(400, "This message has no sender address to trust", nil, envelope.InputError)
		}
		err = app.user.SetImageSender(userID, sender, true)
	case "revoke-sender":
		err = app.user.SetImageSender(userID, imageSender(msg), false)
	default:
		return r.SendErrorEnvelope(400, "Invalid image permission", nil, envelope.InputError)
	}
	if err != nil {
		return r.SendErrorEnvelope(500, "Unable to save image permission", nil, envelope.GeneralError)
	}
	return handleGetMessage(r)
}

func setResourceImageHeaders(r *fastglue.Request) {
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, no-store")
	r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	r.RequestCtx.Response.Header.Set("Cross-Origin-Resource-Policy", "same-origin")
	r.RequestCtx.Response.Header.Set("Content-Security-Policy", "default-src 'none'; sandbox")

	r.RequestCtx.Response.Header.Set("Referrer-Policy", "no-referrer")
}

func handleResourceAvatar(r *fastglue.Request) error {
	setResourceImageHeaders(r)
	app := r.Context.(*App)
	source := string(r.RequestCtx.QueryArgs().Peek("source"))
	if !resourcepolicy.ValidImageURL(source) || !loadResourcePolicy(app).PermitsFetching() {
		return r.SendErrorEnvelope(403, "Avatar loading is not permitted", nil, envelope.PermissionError)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	owner, err := app.resourceImages.AvatarOwner(ctx, source)
	if err != nil {
		return r.SendErrorEnvelope(404, "Avatar unavailable", nil, envelope.NotFoundError)
	}
	body, err := app.resourceImages.GetAvatar(ctx, owner, imageSourceID(source), source, func(tx *sqlx.Tx) error {
		policy, err := resourceImagePolicyTx(app, tx)
		if err != nil {
			return err
		}
		if !policy.PermitsFetching() {
			return resourceimage.ErrImage
		}
		return nil
	})
	if err != nil {
		return r.SendErrorEnvelope(502, "Avatar unavailable or unsupported", nil, envelope.GeneralError)
	}
	if !loadResourcePolicy(app).PermitsFetching() {
		return r.SendErrorEnvelope(403, "Avatar loading is not permitted", nil, envelope.PermissionError)
	}
	r.RequestCtx.SetContentType("image/png")
	r.RequestCtx.SetBody(body)
	return nil
}

func imageLoadingAllowed(policy resourcepolicy.Policy, source string, consent bool) bool {
	return resourcepolicy.ValidImageURL(source) && (policy.AllowsURL(source) || (policy.PermitsConsent() && consent))
}

func resourceImagePolicyTx(app *App, tx *sqlx.Tx) (resourcepolicy.Policy, error) {
	cfg, err := app.setting.GetResourcePolicyTx(tx)
	if err != nil {
		return resourcepolicy.Policy{}, err
	}
	return resourcepolicy.New(cfg)
}
