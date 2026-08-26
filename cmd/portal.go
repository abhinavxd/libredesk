package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/mail"
	"net/url"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	smodels "github.com/abhinavxd/libredesk/internal/conversation/status/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/helpcenter"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	oidcmodels "github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/abhinavxd/libredesk/internal/portalform"
	pfmodels "github.com/abhinavxd/libredesk/internal/portalform/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	tmpl "github.com/abhinavxd/libredesk/internal/template"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/knadh/go-i18n"
	"github.com/microcosm-cc/bluemonday"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
	"golang.org/x/oauth2"
)

const (
	portalSessionPrefix     = "portal_session:"
	portalOTPPrefix         = "portal:otp:"
	portalOTPSendsPrefix    = "portal:otp:sends:"
	portalOTPResendPrefix   = "portal:otp:resend:"
	portalOIDCStatePrefix   = "portal-"
	portalOIDCFlowPrefix    = "portal:oidc:"
	portalWidgetSessionPath = "/portal/widget-session"
	portalViaCode           = "code"
	portalViaOIDC           = "sso"
	portalSessionCookie     = "libredesk_portal_session"
	portalOIDCStateCookie   = "libredesk_portal_oidc_state"
	portalLocaleCookie      = "libredesk_portal_locale"
	portalGuestCookie       = "libredesk_portal_guest"
	portalSessionTTL        = 30 * 24 * time.Hour
	portalGuestTTL          = 2 * time.Hour
	portalLocaleTTL         = 365 * 24 * time.Hour
	portalOTPTTL            = 10 * time.Minute
	portalOTPSendsTTL       = 30 * time.Minute
	portalOTPResendTTL      = 30 * time.Second
	portalOTPMaxAttempts    = 3
	portalOTPMaxSends       = 3
	portalMaxSubjectLength  = 255
	portalMaxFieldLength    = 500
	portalMaxAttachments    = 5
	portalTicketsPageSize   = 20
	portalMessagesPageSize  = 200

	ctxPortalContactID = "portal_contact_id"
	ctxPortalCSRF      = "portal_csrf"
)

// class/id survive sanitization because the quoted-text CSS matches on email clients' quote markers.
var portalHTMLPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class", "id").Globally()
	return p
}()

// portalQuoteMarkers mirrors QUOTE_MARKERS in frontend/shared-ui/utils/quotedContent.js.
var portalQuoteMarkers = []string{
	"<blockquote",
	`id="divRplyFwdMsg"`,
	`id="appendonsend"`,
	`id="OLK_SRC_BODY_SECTION"`,
	`class="OutlookMessageHeader"`,
	`class="yahoo_quoted"`,
	"gmail_quote_container",
}

// checkPortalOTPScript matches the pending code, deleting it on match, expiry corruption, or the attempt cap.
var checkPortalOTPScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local ok, p = pcall(cjson.decode, raw)
if not ok or type(p) ~= 'table' or type(p.code) ~= 'string' or p.code == '' then
	redis.call('DEL', KEYS[1])
	return 0
end
if p.code == ARGV[1] then
	redis.call('DEL', KEYS[1])
	return 1
end
p.attempts = (p.attempts or 0) + 1
if p.attempts >= tonumber(ARGV[2]) then
	redis.call('DEL', KEYS[1])
else
	redis.call('SET', KEYS[1], cjson.encode(p), 'KEEPTTL')
end
return 0
`)

// incrPortalOTPSendsScript sets the counter's TTL only on the first increment, so the window does not slide.
var incrPortalOTPSendsScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return n
`)

// Cached once: the embedded filesystem never changes at runtime.
var portalLanguagePacks = struct {
	once sync.Once
	list []portalLanguage
}{}

// portalPendingOTP is the JSON stored at portalOTPPrefix while a code awaits entry.
type portalPendingOTP struct {
	Code     string `json:"code"`
	Attempts int    `json:"attempts"`
}

type portalLanguage struct {
	Code string
	Name string
	Path string
}

// portalFieldView is one portal ticket form field rendered with the answer already submitted for it.
type portalFieldView struct {
	pfmodels.Field
	Value   string
	Checked bool
}

type portalTicketRow struct {
	ReferenceNumber   string
	Subject           string
	StatusLabel       string
	StatusClass       string
	CreatedAt         string
	CreatedAtISO      string
	LastActivityAt    string
	LastActivityAtISO string
	ReplyLabel        string
}

type portalMessageView struct {
	AuthorName    string
	IsAgent       bool
	CreatedAt     string
	CreatedAtISO  string
	HTML          template.HTML
	Text          string
	HasQuoted     bool
	GroupWithPrev bool
	GroupWithNext bool
	Attachments   attachment.Attachments
}

// portalPage requires a valid portal session. POSTs must also carry the session's CSRF token.
func portalPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)
		if !app.consts.Load().(*constants).PortalEnabled {
			r.RequestCtx.NotFound()
			return nil
		}

		token := string(r.RequestCtx.Request.Header.Cookie(portalSessionCookie))
		contactID, csrf, err := loadPortalSession(app, token)
		if err != nil {
			return redirectPortalLoginReturn(r)
		}

		u, err := app.user.Get(contactID, "", []string{umodels.UserTypeContact})
		if err != nil || !u.Enabled {
			deletePortalSession(app, token)
			clearPortalSessionCookie(r)
			return redirectPortalLoginReturn(r)
		}

		if r.RequestCtx.IsPost() {
			if string(r.RequestCtx.FormValue("csrf")) != csrf {
				return redirectPortalLogin(r)
			}
		}

		r.RequestCtx.SetUserValue(ctxPortalContactID, contactID)
		r.RequestCtx.SetUserValue(ctxPortalCSRF, csrf)
		return handler(r)
	}
}

// portalGuestPage serves the login flow. GETs mint the guest CSRF token, POSTs must carry it back.
func portalGuestPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)
		if !app.consts.Load().(*constants).PortalEnabled {
			r.RequestCtx.NotFound()
			return nil
		}
		token := string(r.RequestCtx.Request.Header.Cookie(portalSessionCookie))
		if token != "" {
			if _, _, err := loadPortalSession(app, token); err == nil && r.RequestCtx.IsGet() {
				return r.RedirectURI("/portal", fasthttp.StatusFound, nil, "")
			}
		}

		guest := string(r.RequestCtx.Request.Header.Cookie(portalGuestCookie))
		if r.RequestCtx.IsPost() {
			if guest == "" || string(r.RequestCtx.FormValue("csrf")) != guest {
				return redirectPortalLogin(r)
			}
		} else if guest == "" {
			fresh, err := stringutil.RandomAlphanumeric(32)
			if err != nil {
				app.lo.Error("error generating portal guest token", "error", err)
				return renderPortalError(r, fasthttp.StatusInternalServerError)
			}
			guest = fresh
		}
		setPortalCookie(r, portalGuestCookie, guest, "/portal", portalGuestTTL)
		r.RequestCtx.SetUserValue(ctxPortalCSRF, guest)
		return handler(r)
	}
}

