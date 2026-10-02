package main

import (
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"testing"
)

func TestRetiredFeatureRoutesAreAbsent(t *testing.T) {
	g := fastglue.New()
	initHandlers(g, nil)
	for _, route := range []struct{ method, path string }{
		{"GET", "/account/profile"}, {"DELETE", "/api/v1/agents/me/avatar"},
		{"GET", "/api/v1/tags"}, {"POST", "/api/v1/tags"}, {"POST", "/api/v1/tags/import"}, {"POST", "/api/v1/conversations/123/tags"},
		{"GET", "/api/v1/notifications"}, {"GET", "/api/v1/notifications/preferences"},
		{"POST", "/api/v1/notifications/push-subscriptions"}, {"GET", "/api/v1/settings/notifications/email"},
		{"GET", "/api/v1/contacts"}, {"POST", "/api/v1/contacts"},
		{"GET", "/api/v1/priorities"}, {"GET", "/api/v1/automations"},
		{"GET", "/api/v1/macros"}, {"GET", "/api/v1/sla"},
		{"GET", "/api/v1/ai/assistants"}, {"GET", "/api/v1/help-centers"},
		{"GET", "/widget.js"}, {"GET", "/widget/ws"},
		{"PUT", "/api/v1/conversations/123/priority"},
		{"PUT", "/api/v1/conversations/123/assignee"},
		{"POST", "/api/v1/conversations"},
		{"GET", "/api/v1/views"}, {"POST", "/api/v1/views"},
		{"GET", "/api/v1/views/1"}, {"PUT", "/api/v1/views/1"}, {"DELETE", "/api/v1/views/1"},
		{"GET", "/api/v1/mailboxes"}, {"GET", "/api/v1/mailboxes/1/conversations"},
		{"GET", "/api/v1/inboxes/1/access"}, {"PUT", "/api/v1/inboxes/1/access"},
	} {
		handler, _ := g.Router.Lookup(route.method, route.path, &fasthttp.RequestCtx{})
		if handler != nil {
			t.Errorf("retired route still registered: %s %s", route.method, route.path)
		}
	}
	for _, path := range []string{"/api/v1/conversations/all", "/api/v1/inboxes", "/api/v1/search/messages"} {
		handler, _ := g.Router.Lookup("GET", path, &fasthttp.RequestCtx{})
		if handler == nil {
			t.Errorf("missing mailbox route: %s", path)
		}
	}
}
