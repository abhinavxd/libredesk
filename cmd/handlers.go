package main

import (
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/ws"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	maxPageSize = 500
	maxIDsParam = 200
)

// initHandlers initializes the HTTP routes and handlers for the application.
func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	// Authentication.
	g.POST("/api/v1/auth/login", rateLimit(handleLogin, "auth"))
	g.GET("/logout", auth(handleLogout))
	g.GET("/api/v1/oidc/{id}/login", rateLimit(handleOIDCLogin, "auth"))
	g.GET("/api/v1/oidc/{id}/finish", rateLimit(handleOIDCCallback, "auth"))

	// i18n.
	g.GET("/api/v1/lang", handleGetAvailableLanguages)
	g.GET("/api/v1/lang/{lang}", handleGetI18nLang)

	// Public config for app initialization.
	g.GET("/api/v1/config", handleGetConfig)

	// Media - supports both authenticated access and signed URLs.
	g.GET("/uploads/{uuid}", authOrSignedURL(handleServeMedia))
	g.GET("/api/v1/resource-images/avatar", auth(rateLimit(handleResourceAvatar, "media")))
	g.POST("/api/v1/media", auth(handleMediaUpload))

	// Settings.
	g.GET("/api/v1/settings/general", auth(handleGetGeneralSettings))
	g.GET("/api/v1/settings/resource-policy", auth(handleGetResourcePolicy))
	g.PUT("/api/v1/settings/resource-policy", perm(handleUpdateResourcePolicy, "general_settings:manage"))
	g.PUT("/api/v1/settings/general", perm(handleUpdateGeneralSettings, "general_settings:manage"))
	g.POST("/api/v1/settings/general/logo", perm(handleUploadSiteLogo, "general_settings:manage"))
	g.GET("/api/v1/system/resource-usage", perm(handleGetResourceUsage, "general_settings:manage"))
	g.GET("/api/v1/system/resource-limits", perm(handleGetResourceLimits, "general_settings:manage"))
	g.PUT("/api/v1/system/resource-limits", perm(handleUpdateResourceLimits, "general_settings:manage"))

	// OpenID connect single sign-on.
	g.GET("/api/v1/oidc", perm(handleGetAllOIDC, "oidc:manage"))
	g.POST("/api/v1/oidc", perm(handleCreateOIDC, "oidc:manage"))
	g.GET("/api/v1/oidc/{id}", perm(handleGetOIDC, "oidc:manage"))
	g.PUT("/api/v1/oidc/{id}", perm(handleUpdateOIDC, "oidc:manage"))
	g.DELETE("/api/v1/oidc/{id}", perm(handleDeleteOIDC, "oidc:manage"))

	// Conversations. Fernmail is address-first: no global inbox or saved views.
	// New emails start from an address through /api/v1/compose.
	g.GET("/api/v1/addresses", perm(handleGetAddresses, "conversations:read"))
	g.GET("/api/v1/addresses/{id}/conversations", perm(handleGetAddressConversations, "conversations:read"))
	g.POST("/api/v1/addresses/{id}/mark-read", perm(handleMarkAddressRead, "conversations:read"))
	g.GET("/api/v1/conversations/sidebar-counts", perm(handleGetSidebarCounts, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}/participants", perm(handleGetConversationParticipants, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/mark-unread", perm(handleMarkConversationAsUnread, "conversations:read"))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleGetMessage, "messages:read"))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}/images/{image}", perm(rateLimit(handleResourceImage, "media"), "messages:read"))
	g.POST("/api/v1/conversations/{cuuid}/messages/{uuid}/images/allow/{scope}", perm(handleAllowResourceImages, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/messages", perm(handleGetMessages, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/transcript", perm(handleDownloadConversationTranscript, "messages:read"))
	g.POST("/api/v1/conversations/{cuuid}/messages", auth(handleSendMessage))
	g.PUT("/api/v1/conversations/{cuuid}/messages/{uuid}/retry", perm(handleRetryMessage, "messages:write"))
	g.DELETE("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleDeleteMessage, "messages:write_private"))
	// Compose New and the review queue. messages:write sends directly;
	// reviews:submit holds the email until an Admin or Agent approves it.
	g.POST("/api/v1/compose", perm(handleCompose, "conversations:create"))
	g.POST("/api/v1/conversations/{uuid}/reviews", perm(handleSubmitReplyReview, "reviews:submit"))
	g.GET("/api/v1/conversations/{uuid}/reviews", perm(handleGetConversationReviews, "conversations:read"))
	g.GET("/api/v1/reviews", auth(handleGetReviews))
	g.GET("/api/v1/reviews/counts", auth(handleGetReviewCounts))
	g.GET("/api/v1/reviews/{uuid}", auth(handleGetReview))
	g.PUT("/api/v1/reviews/{uuid}", perm(handleResubmitReview, "reviews:submit"))
	g.DELETE("/api/v1/reviews/{uuid}", auth(handleDiscardReview))
	g.POST("/api/v1/reviews/{uuid}/approve", perm(handleApproveReview, "reviews:manage"))
	g.POST("/api/v1/reviews/{uuid}/deny", perm(handleDenyReview, "reviews:manage"))
	g.POST("/api/v1/reviews/{uuid}/withdraw", auth(handleWithdrawReview))
	g.POST("/api/v1/reviews/{uuid}/dismiss", auth(handleDismissReview))
	// Key Distribution: Admin-only pools, plus the composer's key card.
	g.GET("/api/v1/activation-keys/settings", perm(handleGetKeyDistributionSettings, "activation_keys:manage"))
	g.PUT("/api/v1/activation-keys/settings", perm(handleUpdateKeyDistributionSettings, "activation_keys:manage"))
	g.GET("/api/v1/activation-keys/apps", perm(handleGetKeyApps, "activation_keys:manage"))
	g.POST("/api/v1/activation-keys/apps", perm(handleCreateKeyApp, "activation_keys:manage"))
	g.PUT("/api/v1/activation-keys/apps/{id}", perm(handleUpdateKeyApp, "activation_keys:manage"))
	g.GET("/api/v1/activation-keys/apps/{id}/keys", perm(handleGetKeys, "activation_keys:manage"))
	g.POST("/api/v1/activation-keys/apps/{id}/keys", perm(handleImportKeys, "activation_keys:manage"))
	g.POST("/api/v1/activation-keys/keys/{id}/reveal", perm(handleRevealKey, "activation_keys:manage"))
	g.POST("/api/v1/activation-keys/keys/{id}/void", perm(handleVoidKey, "activation_keys:manage"))
	g.GET("/api/v1/activation-keys/composer", auth(handleGetKeyComposer))
	g.PUT("/api/v1/activation-keys/composer/last-app", auth(handleSetKeyComposerApp))
	// Draft endpoints
	g.GET("/api/v1/drafts", auth(handleGetAllDrafts))
	g.POST("/api/v1/conversations/{uuid}/draft", auth(handleUpsertConversationDraft))
	g.DELETE("/api/v1/conversations/{uuid}/draft", auth(handleDeleteConversationDraft))

	// Agents.
	g.GET("/api/v1/agents/me", auth(handleGetCurrentAgent))
	g.PUT("/api/v1/agents/me/availability", auth(handleUpdateAgentAvailability))

	g.GET("/api/v1/agents/compact", auth(handleGetAgentsCompact))
	g.GET("/api/v1/agents", perm(handleGetAgents, "users:manage"))
	g.GET("/api/v1/agents/{id}", perm(handleGetAgent, "users:manage"))
	g.POST("/api/v1/agents", perm(handleCreateAgent, "users:manage"))
	g.PUT("/api/v1/agents/{id}", perm(handleUpdateAgent, "users:manage"))
	g.DELETE("/api/v1/agents/{id}", perm(handleDeleteAgent, "users:manage"))
	g.POST("/api/v1/agents/import", perm(handleImportAgents, "users:manage"))
	g.GET("/api/v1/agents/import/status", perm(handleGetAgentImportStatus, "users:manage"))
	g.POST("/api/v1/agents/{id}/api-key", perm(handleGenerateAPIKey, "users:manage"))
	g.DELETE("/api/v1/agents/{id}/api-key", perm(handleRevokeAPIKey, "users:manage"))
	// Users settings screen: role, addresses and enabled state in one place.
	g.GET("/api/v1/admin/users", perm(handleGetManagedUsers, "users:manage"))
	g.POST("/api/v1/admin/users", perm(handleCreateManagedUser, "users:manage"))
	g.PUT("/api/v1/admin/users/{id}/access", perm(handleUpdateManagedUserAccess, "users:manage"))
	g.DELETE("/api/v1/admin/users/{id}", perm(handleDeleteManagedUser, "users:manage"))
	g.POST("/api/v1/agents/reset-password", rateLimit(tryAuth(handleResetPassword), "auth"))
	g.POST("/api/v1/agents/set-password", rateLimit(tryAuth(handleSetPassword), "auth"))

	// Inboxes.
	// These are transport-level credentials and lifecycle controls. The primary
	// product configuration lives under /api/v1/admin/addresses.
	g.GET("/api/v1/admin/addresses", perm(handleGetAdminAddresses, "inboxes:manage"))
	g.GET("/api/v1/admin/address-principals", perm(handleGetAddressPrincipals, "inboxes:manage"))
	g.GET("/api/v1/admin/addresses/{id}", perm(handleGetAdminAddress, "inboxes:manage"))
	g.GET("/api/v1/admin/addresses/{id}/access", perm(handleGetAddressAccess, "inboxes:manage"))
	g.POST("/api/v1/admin/addresses", perm(handleCreateAddress, "inboxes:manage"))
	g.PUT("/api/v1/admin/addresses/{id}", perm(handleUpdateAddress, "inboxes:manage"))
	g.DELETE("/api/v1/admin/addresses/{id}", perm(handleDeleteAddress, "inboxes:manage"))
	g.GET("/api/v1/inboxes", perm(handleGetInboxes, "inboxes:manage"))
	g.GET("/api/v1/inboxes/{id}", perm(handleGetInbox, "inboxes:manage"))
	g.POST("/api/v1/inboxes", perm(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}", perm(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/v1/inboxes/{id}", perm(handleDeleteInbox, "inboxes:manage"))

	// OAuth endpoints for email inboxes.
	g.POST("/api/v1/inboxes/oauth/{provider}/authorize", perm(handleOAuthAuthorize, "inboxes:manage"))
	g.GET("/api/v1/inboxes/oauth/{provider}/callback", perm(handleOAuthCallback, "inboxes:manage"))

	// Webhooks.
	g.GET("/api/v1/webhooks/compact", auth(handleGetWebhooksCompact))
	g.GET("/api/v1/webhooks", perm(handleGetWebhooks, "webhooks:manage"))
	g.GET("/api/v1/webhooks/{id}", perm(handleGetWebhook, "webhooks:manage"))
	g.POST("/api/v1/webhooks", perm(handleCreateWebhook, "webhooks:manage"))
	g.PUT("/api/v1/webhooks/{id}", perm(handleUpdateWebhook, "webhooks:manage"))
	g.DELETE("/api/v1/webhooks/{id}", perm(handleDeleteWebhook, "webhooks:manage"))
	g.PUT("/api/v1/webhooks/{id}/toggle", perm(handleToggleWebhook, "webhooks:manage"))
	g.POST("/api/v1/webhooks/{id}/test", perm(handleTestWebhook, "webhooks:manage"))

	// Templates.
	g.GET("/api/v1/templates", perm(handleGetTemplates, "templates:manage"))
	g.GET("/api/v1/templates/{id}", perm(handleGetTemplate, "templates:manage"))
	g.POST("/api/v1/templates", perm(handleCreateTemplate, "templates:manage"))
	g.PUT("/api/v1/templates/{id}", perm(handleUpdateTemplate, "templates:manage"))
	g.DELETE("/api/v1/templates/{id}", perm(handleDeleteTemplate, "templates:manage"))

	// WebSocket.
	g.GET("/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// getAndHead registers both methods: uptime checkers and link validators probe with HEAD.
	getAndHead := func(path string, h fastglue.FastRequestHandler) {
		g.GET(path, h)
		g.HEAD(path, h)
	}

	// Frontend pages.
	getAndHead("/", notAuthPage(serveIndexPage))
	g.GET("/addresses/{all:*}", authPage(serveIndexPage))
	g.GET("/reviews", authPage(serveIndexPage))
	g.GET("/reviews/{all:*}", authPage(serveIndexPage))
	g.GET("/conversation/{all:*}", authPage(serveIndexPage))
	// Keep legacy bookmarks client-routable while the frontend redirects them to
	// their first accessible address.
	g.GET("/inboxes/{all:*}", authPage(serveIndexPage))
	// Search is retired; old /search bookmarks are redirected by the frontend.
	g.GET("/search", authPage(serveIndexPage))
	g.GET("/admin/{all:*}", authPage(serveIndexPage))
	g.GET("/reset-password", notAuthPage(serveIndexPage))
	g.GET("/set-password", notAuthPage(serveIndexPage))

	// Assets and static files.
	// FIXME: Reduce the number of routes.
	g.GET("/assets/{all:*}", serveFrontendStaticFiles)
	g.GET("/images/{all:*}", serveFrontendStaticFiles)
	getAndHead("/favicon.svg", func(r *fastglue.Request) error {
		return serveMainFrontendFile(r, "favicon.svg", "image/svg+xml", "no-cache")
	})
	getAndHead("/favicon.ico", func(r *fastglue.Request) error {
		return serveMainFrontendFile(r, "favicon.ico", "image/x-icon", "no-cache")
	})
	g.GET("/manifest.webmanifest", serveManifest)
	g.GET("/sw.js", serveServiceWorker)
	g.GET("/static/public/{all:*}", serveStaticFiles)

	// Health check.
	g.GET("/health", handleHealthCheck)
}

// serveIndexPage serves the main index page of the application.
func serveIndexPage(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Prevent caching of the index page.
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")

	// Serve the index.html file from the embedded filesystem.
	file, err := app.fs.Get(path.Join(frontendDir, "index.html"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.Response.SetBodyRaw(file.ReadBytes())

	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return nil
}

// serveStaticFiles serves static assets from the filesystem.
func serveStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	filePath := string(r.RequestCtx.Path())

	file, err := app.fs.Get(filePath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	body := file.ReadBytes()
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400")
	r.RequestCtx.Response.SetBodyRaw(body)
	return nil
}

// serveFrontendStaticFiles serves static assets from the embedded filesystem.
func serveFrontendStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Get the requested file path.
	filePath := string(r.RequestCtx.Path())

	// Fetch and serve the file from the embedded filesystem.
	finalPath := filepath.Join(frontendDir, filePath)
	file, err := app.fs.Get(finalPath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	body := file.ReadBytes()
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.SetBodyRaw(body)
	return nil
}

func serveManifest(r *fastglue.Request) error {
	return serveMainFrontendFile(r, "manifest.webmanifest", "application/manifest+json", "no-cache")
}

func serveServiceWorker(r *fastglue.Request) error {
	r.RequestCtx.Response.Header.Set("Service-Worker-Allowed", "/")
	return serveMainFrontendFile(r, "sw.js", "application/javascript", "no-cache")
}

func serveMainFrontendFile(r *fastglue.Request, name, contentType, cacheControl string) error {
	app := r.Context.(*App)
	file, err := app.fs.Get(filepath.Join(frontendDir, name))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.Response.Header.Set("Cache-Control", cacheControl)
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// getIDsParam parses a comma separated list of positive IDs from a query param.
func getIDsParam(r *fastglue.Request, name string) []int {
	raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek(name)))
	if raw == "" {
		return nil
	}
	var ids []int
	for part := range strings.SplitSeq(raw, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
		if len(ids) == maxIDsParam {
			break
		}
	}
	return ids
}

// getPagination extracts page and page_size from query params with defaults.
func getPagination(r *fastglue.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// getOptionalPagination reads page/page_size, returning pageSize 0 (fetch everything) when absent.
func getOptionalPagination(r *fastglue.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
	pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	if page < 1 {
		page = 1
	}
	if pageSize < 0 {
		pageSize = 0
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// sendErrorEnvelope sends a standardized error response to the client.
func sendErrorEnvelope(r *fastglue.Request, err error) error {
	e, ok := err.(envelope.Error)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error interface conversion failed", nil, fastglue.ErrorType(envelope.GeneralError))
	}
	return r.SendErrorEnvelope(e.Code, e.Error(), e.Data, fastglue.ErrorType(e.ErrorType))
}

// handleHealthCheck handles the health check endpoint.
func handleHealthCheck(r *fastglue.Request) error {
	return r.SendEnvelope(true)
}