func handlePortalTickets(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		contactID = r.RequestCtx.UserValue(ctxPortalContactID).(int)
		lcl       = portalI18n(app, r)
		filter    = string(r.RequestCtx.QueryArgs().Peek("status"))
		page, _   = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	)
	if filter != "open" && filter != "resolved" {
		filter = ""
	}
	if page < 1 {
		page = 1
	}

	conversations, total, err := app.conversation.GetContactPortalConversations(contactID, filter, page, portalTicketsPageSize)
	if err != nil {
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}

	loc := portalTimezone(app)
	rows := make([]portalTicketRow, 0, len(conversations))
	for _, c := range conversations {
		lastActivity := c.CreatedAt
		if c.LastMessageAt.Valid {
			lastActivity = c.LastMessageAt.Time
		}
		label, class := portalStatusLabel(lcl, c.StatusCategory, c.LastMessageSender.String)
		rows = append(rows, portalTicketRow{
			ReferenceNumber:   c.ReferenceNumber,
			Subject:           portalSubject(lcl, c.Subject.String),
			StatusLabel:       label,
			StatusClass:       class,
			CreatedAt:         c.CreatedAt.In(loc).Format("Jan 2, 2006"),
			CreatedAtISO:      c.CreatedAt.In(loc).Format(time.RFC3339),
			LastActivityAt:    lastActivity.In(loc).Format("Jan 2, 2006 15:04"),
			LastActivityAtISO: lastActivity.In(loc).Format(time.RFC3339),
			ReplyLabel:        portalReplyLabel(lcl, c.ReplyCount),
		})
	}

	totalPages := (total + portalTicketsPageSize - 1) / portalTicketsPageSize
	return renderPortalPage(r, "portal-tickets", lcl.T("portal.myTickets"), map[string]interface{}{
		"Tickets":      rows,
		"StatusFilter": filter,
		"Page":         page,
		"TotalPages":   totalPages,
		"PrevPage":     page - 1,
		"NextPage":     page + 1,
		"CanCreate":    portalInboxID(r) > 0 && !app.consts.Load().(*constants).PortalTicketsFromArticleOnly,
	})
}

func handlePortalTicketView(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		lcl = portalI18n(app, r)
	)
	conversation, err := getPortalConversation(r)
	if err != nil {
		return renderPortalError(r, fasthttp.StatusNotFound)
	}

	private := false
	messages, _, err := app.conversation.GetConversationMessages(conversation.UUID, 1, portalMessagesPageSize, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing})
	if err != nil {
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].CreatedAt.Before(messages[j].CreatedAt) })

	loc := portalTimezone(app)
	views := make([]portalMessageView, 0, len(messages))
	for i, msg := range messages {
		app.conversation.SignAttachmentURLs(msg.Attachments)
		view := portalMessageView{
			AuthorName:    portalAuthorName(lcl, msg),
			IsAgent:       msg.SenderType == cmodels.SenderTypeAgent,
			CreatedAt:     msg.CreatedAt.In(loc).Format("Jan 2, 2006 15:04"),
			CreatedAtISO:  msg.CreatedAt.In(loc).Format(time.RFC3339),
			GroupWithPrev: i > 0 && portalCanGroup(messages[i-1], msg),
			GroupWithNext: i < len(messages)-1 && portalCanGroup(msg, messages[i+1]),
			Attachments:   msg.Attachments,
		}
		if msg.ContentType == cmodels.ContentTypeHTML {
			view.HTML = template.HTML(portalHTMLPolicy.Sanitize(msg.Content))
			view.HasQuoted = portalContainsQuoteMarkers(msg.Content)
		} else {
			view.Text = msg.Content
		}
		views = append(views, view)
	}

	label, class := portalStatusLabel(lcl, conversation.StatusCategory.String, conversation.LastMessageSender.String)
	subject := portalSubject(lcl, conversation.Subject.String)
	return renderPortalPage(r, "portal-ticket", subject, map[string]interface{}{
		"ReferenceNumber": conversation.ReferenceNumber,
		"Subject":         subject,
		"StatusLabel":     label,
		"StatusClass":     class,
		"CreatedAt":       conversation.CreatedAt.In(loc).Format("Jan 2, 2006 15:04"),
		"CreatedAtISO":    conversation.CreatedAt.In(loc).Format(time.RFC3339),
		"CSATWidgetURL":   portalCSATWidgetURL(app, conversation.ID),
		"Messages":        views,
		"Error":           string(r.RequestCtx.QueryArgs().Peek("error")),
	})
}

func handlePortalTicketReply(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		contactID = r.RequestCtx.UserValue(ctxPortalContactID).(int)
		lcl       = portalI18n(app, r)
		message   = strings.TrimSpace(string(r.RequestCtx.FormValue("message")))
	)
	conversation, err := getPortalConversation(r)
	if err != nil {
		return renderPortalError(r, fasthttp.StatusNotFound)
	}

	attachments, errKey := portalFormAttachments(r)
	if errKey == "" && message == "" && len(attachments) == 0 {
		errKey = "portal.emptyMessage"
	}
	if errKey == "" && utf8.RuneCountInString(message) > maxChatMessageLength {
		errKey = "portal.messageTooLong"
	}
	if errKey != "" {
		return r.RedirectURI("/portal/tickets/"+conversation.ReferenceNumber, fasthttp.StatusSeeOther, map[string]any{"error": lcl.T(errKey)}, "")
	}

	msg := cmodels.Message{
		ConversationUUID: conversation.UUID,
		ConversationID:   conversation.ID,
		SenderID:         contactID,
		Type:             cmodels.MessageIncoming,
		SenderType:       cmodels.SenderTypeContact,
		Status:           cmodels.MessageStatusReceived,
		Content:          message,
		ContentType:      cmodels.ContentTypeText,
		Private:          false,
		Attachments:      attachments,
	}
	if _, err := app.conversation.ProcessIncomingContactMessage(msg, false); err != nil {
		app.lo.Error("error processing portal reply", "conversation_uuid", conversation.UUID, "error", err)
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}

	return r.RedirectURI("/portal/tickets/"+conversation.ReferenceNumber, fasthttp.StatusSeeOther, nil, "")
}

// handlePortalNewTicket renders the new-ticket form, seeding the subject from the article the contact came from.
func handlePortalNewTicket(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		lcl         = portalI18n(app, r)
		articleSlug = string(r.RequestCtx.QueryArgs().Peek("article"))
	)
	if portalInboxID(r) <= 0 {
		return renderPortalError(r, fasthttp.StatusNotFound)
	}
	subject := portalArticleSubject(app, articleSlug)
	if app.consts.Load().(*constants).PortalTicketsFromArticleOnly && subject == "" {
		return renderPortalError(r, fasthttp.StatusNotFound)
	}
	form := portalTicketForm(app, articleSlug)
	return renderPortalPage(r, "portal-new-ticket", lcl.T("portal.newTicket"), map[string]interface{}{
		"Subject":    subject,
		"Article":    articleSlug,
		"AskSubject": form.AskSubject,
		"FormFields": portalFieldViews(r, form, false),
		"FormName":   form.Name,
	})
}

