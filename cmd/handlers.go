package main

import (
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// initHandlers initializes the HTTP routes and handlers for the application.
func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	// Authentication.
	g.POST("/api/v1/login", handleLogin)
	g.GET("/logout", handleLogout)
	g.GET("/api/v1/oidc/{id}/login", handleOIDCLogin)
	g.GET("/api/v1/oidc/{id}/finish", handleOIDCCallback)

	// Media.
	g.GET("/uploads/{uuid}", auth(handleServeMedia))
	g.POST("/api/v1/media", auth(handleMediaUpload))

	// Settings.
	g.GET("/api/v1/settings/general", handleGetGeneralSettings)
	g.PUT("/api/v1/settings/general", perm(handleUpdateGeneralSettings, "general_settings:manage"))
	g.GET("/api/v1/settings/notifications/email", perm(handleGetEmailNotificationSettings, "notification_settings:manage"))
	g.PUT("/api/v1/settings/notifications/email", perm(handleUpdateEmailNotificationSettings, "notification_settings:manage"))

	// OpenID connect single sign-on.
	g.GET("/api/v1/oidc/enabled", handleGetAllEnabledOIDC)
	g.GET("/api/v1/oidc", perm(handleGetAllOIDC, "oidc:manage"))
	g.POST("/api/v1/oidc", perm(handleCreateOIDC, "oidc:manage"))
	g.POST("/api/v1/oidc/test", perm(handleTestOIDC, "oidc:manage"))
	g.GET("/api/v1/oidc/{id}", perm(handleGetOIDC, "oidc:manage"))
	g.PUT("/api/v1/oidc/{id}", perm(handleUpdateOIDC, "oidc:manage"))
	g.DELETE("/api/v1/oidc/{id}", perm(handleDeleteOIDC, "oidc:manage"))

	// Conversations.
	g.GET("/api/v1/conversations/all", perm(handleGetAllConversations, "conversations:read_all"))
	g.GET("/api/v1/conversations/unassigned", perm(handleGetUnassignedConversations, "conversations:read_unassigned"))
	g.GET("/api/v1/conversations/assigned", perm(handleGetAssignedConversations, "conversations:read_assigned"))
	g.GET("/api/v1/teams/{id}/conversations/unassigned", perm(handleGetTeamUnassignedConversations, "conversations:read_team_inbox"))
	g.GET("/api/v1/views/{id}/conversations", perm(handleGetViewConversations, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}/participants", perm(handleGetConversationParticipants, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user", perm(handleUpdateUserAssignee, "conversations:update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee, "conversations:update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user/remove", perm(handleRemoveUserAssignee, "conversations:update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team/remove", perm(handleRemoveTeamAssignee, "conversations:update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/priority", perm(handleUpdateConversationPriority, "conversations:update_priority"))
	g.PUT("/api/v1/conversations/{uuid}/status", perm(handleUpdateConversationStatus, "conversations:update_status"))
	g.PUT("/api/v1/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen, "conversations:read"))
	g.POST("/api/v1/conversations/{uuid}/tags", perm(handleUpdateConversationtags, "conversations:update_tags"))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleGetMessage, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/messages", perm(handleGetMessages, "messages:read"))
	g.POST("/api/v1/conversations/{cuuid}/messages", perm(handleSendMessage, "messages:write"))
	g.PUT("/api/v1/conversations/{cuuid}/messages/{uuid}/retry", perm(handleRetryMessage, "messages:write"))
	g.POST("/api/v1/conversations", perm(handleCreateConversation, "conversations:write"))

	// Search.
	g.GET("/api/v1/conversations/search", perm(handleSearchConversations, "conversations:read"))
	g.GET("/api/v1/messages/search", perm(handleSearchMessages, "messages:read"))
	g.GET("/api/v1/contacts/search", perm(handleSearchContacts, "conversations:write"))

	// Views.
	g.GET("/api/v1/views/me", perm(handleGetUserViews, "view:manage"))
	g.POST("/api/v1/views/me", perm(handleCreateUserView, "view:manage"))
	g.PUT("/api/v1/views/me/{id}", perm(handleUpdateUserView, "view:manage"))
	g.DELETE("/api/v1/views/me/{id}", perm(handleDeleteUserView, "view:manage"))

	// Status and priority.
	g.GET("/api/v1/statuses", auth(handleGetStatuses))
	g.POST("/api/v1/statuses", perm(handleCreateStatus, "status:manage"))
	g.PUT("/api/v1/statuses/{id}", perm(handleUpdateStatus, "status:manage"))
	g.DELETE("/api/v1/statuses/{id}", perm(handleDeleteStatus, "status:manage"))
	g.GET("/api/v1/priorities", auth(handleGetPriorities))

	// Tag.
	g.GET("/api/v1/tags", auth(handleGetTags))
	g.POST("/api/v1/tags", perm(handleCreateTag, "tags:manage"))
	g.PUT("/api/v1/tags/{id}", perm(handleUpdateTag, "tags:manage"))
	g.DELETE("/api/v1/tags/{id}", perm(handleDeleteTag, "tags:manage"))

	// Macros.
	g.GET("/api/v1/macros", auth(handleGetMacros))
	g.GET("/api/v1/macros/{id}", perm(handleGetMacro, "macros:manage"))
	g.POST("/api/v1/macros", perm(handleCreateMacro, "macros:manage"))
	g.PUT("/api/v1/macros/{id}", perm(handleUpdateMacro, "macros:manage"))
	g.DELETE("/api/v1/macros/{id}", perm(handleDeleteMacro, "macros:manage"))
	g.POST("/api/v1/conversations/{uuid}/macros/{id}/apply", auth(handleApplyMacro))

	// User.
	g.GET("/api/v1/users/me", auth(handleGetCurrentUser))
	g.PUT("/api/v1/users/me", auth(handleUpdateCurrentUser))
	g.GET("/api/v1/users/me/teams", auth(handleGetCurrentUserTeams))
	g.PUT("/api/v1/users/me/availability", auth(handleUpdateUserAvailability))
	g.DELETE("/api/v1/users/me/avatar", auth(handleDeleteAvatar))
	g.GET("/api/v1/users/compact", auth(handleGetUsersCompact))
	g.GET("/api/v1/users", perm(handleGetUsers, "users:manage"))
	g.GET("/api/v1/users/{id}", perm(handleGetUser, "users:manage"))
	g.POST("/api/v1/users", perm(handleCreateUser, "users:manage"))
	g.PUT("/api/v1/users/{id}", perm(handleUpdateUser, "users:manage"))
	g.DELETE("/api/v1/users/{id}", perm(handleDeleteUser, "users:manage"))
	g.POST("/api/v1/users/reset-password", tryAuth(handleResetPassword))
	g.POST("/api/v1/users/set-password", tryAuth(handleSetPassword))

	// Team.
	g.GET("/api/v1/teams/compact", auth(handleGetTeamsCompact))
	g.GET("/api/v1/teams", perm(handleGetTeams, "teams:manage"))
	g.GET("/api/v1/teams/{id}", perm(handleGetTeam, "teams:manage"))
	g.POST("/api/v1/teams", perm(handleCreateTeam, "teams:manage"))
	g.PUT("/api/v1/teams/{id}", perm(handleUpdateTeam, "teams:manage"))
	g.DELETE("/api/v1/teams/{id}", perm(handleDeleteTeam, "teams:manage"))

	// i18n.
	g.GET("/api/v1/lang/{lang}", handleGetI18nLang)

	// Automation.
	g.GET("/api/v1/automation/rules", perm(handleGetAutomationRules, "automations:manage"))
	g.GET("/api/v1/automation/rules/{id}", perm(handleGetAutomationRule, "automations:manage"))
	g.POST("/api/v1/automation/rules", perm(handleCreateAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automation/rules/{id}/toggle", perm(handleToggleAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automation/rules/{id}", perm(handleUpdateAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automation/rules/weights", perm(handleUpdateAutomationRuleWeights, "automations:manage"))
	g.PUT("/api/v1/automation/rules/execution-mode", perm(handleUpdateAutomationRuleExecutionMode, "automations:manage"))
	g.DELETE("/api/v1/automation/rules/{id}", perm(handleDeleteAutomationRule, "automations:manage"))

	// Inbox.
	g.GET("/api/v1/inboxes", auth(handleGetInboxes))
	g.GET("/api/v1/inboxes/{id}", perm(handleGetInbox, "inboxes:manage"))
	g.POST("/api/v1/inboxes", perm(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}", perm(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/v1/inboxes/{id}", perm(handleDeleteInbox, "inboxes:manage"))

	// Role.
	g.GET("/api/v1/roles", perm(handleGetRoles, "roles:manage"))
	g.GET("/api/v1/roles/{id}", perm(handleGetRole, "roles:manage"))
	g.POST("/api/v1/roles", perm(handleCreateRole, "roles:manage"))
	g.PUT("/api/v1/roles/{id}", perm(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/v1/roles/{id}", perm(handleDeleteRole, "roles:manage"))

	// Dashboard.
	g.GET("/api/v1/reports/overview/counts", perm(handleDashboardCounts, "reports:manage"))
	g.GET("/api/v1/reports/overview/charts", perm(handleDashboardCharts, "reports:manage"))

	// Template.
	g.GET("/api/v1/templates", perm(handleGetTemplates, "templates:manage"))
	g.GET("/api/v1/templates/{id}", perm(handleGetTemplate, "templates:manage"))
	g.POST("/api/v1/templates", perm(handleCreateTemplate, "templates:manage"))
	g.PUT("/api/v1/templates/{id}", perm(handleUpdateTemplate, "templates:manage"))
	g.DELETE("/api/v1/templates/{id}", perm(handleDeleteTemplate, "templates:manage"))

	// Business hours.
	g.GET("/api/v1/business-hours", perm(handleGetBusinessHours, "business_hours:manage"))
	g.GET("/api/v1/business-hours/{id}", perm(handleGetBusinessHour, "business_hours:manage"))
	g.POST("/api/v1/business-hours", perm(handleCreateBusinessHours, "business_hours:manage"))
	g.PUT("/api/v1/business-hours/{id}", perm(handleUpdateBusinessHours, "business_hours:manage"))
	g.DELETE("/api/v1/business-hours/{id}", perm(handleDeleteBusinessHour, "business_hours:manage"))

	// SLA.
	g.GET("/api/v1/sla", perm(handleGetSLAs, "sla:manage"))
	g.GET("/api/v1/sla/{id}", perm(handleGetSLA, "sla:manage"))
	g.POST("/api/v1/sla", perm(handleCreateSLA, "sla:manage"))
	g.PUT("/api/v1/sla/{id}", perm(handleUpdateSLA, "sla:manage"))
	g.DELETE("/api/v1/sla/{id}", perm(handleDeleteSLA, "sla:manage"))

	// AI completion.
	g.GET("/api/v1/ai/prompts", auth(handleGetAIPrompts))
	g.POST("/api/v1/ai/completion", auth(handleAICompletion))
	g.PUT("/api/v1/ai/provider", perm(handleUpdateAIProvider, "ai:manage"))

	// WebSocket.
	g.GET("/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Frontend pages.
	g.GET("/", notAuthPage(serveIndexPage))
	g.GET("/inboxes/{all:*}", authPage(serveIndexPage))
	g.GET("/teams/{all:*}", authPage(serveIndexPage))
	g.GET("/views/{all:*}", authPage(serveIndexPage))
	g.GET("/admin/{all:*}", authPage(serveIndexPage))
	g.GET("/reports/{all:*}", authPage(serveIndexPage))
	g.GET("/account/{all:*}", authPage(serveIndexPage))
	g.GET("/reset-password", notAuthPage(serveIndexPage))
	g.GET("/set-password", notAuthPage(serveIndexPage))
	// FIXME: Don't need three separate routes for the same thing.
	g.GET("/assets/{all:*}", serveFrontendStaticFiles)
	g.GET("/images/{all:*}", serveFrontendStaticFiles)
	g.GET("/static/public/{all:*}", serveStaticFiles)

	// Public pages.
	g.GET("/csat/{uuid}", handleShowCSAT)
	g.POST("/csat/{uuid}", handleUpdateCSATResponse)

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
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.file}"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.SetBody(file.ReadBytes())

	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSavingSession"), nil))
	}
	return nil
}

// serveStaticFiles serves static assets from the embedded filesystem.
func serveStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Get the requested file path.
	filePath := string(r.RequestCtx.Path())

	file, err := app.fs.Get(filePath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.file}"), nil, envelope.NotFoundError)
	}

	// Set the appropriate Content-Type based on the file extension.
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(file.ReadBytes())
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.SetBody(file.ReadBytes())
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
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.entities.file}"), nil, envelope.NotFoundError)
	}

	// Set the appropriate Content-Type based on the file extension.
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(file.ReadBytes())
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
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
