package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"html"
	"html/template"
	"net/url"
	"strconv"
	"strings"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	portalLoginTTL      = 20 * time.Minute
	portalSessionTTL    = 7 * 24 * time.Hour
	portalCookieName    = "portal_session"
	portalLoginPrefix   = "portal:login:"
	portalSessionPrefix = "portal:sess:"
)

func handlePortalLogin(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req struct {
		Email string `json:"email"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendEnvelope(true)
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !stringutil.ValidEmail(email) {
		return r.SendEnvelope(true)
	}
	contact, err := app.user.GetContactByEmail(email)
	if err != nil {
		return r.SendEnvelope(true)
	}
	token, err := randomToken()
	if err != nil {
		return r.SendEnvelope(true)
	}
	if err := app.redis.Set(context.Background(), portalLoginPrefix+token, strconv.Itoa(contact.ID), portalLoginTTL).Err(); err != nil {
		app.lo.Error("error storing portal login token", "error", err)
		return r.SendEnvelope(true)
	}
	root, _ := app.setting.GetAppRootURL()
	link := strings.TrimRight(root, "/") + "/portal?token=" + url.QueryEscape(token)
	_ = app.notifier.Send(notifier.Message{
		RecipientEmails: []string{email},
		Subject:         app.i18n.T("portal.loginEmailSubject"),
		Content:         app.i18n.Ts("portal.loginEmailBody", "link", html.EscapeString(link)),
		Provider:        notifier.ProviderEmail,
	})
	return r.SendEnvelope(true)
}

func handlePortalHome(r *fastglue.Request) error {
	app := r.Context.(*App)
	if token := string(r.RequestCtx.QueryArgs().Peek("token")); token != "" {
		return consumePortalToken(r, token)
	}
	contactID := portalContactID(r)
	if contactID == 0 {
		return renderPortalLogin(r)
	}
	contact, err := app.user.GetContactOrVisitor(contactID, "")
	if err != nil {
		clearPortalCookie(r)
		return renderPortalLogin(r)
	}
	convs, err := app.conversation.GetContactPreviousConversations(contactID, 200)
	if err != nil {
		convs = nil
	}
	return app.tmpl.RenderWebPage(r.RequestCtx, "portal", map[string]any{
		"Data": map[string]any{
			"Title":         app.i18n.T("portal.title"),
			"ContactName":   strings.TrimSpace(contact.FirstName + " " + contact.LastName),
			"Conversations": convs,
		},
	})
}

func handlePortalConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	contactID := portalContactID(r)
	if contactID == 0 {
		return renderPortalLogin(r)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	conv, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil || conv.ContactID != contactID {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]any{
			"Data": map[string]any{"ErrorMessage": app.i18n.T("globals.messages.pageNotFound")},
		})
	}
	private := false
	msgs, _, err := app.conversation.GetConversationMessages(uuid, 1, 200, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing})
	if err != nil {
		msgs = nil
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	viewMsgs := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		viewMsgs = append(viewMsgs, map[string]any{
			"Author":  msg.Author,
			"Content": template.HTML(msg.Content),
		})
	}
	return app.tmpl.RenderWebPage(r.RequestCtx, "portal-conversation", map[string]any{
		"Data": map[string]any{
			"Title":        conv.Subject.String,
			"Conversation": conv,
			"Messages":     viewMsgs,
		},
	})
}

func handlePortalReply(r *fastglue.Request) error {
	app := r.Context.(*App)
	contactID := portalContactID(r)
	if contactID == 0 {
		return renderPortalLogin(r)
	}
	uuid := r.RequestCtx.UserValue("uuid").(string)
	conv, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil || conv.ContactID != contactID {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.pageNotFound"), nil, envelope.NotFoundError)
	}
	content := strings.TrimSpace(string(r.RequestCtx.FormValue("content")))
	if content == "" {
		r.RequestCtx.Redirect("/portal/conversations/"+uuid, fasthttp.StatusSeeOther)
		return nil
	}
	if _, err := app.conversation.CreateContactMessage(nil, contactID, uuid, html.EscapeString(content), cmodels.ContentTypeHTML, false, ""); err != nil {
		app.lo.Error("error creating portal reply", "error", err)
	}
	r.RequestCtx.Redirect("/portal/conversations/"+uuid, fasthttp.StatusSeeOther)
	return nil
}

func handlePortalLogout(r *fastglue.Request) error {
	if tok := string(r.RequestCtx.Request.Header.Cookie(portalCookieName)); tok != "" {
		app := r.Context.(*App)
		app.redis.Del(context.Background(), portalSessionPrefix+tok)
	}
	clearPortalCookie(r)
	r.RequestCtx.Redirect("/portal", fasthttp.StatusSeeOther)
	return nil
}

func consumePortalToken(r *fastglue.Request, token string) error {
	app := r.Context.(*App)
	val, err := app.redis.Get(context.Background(), portalLoginPrefix+token).Result()
	if err != nil || val == "" {
		return renderPortalLogin(r)
	}
	app.redis.Del(context.Background(), portalLoginPrefix+token)
	sess, err := randomToken()
	if err != nil {
		return renderPortalLogin(r)
	}
	if err := app.redis.Set(context.Background(), portalSessionPrefix+sess, val, portalSessionTTL).Err(); err != nil {
		return renderPortalLogin(r)
	}
	setPortalCookie(r, sess)
	r.RequestCtx.Redirect("/portal", fasthttp.StatusSeeOther)
	return nil
}

func portalContactID(r *fastglue.Request) int {
	app := r.Context.(*App)
	tok := string(r.RequestCtx.Request.Header.Cookie(portalCookieName))
	if tok == "" {
		return 0
	}
	val, err := app.redis.Get(context.Background(), portalSessionPrefix+tok).Result()
	if err != nil || val == "" {
		return 0
	}
	id, _ := strconv.Atoi(val)
	return id
}

func renderPortalLogin(r *fastglue.Request) error {
	app := r.Context.(*App)
	return app.tmpl.RenderWebPage(r.RequestCtx, "portal-login", map[string]any{
		"Data": map[string]any{"Title": app.i18n.T("portal.title")},
	})
}

func setPortalCookie(r *fastglue.Request, value string) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKey(portalCookieName)
	c.SetValue(value)
	c.SetPath("/portal")
	c.SetHTTPOnly(true)
	c.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	c.SetMaxAge(int(portalSessionTTL.Seconds()))
	if r.RequestCtx.IsTLS() {
		c.SetSecure(true)
	}
	r.RequestCtx.Response.Header.SetCookie(c)
}

func clearPortalCookie(r *fastglue.Request) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKey(portalCookieName)
	c.SetValue("")
	c.SetPath("/portal")
	c.SetHTTPOnly(true)
	c.SetExpire(time.Unix(0, 0))
	r.RequestCtx.Response.Header.SetCookie(c)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