func handlePortalCreateTicket(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		contactID   = r.RequestCtx.UserValue(ctxPortalContactID).(int)
		lcl         = portalI18n(app, r)
		subject     = strings.TrimSpace(string(r.RequestCtx.FormValue("subject")))
		message     = strings.TrimSpace(string(r.RequestCtx.FormValue("message")))
		articleSlug = strings.TrimSpace(string(r.RequestCtx.FormValue("article")))
		form        = portalTicketForm(app, articleSlug)
	)
	if !form.AskSubject {
		subject = portalArticleSubject(app, articleSlug)
		if subject == "" {
			subject = form.Name
		}
	}
	retry := func(errMsg string) error {
		return renderPortalPage(r, "portal-new-ticket", lcl.T("portal.newTicket"), map[string]interface{}{
			"Error":      errMsg,
			"Subject":    subject,
			"Message":    message,
			"Article":    articleSlug,
			"AskSubject": form.AskSubject,
			"FormFields": portalFieldViews(r, form, true),
			"FormName":   form.Name,
		})
	}

	inboxID := portalInboxID(r)
	if inboxID <= 0 {
		return renderPortalError(r, fasthttp.StatusNotFound)
	}
	inboxRecord, err := app.inbox.GetDBRecord(inboxID)
	if err != nil || !inboxRecord.Enabled || inboxRecord.Channel != inbox.ChannelEmail {
		app.lo.Error("portal inbox is missing, disabled or not an email inbox", "inbox_id", inboxID)
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}

	attachments, errKey := portalFormAttachments(r)
	switch {
	case errKey != "":
	case subject == "":
		errKey = "portal.emptySubject"
	case utf8.RuneCountInString(subject) > portalMaxSubjectLength:
		errKey = "portal.subjectTooLong"
	case message == "":
		errKey = "portal.emptyMessage"
	case utf8.RuneCountInString(message) > maxChatMessageLength:
		errKey = "portal.messageTooLong"
	}
	if errKey != "" {
		return retry(lcl.T(errKey))
	}

	attrs, headerLines, errKey, fieldLabel := portalFormAnswers(r, form)
	if errKey != "" {
		return retry(lcl.Ts(errKey, "name", fieldLabel))
	}
	if articleTitle := portalArticleSubject(app, articleSlug); articleTitle != "" {
		headerLines = append(headerLines, [2]string{lcl.Tc("globals.terms.article", 1), articleTitle})
	}
	if via := portalSessionVia(app, r, lcl); via != "" {
		headerLines = append(headerLines, [2]string{lcl.T("portal.signedInVia"), via})
	}
	if block := portalform.RenderHeaderBlock(headerLines); block != "" {
		message = block + "\n" + message
	}

	meta := map[string]any{
		"ip":         realip.FromRequest(r.RequestCtx),
		"user_agent": string(r.RequestCtx.Request.Header.Peek("User-Agent")),
	}
	_, conversationUUID, err := app.conversation.CreateConversation(contactID, inboxID, "", time.Now(), subject, false, meta, attrs,
		maxChatConversationsPerContact, chatConversationRateLimitWindow)
	if err != nil {
		app.lo.Error("error creating portal conversation", "contact_id", contactID, "inbox_id", inboxID, "error", err)
		var envErr envelope.Error
		if errors.As(err, &envErr) && envErr.ErrorType == envelope.RateLimitError {
			return retry(lcl.T("globals.messages.tooManyRequests"))
		}
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}

	msg := cmodels.Message{
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		Type:             cmodels.MessageIncoming,
		SenderType:       cmodels.SenderTypeContact,
		Status:           cmodels.MessageStatusReceived,
		Content:          message,
		ContentType:      cmodels.ContentTypeText,
		Private:          false,
		Attachments:      attachments,
	}
	if _, err := app.conversation.ProcessIncomingContactMessage(msg, true); err != nil {
		app.lo.Error("error inserting portal ticket message", "conversation_uuid", conversationUUID, "error", err)
		if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
			app.lo.Error("error deleting conversation after portal message insert failure", "conversation_uuid", conversationUUID, "error", err)
		}
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		return r.RedirectURI("/portal", fasthttp.StatusSeeOther, nil, "")
	}
	return r.RedirectURI("/portal/tickets/"+conversation.ReferenceNumber, fasthttp.StatusSeeOther, nil, "")
}

func handlePortalLoginPage(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		lcl    = portalI18n(app, r)
		errMsg = ""
	)
	switch string(r.RequestCtx.QueryArgs().Peek("error")) {
	case "oidc":
		errMsg = lcl.T("portal.oidcFailed")
	case "agent_email":
		errMsg = lcl.T("portal.agentEmail")
	case "disabled":
		errMsg = lcl.T("portal.accountDisabled")
	}
	return renderPortalPage(r, "portal-login", lcl.T("auth.signInButton"), map[string]interface{}{
		"Providers": portalOIDCProviders(app),
		"Error":     errMsg,
		"Return":    portalReturnPath(r, string(r.RequestCtx.QueryArgs().Peek("return"))),
	})
}

func handlePortalSendCode(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		lcl       = portalI18n(app, r)
		email     = normalizePortalEmail(string(r.RequestCtx.FormValue("email")))
		returnTo  = portalReturnPath(r, string(r.RequestCtx.FormValue("return")))
		ctx       = context.Background()
		loginPage = func(errKey string) error {
			return renderPortalPage(r, "portal-login", lcl.T("auth.signInButton"), map[string]interface{}{
				"Providers": portalOIDCProviders(app),
				"Error":     lcl.T(errKey),
				"Return":    returnTo,
			})
		}
	)
	if _, err := mail.ParseAddress(email); err != nil {
		return loginPage("portal.invalidEmail")
	}

	// Resend throttle and send cap, both per address.
	ok, err := app.redis.SetNX(ctx, portalOTPResendPrefix+email, "1", portalOTPResendTTL).Result()
	if err != nil {
		app.lo.Error("error throttling portal otp sends", "error", err)
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}
	if ok {
		sends, err := incrPortalOTPSendsScript.Run(ctx, app.redis, []string{portalOTPSendsPrefix + email}, int(portalOTPSendsTTL.Seconds())).Int()
		if err != nil {
			app.lo.Error("error counting portal otp sends", "error", err)
			return renderPortalError(r, fasthttp.StatusInternalServerError)
		}
		if sends > portalOTPMaxSends {
			return loginPage("portal.tooManyCodeRequests")
		}
		if err := sendPortalLoginCode(app, email); err != nil {
			app.lo.Error("error sending portal login code", "error", err)
			return renderPortalError(r, fasthttp.StatusInternalServerError)
		}
	}

	// The response never reveals whether the address is known.
	return renderPortalPage(r, "portal-verify", lcl.T("portal.enterCode"), map[string]interface{}{
		"Email":  email,
		"Return": returnTo,
	})
}

// The verify page's state arrives with the login POST, so a direct GET has nothing to render.
func handlePortalVerifyPage(r *fastglue.Request) error {
	return r.RedirectURI("/portal/login", fasthttp.StatusFound, nil, "")
}

