package main

import (
	"errors"
	"slices"
	"strings"

	"github.com/jakedolan443/fernmail/internal/address"
	amodels "github.com/jakedolan443/fernmail/internal/auth/models"
	authzModels "github.com/jakedolan443/fernmail/internal/authz/models"
	"github.com/jakedolan443/fernmail/internal/conversation"
	cmodels "github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	wmodels "github.com/jakedolan443/fernmail/internal/webhook/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Sending rule shared by replies and new emails: messages:write sends
// directly; otherwise reviews:submit holds the email for an Admin or Agent to
// approve; otherwise the request is refused.

type outgoingReq struct {
	AddressID   int      `json:"address_id"`
	Subject     string   `json:"subject"`
	Content     string   `json:"content"`
	To          []string `json:"to"`
	CC          []string `json:"cc"`
	BCC         []string `json:"bcc"`
	Attachments []int    `json:"attachments"`
}

type composeResp struct {
	// Sent is true when the email was queued for delivery, false when it went to review.
	Sent             bool            `json:"sent"`
	ConversationUUID string          `json:"conversation_uuid,omitempty"`
	Review           *cmodels.Review `json:"review,omitempty"`
}

type reviewDecisionReq struct {
	Note string `json:"note"`
}

func canReview(user umodels.User) bool {
	return slices.Contains(user.Permissions, authzModels.PermReviewsManage)
}

func canSubmitForReview(user umodels.User) bool {
	return slices.Contains(user.Permissions, authzModels.PermReviewsSubmit)
}

func canSendDirectly(user umodels.User) bool {
	return slices.Contains(user.Permissions, authzModels.PermMessagesWrite)
}

func loadAgent(r *fastglue.Request) (umodels.User, error) {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	return app.user.GetAgentCachedOrLoad(auser.ID)
}

// sendableAddress checks that the user can read the address and that it can
// currently send mail.
func sendableAddress(app *App, userID, addressID int) (address.Address, error) {
	entry, err := app.address.Get(addressID)
	if errors.Is(err, address.ErrNotFound) {
		return entry, envelope.NewError(envelope.InputError, "Choose an address to send from.", nil)
	}
	if err != nil {
		return entry, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := requireAddressAccess(app, userID, addressID); err != nil {
		return entry, err
	}
	if !entry.Enabled {
		return entry, envelope.NewError(envelope.InputError, app.i18n.T("address.disabled"), nil)
	}
	inbox, err := app.inbox.GetDBRecord(entry.InboxID)
	if err != nil {
		return entry, err
	}
	if !inbox.Enabled {
		return entry, envelope.NewError(envelope.InputError, app.i18n.T("status.disabledInbox"), nil)
	}
	return entry, nil
}

// handleCompose starts a new email from one of the user's addresses.
func handleCompose(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req outgoingReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	if !canSendDirectly(user) && !canSubmitForReview(user) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	entry, err := sendableAddress(app, user.ID, req.AddressID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	media, err := getUnassociatedMedia(app, req.Attachments, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	in := cmodels.ReviewInput{AddressID: entry.ID, Subject: req.Subject, Content: req.Content, To: req.To, CC: req.CC, BCC: req.BCC, Media: media}

	if !canSendDirectly(user) {
		review, err := app.conversation.SubmitReview(user.ID, in)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(composeResp{Review: &review})
	}

	in, err = conversation.NormalizeOutgoing(in, true)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversationUUID, contactID, err := app.conversation.StartConversation(entry.ID, entry.InboxID, in.Subject, in.To[0])
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err := app.conversation.QueueReply(media, entry.InboxID, user.ID, contactID, conversationUUID, in.Content, in.To, in.CC, in.BCC, map[string]any{}); err != nil {
		if delErr := app.conversation.DeleteConversation(conversationUUID); delErr != nil {
			app.lo.Error("error removing conversation after failed compose", "uuid", conversationUUID, "error", delErr)
		}
		return sendErrorEnvelope(r, err)
	}
	if created, err := app.conversation.GetConversation(0, conversationUUID, ""); err == nil {
		app.webhook.TriggerEvent(wmodels.EventConversationCreated, created)
	}
	return r.SendEnvelope(composeResp{Sent: true, ConversationUUID: conversationUUID})
}

// handleSubmitReplyReview holds a Contributor's reply for review.
func handleSubmitReplyReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	cuuid := r.RequestCtx.UserValue("uuid").(string)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conv, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !conv.AddressID.Valid {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "This conversation has no address to reply from.", nil, envelope.InputError)
	}
	var req outgoingReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	entry, err := sendableAddress(app, user.ID, conv.AddressID.Int)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	media, err := getUnassociatedMedia(app, req.Attachments, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	review, err := app.conversation.SubmitReview(user.ID, cmodels.ReviewInput{
		AddressID: entry.ID, ConversationID: conv.ID, Content: req.Content, To: req.To, CC: req.CC, BCC: req.BCC, Media: media,
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(review)
}

// handleGetConversationReviews returns the review cards a thread should show.
func handleGetConversationReviews(r *fastglue.Request) error {
	app := r.Context.(*App)
	cuuid := r.RequestCtx.UserValue("uuid").(string)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canReview(user) && !canSubmitForReview(user) {
		return r.SendEnvelope([]cmodels.Review{})
	}
	conv, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	reviews, err := app.conversation.ConversationReviews(conv.ID, user, canReview(user))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	for i := range reviews {
		prepareReviewDisplay(app, &reviews[i])
	}
	return r.SendEnvelope(reviews)
}

// handleGetReviews lists a reviewer's queue, or a Contributor's own submissions.
func handleGetReviews(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canReview(user) && !canSubmitForReview(user) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	reviews, err := app.conversation.ListReviews(user, canReview(user))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(reviews)
}

func handleGetReviewCounts(r *fastglue.Request) error {
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canReview(user) && !canSubmitForReview(user) {
		return r.SendEnvelope(cmodels.ReviewCounts{})
	}
	counts, err := r.Context.(*App).conversation.CountReviews(user, canReview(user))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(counts)
}

// loadReviewFor returns the review if the user is its author or may decide it.
func loadReviewFor(app *App, user umodels.User, uuid string) (cmodels.Review, bool, error) {
	review, err := app.conversation.GetReview(uuid)
	if err != nil {
		return review, false, err
	}
	allowed, err := app.address.CanAccess(user.ID, review.AddressID)
	if err != nil {
		return review, false, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	isAuthor := review.AuthorID == user.ID
	reviewer := canReview(user) && !isAuthor
	if !allowed || (!isAuthor && !reviewer) {
		return review, false, envelope.NewError(envelope.NotFoundError, "This email is no longer waiting for review.", nil)
	}
	return review, isAuthor, nil
}

func handleGetReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	review, _, err := loadReviewFor(app, user, r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	prepareReviewDisplay(app, &review)
	return r.SendEnvelope(review)
}

// requireReviewer allows a decision only from a reviewer who is not the author
// and whose address can still send mail.
func requireReviewer(app *App, user umodels.User, uuid string) (cmodels.Review, error) {
	review, isAuthor, err := loadReviewFor(app, user, uuid)
	if err != nil {
		return review, err
	}
	if isAuthor || !canReview(user) {
		return review, envelope.NewError(envelope.PermissionError, "You can't review your own email.", nil)
	}
	return review, nil
}

func handleApproveReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	review, err := requireReviewer(app, user, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err := sendableAddress(app, user.ID, review.AddressID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	decided, _, err := app.conversation.ApproveReview(uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if decided.Kind == cmodels.ReviewKindNew && decided.ConversationUUID.Valid {
		if created, err := app.conversation.GetConversation(0, decided.ConversationUUID.String, ""); err == nil {
			app.webhook.TriggerEvent(wmodels.EventConversationCreated, created)
		}
	}
	return r.SendEnvelope(decided)
}

func handleDenyReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	if _, err := requireReviewer(app, user, uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req reviewDecisionReq
	if len(r.RequestCtx.PostBody()) > 0 {
		if err := r.Decode(&req, "json"); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
		}
	}
	decided, err := app.conversation.ReturnReview(uuid, &user, req.Note)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(decided)
}

// requireAuthor allows withdraw, edit, discard and dismiss only by the author.
func requireAuthor(app *App, user umodels.User, uuid string) error {
	_, isAuthor, err := loadReviewFor(app, user, uuid)
	if err != nil {
		return err
	}
	if !isAuthor {
		return envelope.NewError(envelope.PermissionError, app.i18n.T("status.deniedPermission"), nil)
	}
	return nil
}

func handleWithdrawReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	if err := requireAuthor(app, user, uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	review, err := app.conversation.ReturnReview(uuid, nil, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(review)
}

// handleResubmitReview sends an edited, returned new email back for review.
func handleResubmitReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	if err := requireAuthor(app, user, uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !canSubmitForReview(user) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	var req outgoingReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	entry, err := sendableAddress(app, user.ID, req.AddressID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Files a returned submission still holds are the author's unsent uploads.
	media, err := getUnassociatedMedia(app, req.Attachments, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	review, err := app.conversation.ResubmitReview(uuid, user.ID, cmodels.ReviewInput{
		AddressID: entry.ID, Subject: req.Subject, Content: req.Content, To: req.To, CC: req.CC, BCC: req.BCC, Media: media,
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(review)
}

func handleDiscardReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	if err := requireAuthor(app, user, uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.DiscardReview(uuid, user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleDismissReview(r *fastglue.Request) error {
	app := r.Context.(*App)
	user, err := loadAgent(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	if err := requireAuthor(app, user, uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.DismissReview(uuid, user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// prepareReviewDisplay sanitizes a submission for display, showing its own
// inline images and blocking everything external.
func prepareReviewDisplay(app *App, review *cmodels.Review) {
	content := review.Content
	trusted := []string{}
	for _, file := range review.Attachments {
		if file.Inline {
			content = strings.ReplaceAll(content, "cid:ldsk-"+strings.ToLower(file.UUID), file.URL)
		}
		if isDisplayImage(file.ContentType) {
			trusted = append(trusted, file.URL)
		}
	}
	display := loadResourcePolicy(app).PrepareDisplayWithImages(content, trusted, func(string) string { return "" })
	review.Display = &display
}
