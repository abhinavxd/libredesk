package main

import (
	"html"
	"net/url"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
)

func loadResourcePolicy(app *App) resourcepolicy.Policy {
	cfg, err := app.setting.GetResourcePolicy()
	if err != nil {
		app.lo.Error("error reading resource policy, blocking external resources", "error", err)
		return resourcepolicy.Policy{}
	}
	p, err := resourcepolicy.New(cfg)
	if err != nil {
		app.lo.Error("invalid resource policy, blocking external resources", "error", err)
	}
	return p
}

func prepareMessageDisplay(app *App, msg *cmodels.Message, rootURL string, policy resourcepolicy.Policy, userID int) {
	original := *msg
	sources := messageImageSources(original)
	allowed, senderTrusted := false, false
	if policy.PermitsConsent() && len(sources) > 0 {
		allowed, senderTrusted = imagePermission(app, userID, original)
	}
	app.conversation.SignAttachmentURLs(msg.Attachments)
	trusted := resolveQuotedCIDs(app, msg)
	resolveAttachmentCIDs(msg, rootURL)
	for _, att := range msg.Attachments {
		if isDisplayImage(att.ContentType) {
			trusted = append(trusted, att.URL)
		}
	}

	if msg.ContentType != cmodels.ContentTypeHTML {
		setMessageDisplay(msg, policy, trusted)
		return
	}
	pendingPermission := false
	display := policy.PrepareDisplayWithImages(msg.Content, trusted, func(source string) string {
		id := imageSourceID(source)
		if sources[id] == "" {
			return ""
		}
		if !allowed && !policy.AllowsURL(source) {
			pendingPermission = true
			return ""
		}
		return "/api/v1/conversations/" + url.PathEscape(msg.ConversationUUID) + "/messages/" + url.PathEscape(msg.UUID) + "/images/" + id
	})
	display.CanAllow = policy.PermitsConsent() && pendingPermission
	display.Sender = imageSender(original)
	display.SenderTrusted = senderTrusted
	msg.Display = &display
}

func setMessageDisplay(msg *cmodels.Message, policy resourcepolicy.Policy, trusted []string) {
	if msg.ContentType != cmodels.ContentTypeHTML {
		msg.Display = &resourcepolicy.Display{HTML: html.EscapeString(msg.Content), BlockedDomains: []string{}}
		return
	}
	display := policy.PrepareDisplay(msg.Content, trusted)
	msg.Display = &display
}

func isDisplayImage(contentType string) bool {
	return resourcepolicy.IsDisplayImage(contentType)
}