func handlePortalVerifyCode(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		lcl      = portalI18n(app, r)
		email    = normalizePortalEmail(string(r.RequestCtx.FormValue("email")))
		code     = strings.TrimSpace(string(r.RequestCtx.FormValue("code")))
		returnTo = portalReturnPath(r, string(r.RequestCtx.FormValue("return")))
		ctx      = context.Background()
	)
	if _, err := mail.ParseAddress(email); err != nil {
		return redirectPortalLogin(r)
	}

	res, err := checkPortalOTPScript.Run(ctx, app.redis, []string{portalOTPPrefix + email}, code, portalOTPMaxAttempts).Int()
	if err != nil {
		app.lo.Error("error checking portal otp", "error", err)
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}
	if res != 1 {
		return renderPortalPage(r, "portal-verify", lcl.T("portal.enterCode"), map[string]interface{}{
			"Email":  email,
			"Error":  lcl.T("portal.invalidCode"),
			"Return": returnTo,
		})
	}

	return startPortalSession(r, email, portalViaCode, returnTo)
}

// handlePortalLogout also drops the widget session minted alongside the portal one.
func handlePortalLogout(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		contactID = r.RequestCtx.UserValue(ctxPortalContactID).(int)
		token     = string(r.RequestCtx.Request.Header.Cookie(portalSessionCookie))
	)
	deletePortalSession(app, token)
	clearPortalSessionCookie(r)
	deletePortalWidgetSession(app, contactID)
	return r.RedirectURI("/portal/login", fasthttp.StatusSeeOther, nil, "")
}

func handlePortalSetLocale(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		lang = string(r.RequestCtx.QueryArgs().Peek("lang"))
	)
	if !app.consts.Load().(*constants).PortalEnabled {
		r.RequestCtx.NotFound()
		return nil
	}
	if slices.Contains(portalLocales(app), lang) {
		setPortalCookie(r, portalLocaleCookie, lang, "/portal", portalLocaleTTL)
	}
	redirectPath(r.RequestCtx, portalReturnPath(r, string(r.RequestCtx.QueryArgs().Peek("return"))), fasthttp.StatusSeeOther)
	return nil
}

func handlePortalWidgetSession(r *fastglue.Request) error {
	app := r.Context.(*App)
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, no-store")
	if !app.consts.Load().(*constants).PortalEnabled {
		r.RequestCtx.NotFound()
		return nil
	}
	contactID, _, err := loadPortalSession(app, string(r.RequestCtx.Request.Header.Cookie(portalSessionCookie)))
	if err != nil {
		return r.SendEnvelope(map[string]any{"session_token": ""})
	}
	return r.SendEnvelope(map[string]any{"session_token": portalWidgetSessionToken(app, contactID)})
}

func handlePortalOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if !app.consts.Load().(*constants).PortalEnabled {
		r.RequestCtx.NotFound()
		return nil
	}
	if err != nil {
		return redirectPortalLoginError(r, "oidc")
	}

	provider, err := app.oidc.Get(providerID)
	if err != nil || !provider.Enabled || !provider.EnabledForPortal {
		app.lo.Warn("portal oidc login attempted for a provider that is not enabled for the portal", "provider_id", providerID)
		return redirectPortalLoginError(r, "oidc")
	}

	nonce, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("error generating portal oidc state", "error", err)
		return redirectPortalLoginError(r, "oidc")
	}
	idNonce, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("error generating portal oidc nonce", "error", err)
		return redirectPortalLoginError(r, "oidc")
	}
	state := portalOIDCStatePrefix + nonce
	codeVerifier := oauth2.GenerateVerifier()

	if err := app.redis.HSet(context.Background(), portalOIDCFlowPrefix+state, map[string]any{
		"nonce":         idNonce,
		"code_verifier": codeVerifier,
		"provider_id":   providerID,
		"return_to":     portalReturnPath(r, string(r.RequestCtx.QueryArgs().Peek("return"))),
	}).Err(); err != nil {
		app.lo.Error("error saving portal oidc flow", "error", err)
		return redirectPortalLoginError(r, "oidc")
	}
	app.redis.Expire(context.Background(), portalOIDCFlowPrefix+state, portalOTPTTL)

	authURL, err := app.auth.LoginURL(providerID, state, idNonce, codeVerifier)
	if err != nil {
		app.lo.Error("error getting oidc login url for portal", "provider_id", providerID, "error", err)
		return redirectPortalLoginError(r, "oidc")
	}

	// The callback URL is outside /portal, so the state cookie spans the whole site.
	setPortalCookie(r, portalOIDCStateCookie, state, "/", portalOTPTTL)
	return r.Redirect(authURL, fasthttp.StatusFound, nil, "")
}

func handlePortalOIDCCallback(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		code            = string(r.RequestCtx.QueryArgs().Peek("code"))
		state           = string(r.RequestCtx.QueryArgs().Peek("state"))
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if !app.consts.Load().(*constants).PortalEnabled {
		r.RequestCtx.NotFound()
		return nil
	}
	if err != nil {
		return redirectPortalLoginError(r, "oidc")
	}

	cookieState := string(r.RequestCtx.Request.Header.Cookie(portalOIDCStateCookie))
	clearPortalCookie(r, portalOIDCStateCookie, "/")
	if cookieState == "" || state != cookieState {
		app.lo.Error("portal oidc state mismatch", "provider_id", providerID)
		return redirectPortalLoginError(r, "oidc")
	}

	provider, err := app.oidc.Get(providerID)
	if err != nil || !provider.Enabled || !provider.EnabledForPortal {
		app.lo.Warn("portal oidc callback for a provider that is not enabled for the portal", "provider_id", providerID)
		return redirectPortalLoginError(r, "oidc")
	}

	ctx := context.Background()
	flow, err := app.redis.HGetAll(ctx, portalOIDCFlowPrefix+state).Result()
	app.redis.Del(ctx, portalOIDCFlowPrefix+state)
	if err != nil || flow["nonce"] == "" || flow["provider_id"] != strconv.Itoa(providerID) {
		app.lo.Error("portal oidc flow expired or does not match the callback", "provider_id", providerID, "error", err)
		return redirectPortalLoginError(r, "oidc")
	}

	if oauthErr := string(r.RequestCtx.QueryArgs().Peek("error")); oauthErr != "" {
		app.lo.Warn("portal oidc provider returned an error", "provider_id", providerID, "oauth_error", oauthErr)
		return redirectPortalLoginError(r, "oidc")
	}

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code, flow["code_verifier"], flow["nonce"])
	if err != nil {
		app.lo.Error("error exchanging portal oidc token", "provider_id", providerID, "error", err)
		return redirectPortalLoginError(r, "oidc")
	}

	email := normalizePortalEmail(claims.Email)
	if _, err := mail.ParseAddress(email); err != nil {
		app.lo.Warn("portal oidc claims carry no usable email", "provider_id", providerID)
		return redirectPortalLoginError(r, "oidc")
	}

	returnTo := flow["return_to"]
	if returnTo == "" {
		returnTo = "/portal"
	}
	return startPortalSession(r, email, portalViaOIDC, returnTo)
}

func startPortalSession(r *fastglue.Request, email, via, returnTo string) error {
	app := r.Context.(*App)

	// Agent emails never get a portal session.
	if agent, err := app.user.GetAgent(0, email); err == nil && agent.ID > 0 {
		return redirectPortalLoginError(r, "agent_email")
	}

	user := umodels.User{Email: null.StringFrom(email), FirstName: portalContactFirstName(email)}
	if err := app.user.ResolveContact(&user, umodels.ContactReuse); err != nil {
		app.lo.Error("error resolving portal contact", "error", err)
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}

	u, err := app.user.Get(user.ID, "", []string{umodels.UserTypeContact})
	if err != nil || !u.Enabled {
		app.lo.Warn("portal login rejected for disabled or missing contact", "contact_id", user.ID)
		return redirectPortalLoginError(r, "disabled")
	}

	return finishPortalLogin(r, user.ID, returnTo, via)
}

func finishPortalLogin(r *fastglue.Request, contactID int, returnTo, via string) error {
	app := r.Context.(*App)
	token, err := createPortalSession(app, contactID, via)
	if err != nil {
		app.lo.Error("error creating portal session", "error", err)
		return renderPortalError(r, fasthttp.StatusInternalServerError)
	}
	// A same-name cookie left at /portal is more specific than the one below and would win the lookup.
	clearPortalCookie(r, portalSessionCookie, "/portal")
	setPortalCookie(r, portalSessionCookie, token, "/", portalSessionTTL)
	app.lo.Info("portal login successful", "contact_id", contactID)
	return r.Redirect(returnTo, fasthttp.StatusSeeOther, nil, "")
}

func sendPortalLoginCode(app *App, email string) error {
	code, err := stringutil.RandomNumeric(6)
	if err != nil {
		return err
	}
	b, err := json.Marshal(portalPendingOTP{Code: code})
	if err != nil {
		return err
	}
	if err := app.redis.Set(context.Background(), portalOTPPrefix+email, b, portalOTPTTL).Err(); err != nil {
		return err
	}

	lcl := localeI18n(app, portalDefaultLocale(app))
	content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplPortalLoginCode, map[string]string{
		"Code":    code,
		"Heading": lcl.T("portal.loginCodeEmailSubject"),
		"Body":    lcl.T("portal.loginCodeEmailBody"),
		"Ignore":  lcl.T("portal.loginCodeEmailIgnore"),
	})
	if err != nil {
		return err
	}
	return app.notifier.Send(notifier.Message{
		RecipientEmails: []string{email},
		Subject:         lcl.T("portal.loginCodeEmailSubject"),
		Content:         content,
		Provider:        notifier.ProviderEmail,
	})
}

// getPortalConversation rejects a conversation that does not belong to the session contact.
func getPortalConversation(r *fastglue.Request) (cmodels.Conversation, error) {
	var (
		app       = r.Context.(*App)
		contactID = r.RequestCtx.UserValue(ctxPortalContactID).(int)
		refNum, _ = r.RequestCtx.UserValue("number").(string)
	)
	conversation, err := app.conversation.GetConversation(0, "", refNum)
	if err != nil {
		return cmodels.Conversation{}, err
	}
	if conversation.ContactID != contactID {
		app.lo.Warn("portal access denied to conversation", "reference_number", refNum, "contact_id", contactID, "conversation_contact_id", conversation.ContactID)
		return cmodels.Conversation{}, fmt.Errorf("conversation does not belong to contact")
	}
	return conversation, nil
}

// portalFormAttachments returns a non-empty i18n key when a file is rejected.
func portalFormAttachments(r *fastglue.Request) (attachment.Attachments, string) {
	app := r.Context.(*App)
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return nil, ""
	}
	files := form.File["files"]
	if len(files) == 0 {
		return nil, ""
	}
	if len(files) > portalMaxAttachments {
		return nil, "portal.tooManyAttachments"
	}

	consts := app.consts.Load().(*constants)
	attachments := make(attachment.Attachments, 0, len(files))
	for _, fileHeader := range files {
		if fileHeader.Size <= 0 {
			continue
		}
		if bytesToMegabytes(fileHeader.Size) > float64(consts.MaxFileUploadSizeMB) {
			return nil, "portal.attachmentTooLarge"
		}
		name := stringutil.SanitizeFilename(fileHeader.Filename)
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
		if !slices.Contains(consts.AllowedUploadFileExtensions, "*") && !slices.Contains(consts.AllowedUploadFileExtensions, ext) {
			return nil, "portal.attachmentTypeNotAllowed"
		}
		file, err := fileHeader.Open()
		if err != nil {
			return nil, "portal.attachmentReadFailed"
		}
		content := make([]byte, fileHeader.Size)
		if _, err := io.ReadFull(file, content); err != nil {
			file.Close()
			return nil, "portal.attachmentReadFailed"
		}
		file.Close()
		attachments = append(attachments, attachment.Attachment{
			Name:        name,
			ContentType: fileHeader.Header.Get("Content-Type"),
			Size:        int(fileHeader.Size),
			Content:     content,
			Disposition: attachment.DispositionAttachment,
		})
	}
	return attachments, ""
}

func renderPortalPage(r *fastglue.Request, page, title string, data map[string]interface{}) error {
	app := r.Context.(*App)
	locale := portalLocale(app, r)
	lcl := localeI18n(app, locale)

	data["Title"] = title
	data["Locale"] = locale
	data["LocaleLinks"] = portalLocaleLinks(app, r)
	data["Dir"] = localeDir(locale)
	data["CSRF"], _ = r.RequestCtx.UserValue(ctxPortalCSRF).(string)
	_, loggedIn := r.RequestCtx.UserValue(ctxPortalContactID).(int)
	data["LoggedIn"] = loggedIn
	data["Brand"] = portalBrand(app)
	data["MaxMessageLength"] = maxChatMessageLength
	data["AcceptAttachments"] = portalAcceptAttachments(app)
	data["LivechatInboxUUID"] = portalLivechatInboxUUID(app)
	data["LivechatSessionPath"] = portalWidgetSessionPath

	err := app.tmpl.RenderWebPage(r.RequestCtx, page, map[string]interface{}{
		"L":    lcl,
		"Data": data,
	})
	if err != nil {
		app.lo.Error("error rendering portal page", "page", page, "error", err)
		return err
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "private, no-store")
	r.RequestCtx.Response.Header.Set("X-Robots-Tag", noIndexHeader)
	r.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	r.RequestCtx.Response.Header.Set("X-Frame-Options", "SAMEORIGIN")
	r.RequestCtx.Response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	return nil
}

func renderPortalError(r *fastglue.Request, status int) error {
	app := r.Context.(*App)
	lcl := portalI18n(app, r)
	headingKey := "globals.messages.pageNotFound"
	if status == fasthttp.StatusInternalServerError {
		headingKey = "globals.messages.somethingWentWrong"
	}
	err := renderPortalPage(r, "portal-error", lcl.T(headingKey), map[string]interface{}{
		"ErrorCode": strconv.Itoa(status),
	})
	r.RequestCtx.SetStatusCode(status)
	return err
}

// portalBrand themes the portal from the linked help center, falling back to the app defaults.
func portalBrand(app *App) map[string]interface{} {
	consts := app.consts.Load().(*constants)
	brand := map[string]interface{}{
		"Name":              consts.SiteName,
		"LogoURL":           consts.LogoURL,
		"Template":          hcmodels.TemplateClassic,
		"Color":             hcmodels.DefaultTheme().Color,
		"ThemeCSS":          template.CSS(""),
		"CustomCSS":         template.CSS(""),
		"CustomJS":          template.JS(""),
		"NavLinks":          []hcmodels.NavLink{},
		"FooterLinks":       []hcmodels.NavLink{},
		"SocialLinks":       []hcmodels.SocialLink{},
		"FooterTaglineHTML": template.HTML(""),
		"HelpCenterURL":     "",
		"HelpCenterSlug":    "",
		"ArticleBasePath":   "",
		"ArticleLocale":     "",
		"AnnouncementHTML":  template.HTML(""),
		"AnnouncementKey":   "",
		"AnnouncementLink":  "",
		"AnnouncementText":  "",
		"Favicon":           "",
	}
	if consts.PortalHelpCenterID <= 0 {
		return brand
	}
	hc, err := app.helpcenter.GetHelpCenterByID(consts.PortalHelpCenterID)
	if err != nil || !hc.IsActive {
		return brand
	}
	theme := helpCenterTheme(hc)
	brand["Name"] = hc.Name
	if hc.Template != "" {
		brand["Template"] = hc.Template
	}
	if theme.LogoURL != "" {
		brand["LogoURL"] = publicAssetPaths(app, theme.LogoURL)
	}
	brand["NavLinks"] = theme.NavLinks
	brand["FooterLinks"] = theme.FooterLinks
	brand["SocialLinks"] = theme.SocialLinks
	brand["FooterTaglineHTML"] = template.HTML(helpcenter.RenderInlineMarkdown(theme.Footer.Tagline))
	brand["Color"] = theme.Color
	brand["Favicon"] = publicAssetPaths(app, theme.Favicon)
	theme.Header.BackgroundImage = publicAssetPaths(app, theme.Header.BackgroundImage)
	brand["ThemeCSS"] = buildThemeCSSVars(theme)
	brand["CustomCSS"] = template.CSS(hc.CustomCSS)
	brand["CustomJS"] = template.JS(hc.CustomJS)
	brand["HelpCenterURL"] = helpCenterBaseURL(app, hc) + helpCenterHomePath(hc, hc.DefaultLocale)
	brand["HelpCenterSlug"] = hc.Slug
	brand["ArticleBasePath"] = helpCenterHomePath(hc, hc.DefaultLocale) + "/articles/"
	brand["ArticleLocale"] = hc.DefaultLocale
	brand["AnnouncementHTML"] = template.HTML(helpcenter.RenderInlineMarkdown(theme.Announcement.Text))
	brand["AnnouncementKey"] = announcementKey(hc.Slug, theme.Announcement)
	brand["AnnouncementLink"] = theme.Announcement.LinkURL
	brand["AnnouncementText"] = theme.Announcement.LinkLabel
	return brand
}

// portalAcceptAttachments is empty when every extension is allowed.
func portalAcceptAttachments(app *App) string {
	exts := app.consts.Load().(*constants).AllowedUploadFileExtensions
	if len(exts) == 0 || slices.Contains(exts, "*") {
		return ""
	}
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		out = append(out, "."+strings.TrimPrefix(strings.ToLower(strings.TrimSpace(e)), "."))
	}
	return strings.Join(out, ",")
}

func portalOIDCProviders(app *App) []oidcmodels.OIDC {
	all, err := app.oidc.GetAll()
	if err != nil {
		return nil
	}
	providers := make([]oidcmodels.OIDC, 0, len(all))
	for _, p := range all {
		if !p.Enabled || !p.EnabledForPortal {
			continue
		}
		p.ClientID, p.ClientSecret = "", ""
		p.SetProviderLogo()
		providers = append(providers, p)
	}
	return providers
}

func portalI18n(app *App, r *fastglue.Request) *i18n.I18n {
	return localeI18n(app, portalLocale(app, r))
}

// portalLocale resolves the visitor's locale from the switcher cookie, then Accept-Language, then the default.
func portalLocale(app *App, r *fastglue.Request) string {
	def := portalDefaultLocale(app)
	allowed := portalLocales(app)
	if len(allowed) < 2 {
		return def
	}
	if c := string(r.RequestCtx.Request.Header.Cookie(portalLocaleCookie)); slices.Contains(allowed, c) {
		return c
	}
	if loc, ok := matchAcceptLanguage(string(r.RequestCtx.Request.Header.Peek("Accept-Language")), allowed); ok {
		return loc
	}
	return def
}

func portalDefaultLocale(app *App) string {
	return ko.String("app.lang")
}

func portalLocales(app *App) []string {
	packs := portalLanguages(app)
	codes := make([]string, 0, len(packs))
	for _, p := range packs {
		codes = append(codes, p.Code)
	}
	return codes
}

func portalLanguages(app *App) []portalLanguage {
	portalLanguagePacks.once.Do(func() {
		files, err := app.fs.Glob("/i18n/*.json")
		if err != nil {
			return
		}
		for _, f := range files {
			code := strings.TrimSuffix(filepath.Base(f), ".json")
			name := code
			if b, err := app.fs.Read(f); err == nil {
				var meta map[string]string
				if json.Unmarshal(b, &meta) == nil && meta["_.name"] != "" {
					name = meta["_.name"]
				}
			}
			portalLanguagePacks.list = append(portalLanguagePacks.list, portalLanguage{Code: code, Name: name})
		}
		sort.Slice(portalLanguagePacks.list, func(i, j int) bool {
			return portalLanguagePacks.list[i].Name < portalLanguagePacks.list[j].Name
		})
	})
	return portalLanguagePacks.list
}

func portalLocaleLinks(app *App, r *fastglue.Request) []localeLink {
	packs := portalLanguages(app)
	if len(packs) < 2 {
		return nil
	}
	returnTo := string(r.RequestCtx.URI().RequestURI())
	links := make([]localeLink, 0, len(packs))
	for _, p := range packs {
		links = append(links, localeLink{
			Locale: p.Code,
			Path:   "/portal/locale?lang=" + url.QueryEscape(p.Code) + "&return=" + url.QueryEscape(returnTo),
		})
	}
	return links
}

// matchAcceptLanguage falls back to a base-language match when no allowed tag matches the region.
func matchAcceptLanguage(header string, allowed []string) (string, bool) {
	type pref struct {
		tag string
		q   float64
	}
	prefs := []pref{}
	for _, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		tag = strings.TrimSpace(tag)
		if tag == "" || tag == "*" {
			continue
		}
		q := 1.0
		if _, raw, ok := strings.Cut(params, "q="); ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
				q = v
			}
		}
		if q > 0 {
			prefs = append(prefs, pref{tag: tag, q: q})
		}
	}
	sort.SliceStable(prefs, func(i, j int) bool { return prefs[i].q > prefs[j].q })

	for _, p := range prefs {
		for _, loc := range allowed {
			if strings.EqualFold(loc, p.tag) {
				return loc, true
			}
		}
		base, _, _ := strings.Cut(p.tag, "-")
		for _, loc := range allowed {
			locBase, _, _ := strings.Cut(loc, "-")
			if strings.EqualFold(locBase, base) {
				return loc, true
			}
		}
	}
	return "", false
}

// portalTimezone falls back to UTC when the configured zone is unset or unloadable.
func portalTimezone(app *App) *time.Location {
	if tz := app.setting.GetAppTimezone(); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.UTC
}

// portalStatusLabel maps internal status categories onto the three customer-facing labels and a CSS class.
func portalStatusLabel(lcl *i18n.I18n, category, lastMessageSender string) (string, string) {
	if category == smodels.CategoryResolved {
		return lcl.T("globals.terms.resolved"), "resolved"
	}
	if lastMessageSender == cmodels.SenderTypeAgent {
		return lcl.T("portal.statusAwaitingYourReply"), "waiting"
	}
	return lcl.T("portal.statusInProgress"), "open"
}

// portalTicketForm resolves the article's form override, else the portal default, else an empty subject-asking form.
func portalTicketForm(app *App, articleSlug string) pfmodels.Form {
	fallback := pfmodels.Form{AskSubject: true}
	consts := app.consts.Load().(*constants)

	formID := consts.PortalFormID
	if articleSlug != "" && consts.PortalHelpCenterID > 0 {
		if hc, err := app.helpcenter.GetHelpCenterByID(consts.PortalHelpCenterID); err == nil {
			if article, err := app.helpcenter.GetPublishedArticle(hc.Slug, articleSlug, hc.DefaultLocale); err == nil && article.PortalFormID.Valid {
				formID = article.PortalFormID.Int
			}
		}
	}
	if formID <= 0 {
		return fallback
	}
	form, err := app.portalForm.Get(formID)
	if err != nil {
		return fallback
	}
	return form
}

// portalFieldViews carries the submitted answers back into a re-rendered form.
func portalFieldViews(r *fastglue.Request, form pfmodels.Form, submitted bool) []portalFieldView {
	views := make([]portalFieldView, 0, len(form.Fields))
	for _, f := range form.Fields {
		view := portalFieldView{Field: f}
		if submitted {
			raw := strings.TrimSpace(string(r.RequestCtx.FormValue("field_" + f.Key)))
			if f.Type == pfmodels.FieldTypeCheckbox {
				view.Checked = raw != ""
			} else {
				view.Value = raw
			}
		}
		views = append(views, view)
	}
	return views
}

// portalFormAnswers returns the i18n key and label of the first answer it rejects.
func portalFormAnswers(r *fastglue.Request, form pfmodels.Form) (map[string]any, [][2]string, string, string) {
	var (
		attrs = map[string]any{}
		lines = [][2]string{}
	)
	for _, f := range form.Fields {
		raw := strings.TrimSpace(string(r.RequestCtx.FormValue("field_" + f.Key)))
		if f.Type == pfmodels.FieldTypeCheckbox {
			if raw == "" {
				continue
			}
			if f.Target == pfmodels.TargetAttribute {
				attrs[f.AttributeKey] = true
			} else {
				lines = append(lines, [2]string{f.Label, "yes"})
			}
			continue
		}
		if raw == "" {
			if f.Required {
				return nil, nil, "globals.messages.required", f.Label
			}
			continue
		}
		if utf8.RuneCountInString(raw) > portalMaxFieldLength || len(f.Options) > 0 && !slices.Contains(f.Options, raw) {
			return nil, nil, "portal.invalidField", f.Label
		}
		if f.Target != pfmodels.TargetAttribute {
			lines = append(lines, [2]string{f.Label, raw})
			continue
		}
		if f.Type == pfmodels.FieldTypeNumber {
			n, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, nil, "portal.invalidField", f.Label
			}
			attrs[f.AttributeKey] = n
			continue
		}
		attrs[f.AttributeKey] = raw
	}
	if len(attrs) == 0 {
		attrs = nil
	}
	return attrs, lines, "", ""
}

// portalSessionVia labels how the current session signed in.
func portalSessionVia(app *App, r *fastglue.Request, lcl *i18n.I18n) string {
	token := string(r.RequestCtx.Request.Header.Cookie(portalSessionCookie))
	if token == "" {
		return ""
	}
	via, _ := app.redis.HGet(context.Background(), portalSessionPrefix+token, "via").Result()
	switch via {
	case portalViaCode:
		return lcl.T("portal.signedInViaCode")
	case portalViaOIDC:
		return lcl.T("portal.signedInViaSSO")
	}
	return ""
}

// portalArticleSubject is empty when the slug names no published article.
func portalArticleSubject(app *App, articleSlug string) string {
	consts := app.consts.Load().(*constants)
	if articleSlug == "" || consts.PortalHelpCenterID <= 0 {
		return ""
	}
	hc, err := app.helpcenter.GetHelpCenterByID(consts.PortalHelpCenterID)
	if err != nil || !hc.IsActive {
		return ""
	}
	article, err := app.helpcenter.GetPublishedArticle(hc.Slug, articleSlug, hc.DefaultLocale)
	if err != nil {
		return ""
	}
	return article.Title
}

// portalCSATWidgetURL is empty when no survey was sent for the conversation.
func portalCSATWidgetURL(app *App, conversationID int) string {
	csat, err := app.csat.GetByConversationID(conversationID)
	if err != nil {
		return ""
	}
	return app.csat.MakePublicURL(app.consts.Load().(*constants).AppBaseURL, csat.UUID) + "/widget"
}

// portalReplyLabel counts the replies after the contact's opening message.
func portalReplyLabel(lcl *i18n.I18n, count int) string {
	if count <= 0 {
		return lcl.T("portal.noReplies")
	}
	// Tc picks the plural form but does not substitute, Ts substitutes but always picks the singular.
	return strings.ReplaceAll(lcl.Tc("portal.replyCount", count), "{count}", strconv.Itoa(count))
}

// portalSubject falls back to a placeholder for subjectless (livechat) conversations.
func portalSubject(lcl *i18n.I18n, subject string) string {
	if strings.TrimSpace(subject) == "" {
		return lcl.T("portal.noSubject")
	}
	return subject
}

// portalCanGroup mirrors the agent app's message grouping: same sender within the same 60-second bucket.
func portalCanGroup(a, b cmodels.Message) bool {
	if a.SenderType != b.SenderType || a.SenderID == 0 || a.SenderID != b.SenderID {
		return false
	}
	return a.CreatedAt.Unix()/60 == b.CreatedAt.Unix()/60
}

func portalContainsQuoteMarkers(html string) bool {
	for _, marker := range portalQuoteMarkers {
		if strings.Contains(html, marker) {
			return true
		}
	}
	return false
}

// portalAuthorName hides agent identity down to their name and labels the contact's own messages.
func portalAuthorName(lcl *i18n.I18n, msg cmodels.Message) string {
	if msg.SenderType == cmodels.SenderTypeContact {
		return lcl.T("globals.terms.you")
	}
	name := strings.TrimSpace(msg.Author.FirstName + " " + msg.Author.LastName)
	if name == "" {
		return lcl.T("portal.supportTeam")
	}
	return name
}

func portalContactFirstName(email string) string {
	local, _, _ := strings.Cut(email, "@")
	if local == "" {
		return email
	}
	return local
}

func portalInboxID(r *fastglue.Request) int {
	return r.Context.(*App).consts.Load().(*constants).PortalInboxID
}

// portalLivechatInbox reports false when the inbox is unset, disabled or not a livechat channel.
func portalLivechatInbox(app *App) (imodels.Inbox, bool) {
	id := app.consts.Load().(*constants).PortalLivechatInboxID
	if id <= 0 {
		return imodels.Inbox{}, false
	}
	inboxRecord, err := app.inbox.GetDBRecord(id)
	if err != nil || !inboxRecord.Enabled || inboxRecord.Channel != inbox.ChannelLiveChat {
		return imodels.Inbox{}, false
	}
	return inboxRecord, true
}

// portalLivechatInboxUUID is empty when the inbox is unset, disabled or not a livechat channel.
func portalLivechatInboxUUID(app *App) string {
	inboxRecord, ok := portalLivechatInbox(app)
	if !ok {
		return ""
	}
	return inboxRecord.UUID
}

// portalWidgetSessionToken reuses the contact's existing widget session when one is still valid.
func portalWidgetSessionToken(app *App, contactID int) string {
	inboxRecord, ok := portalLivechatInbox(app)
	if !ok {
		return ""
	}
	var config livechat.Config
	if err := json.Unmarshal(inboxRecord.Config, &config); err != nil {
		app.lo.Error("error parsing linked livechat inbox config", "inbox_id", inboxRecord.ID, "error", err)
		return ""
	}

	var (
		ctx        = context.Background()
		reverseKey = widgetUserKey(inboxRecord.ID, contactID)
		ttl        = getSessionDuration(config)
	)
	if token, err := app.redis.Get(ctx, reverseKey).Result(); err == nil && token != "" {
		if _, err := loadSession(app, token, config); err == nil {
			app.redis.Expire(ctx, reverseKey, ttl)
			return token
		}
	}

	u, err := app.user.Get(contactID, "", []string{umodels.UserTypeContact})
	if err != nil {
		return ""
	}
	token, err := generateSessionToken(app, contactID, inboxRecord.ID, false, u.ExternalUserID.String, ttl)
	if err != nil {
		app.lo.Error("error creating widget session for portal contact", "contact_id", contactID, "error", err)
		return ""
	}
	app.redis.Set(ctx, reverseKey, token, ttl)
	return token
}

func deletePortalWidgetSession(app *App, contactID int) {
	inboxRecord, ok := portalLivechatInbox(app)
	if !ok {
		return
	}
	var (
		ctx = context.Background()
		key = widgetUserKey(inboxRecord.ID, contactID)
	)
	if token, err := app.redis.Get(ctx, key).Result(); err == nil && token != "" {
		deleteSessionToken(app, token)
	}
	app.redis.Del(ctx, key)
}

// portalReturnTo keeps a redirect target only while it resolves to a page inside the portal.
func portalReturnTo(baseURL, raw string) string {
	raw = strings.TrimSpace(raw)
	if base := strings.TrimSuffix(baseURL, "/"); base != "" {
		raw = strings.TrimPrefix(raw, base)
	}
	if strings.ContainsAny(raw, "\\\r\n\t") {
		return "/portal"
	}
	if raw != "/portal" && !strings.HasPrefix(raw, "/portal/") && !strings.HasPrefix(raw, "/portal?") {
		return "/portal"
	}
	return raw
}

func portalReturnPath(r *fastglue.Request, raw string) string {
	return portalReturnTo(r.Context.(*App).consts.Load().(*constants).AppBaseURL, raw)
}

func normalizePortalEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func redirectPortalLogin(r *fastglue.Request) error {
	return r.RedirectURI("/portal/login", fasthttp.StatusSeeOther, nil, "")
}

// redirectPortalLoginReturn tags the login redirect with the page the visitor was aiming for.
func redirectPortalLoginReturn(r *fastglue.Request) error {
	target := portalReturnPath(r, string(r.RequestCtx.URI().RequestURI()))
	if !r.RequestCtx.IsGet() || target == "/portal" {
		return redirectPortalLogin(r)
	}
	return r.RedirectURI("/portal/login", fasthttp.StatusSeeOther, map[string]any{"return": target}, "")
}

func redirectPortalLoginError(r *fastglue.Request, code string) error {
	return r.RedirectURI("/portal/login", fasthttp.StatusSeeOther, map[string]any{"error": code}, "")
}

// createPortalSession stores a new session token in Redis and returns it.
func createPortalSession(app *App, userID int, via string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating portal session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	csrf, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		return "", fmt.Errorf("generating portal csrf token: %w", err)
	}

	ctx := context.Background()
	pipe := app.redis.Pipeline()
	pipe.HSet(ctx, portalSessionPrefix+token, map[string]any{"user_id": strconv.Itoa(userID), "csrf": csrf, "via": via})
	pipe.Expire(ctx, portalSessionPrefix+token, portalSessionTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("storing portal session: %w", err)
	}
	return token, nil
}

// loadPortalSession returns the session's contact ID and CSRF token, sliding the TTL.
func loadPortalSession(app *App, token string) (int, string, error) {
	if token == "" {
		return 0, "", fmt.Errorf("no portal session token")
	}
	ctx := context.Background()
	data, err := app.redis.HGetAll(ctx, portalSessionPrefix+token).Result()
	if err != nil {
		return 0, "", fmt.Errorf("looking up portal session: %w", err)
	}
	userID, _ := strconv.Atoi(data["user_id"])
	if userID <= 0 {
		return 0, "", fmt.Errorf("portal session not found or expired")
	}
	app.redis.Expire(ctx, portalSessionPrefix+token, portalSessionTTL)
	return userID, data["csrf"], nil
}

func deletePortalSession(app *App, token string) {
	if token == "" {
		return
	}
	app.redis.Del(context.Background(), portalSessionPrefix+token)
}

// setPortalCookie sets an HttpOnly, SameSite=Lax cookie scoped to path.
func setPortalCookie(r *fastglue.Request, name, value, path string, maxAge time.Duration) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKey(name)
	c.SetValue(value)
	c.SetPath(path)
	c.SetHTTPOnly(true)
	c.SetSecure(!ko.Bool("app.server.disable_secure_cookies"))
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	c.SetMaxAge(int(maxAge.Seconds()))
	r.RequestCtx.Response.Header.Add("Set-Cookie", c.String())
}

func clearPortalSessionCookie(r *fastglue.Request) {
	clearPortalCookie(r, portalSessionCookie, "/")
	clearPortalCookie(r, portalSessionCookie, "/portal")
}

func clearPortalCookie(r *fastglue.Request, name, path string) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKey(name)
	c.SetValue("")
	c.SetPath(path)
	c.SetHTTPOnly(true)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	c.SetMaxAge(-1)
	r.RequestCtx.Response.Header.Add("Set-Cookie", c.String())
}
