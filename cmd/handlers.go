package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const maxPageSize = 500

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

	// Built-in MCP (API key only).
	g.OPTIONS("/mcp", handleMCPOptions)
	g.GET("/mcp", mcpAuth(handleMCP))
	g.POST("/mcp", mcpAuth(handleMCP))
	g.DELETE("/mcp", mcpAuth(handleMCP))

	// Media - supports both authenticated access and signed URLs.
	g.GET("/uploads/{uuid}", authOrSignedURL(handleServeMedia))
	g.POST("/api/v1/media", auth(handleMediaUpload))

	// Settings.
	g.GET("/api/v1/settings/general", auth(handleGetGeneralSettings))
	g.PUT("/api/v1/settings/general", perm(clearsHCCache(handleUpdateGeneralSettings), "general_settings:manage"))
	g.GET("/api/v1/settings/notifications/email", perm(handleGetEmailNotificationSettings, "notification_settings:manage"))
	g.PUT("/api/v1/settings/notifications/email", perm(handleUpdateEmailNotificationSettings, "notification_settings:manage"))

	// OpenID connect single sign-on.
	g.GET("/api/v1/oidc", perm(handleGetAllOIDC, "oidc:manage"))
	g.POST("/api/v1/oidc", perm(handleCreateOIDC, "oidc:manage"))
	g.GET("/api/v1/oidc/{id}", perm(handleGetOIDC, "oidc:manage"))
	g.PUT("/api/v1/oidc/{id}", perm(handleUpdateOIDC, "oidc:manage"))
	g.DELETE("/api/v1/oidc/{id}", perm(handleDeleteOIDC, "oidc:manage"))

	// Conversations.
	g.GET("/api/v1/conversations/all", perm(handleGetAllConversations, "conversations:read_all"))
	g.GET("/api/v1/conversations/unassigned", perm(handleGetUnassignedConversations, "conversations:read_unassigned"))
	g.GET("/api/v1/conversations/assigned", perm(handleGetAssignedConversations, "conversations:read_assigned"))
	g.GET("/api/v1/conversations/mentioned", perm(handleGetMentionedConversations, "conversations:read"))
	g.GET("/api/v1/teams/{id}/conversations/unassigned", perm(handleGetTeamUnassignedConversations, "conversations:read_team_inbox"))
	g.GET("/api/v1/views/{id}/conversations", perm(handleGetViewConversations, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	g.DELETE("/api/v1/conversations/{uuid}", perm(handleDeleteConversation, "conversations:write"))
	g.POST("/api/v1/conversations/{uuid}/merge", perm(handleMergeConversation, "conversations:write"))
	g.GET("/api/v1/conversations/{uuid}/participants", perm(handleGetConversationParticipants, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user", perm(handleUpdateUserAssignee, "conversations:update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee, "conversations:update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user/remove", perm(handleRemoveUserAssignee, "conversations:update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team/remove", perm(handleRemoveTeamAssignee, "conversations:update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/priority", perm(handleUpdateConversationPriority, "conversations:update_priority"))
	g.PUT("/api/v1/conversations/{uuid}/status", perm(handleUpdateConversationStatus, "conversations:update_status"))
	g.PUT("/api/v1/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/mark-unread", perm(handleMarkConversationAsUnread, "conversations:read"))
	g.POST("/api/v1/conversations/{uuid}/tags", perm(handleUpdateConversationtags, "conversations:update_tags"))
	g.GET("/api/v1/conversations/{uuid}/page-visits", perm(handleGetContactPageVisits, "conversations:read"))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleGetMessage, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/messages", perm(handleGetMessages, "messages:read"))
	g.GET("/api/v1/conversations/{uuid}/transcript", perm(handleDownloadConversationTranscript, "messages:read"))
	g.POST("/api/v1/conversations/{cuuid}/messages", auth(handleSendMessage))
	g.PUT("/api/v1/conversations/{cuuid}/messages/{uuid}/retry", perm(handleRetryMessage, "messages:write"))
	g.DELETE("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleDeleteMessage, "messages:write_private"))
	g.POST("/api/v1/conversations", perm(handleCreateConversation, "conversations:write"))
	g.PUT("/api/v1/conversations/{uuid}/custom-attributes", auth(handleUpdateConversationCustomAttributes))
	g.PUT("/api/v1/conversations/{uuid}/contacts/custom-attributes", auth(handleUpdateContactCustomAttributes))
	// Draft endpoints
	g.GET("/api/v1/drafts", auth(handleGetAllDrafts))
	g.POST("/api/v1/conversations/{uuid}/draft", auth(handleUpsertConversationDraft))
	g.DELETE("/api/v1/conversations/{uuid}/draft", auth(handleDeleteConversationDraft))

	// Search.
	g.GET("/api/v1/conversations/search", perm(handleSearchConversations, "conversations:read"))
	g.GET("/api/v1/messages/search", perm(handleSearchMessages, "messages:read"))
	g.GET("/api/v1/contacts/search", perm(handleSearchContacts, "contacts:read"))

	// Views.
	g.GET("/api/v1/views/me", perm(handleGetUserViews, "view:manage"))
	g.POST("/api/v1/views/me", perm(handleCreateUserView, "view:manage"))
	g.PUT("/api/v1/views/me/{id}", perm(handleUpdateUserView, "view:manage"))
	g.DELETE("/api/v1/views/me/{id}", perm(handleDeleteUserView, "view:manage"))

	g.GET("/api/v1/views/shared", auth(handleGetSharedViews))

	g.GET("/api/v1/shared-views", perm(handleGetAllSharedViews, "shared_views:manage"))
	g.GET("/api/v1/shared-views/{id}", perm(handleGetSharedView, "shared_views:manage"))
	g.POST("/api/v1/shared-views", perm(handleCreateSharedView, "shared_views:manage"))
	g.PUT("/api/v1/shared-views/{id}", perm(handleUpdateSharedView, "shared_views:manage"))
	g.DELETE("/api/v1/shared-views/{id}", perm(handleDeleteSharedView, "shared_views:manage"))

	// Status and priority.
	g.GET("/api/v1/statuses", auth(handleGetStatuses))
	g.POST("/api/v1/statuses", perm(handleCreateStatus, "status:manage"))
	g.PUT("/api/v1/statuses/{id}", perm(handleUpdateStatus, "status:manage"))
	g.DELETE("/api/v1/statuses/{id}", perm(handleDeleteStatus, "status:manage"))
	g.GET("/api/v1/priorities", auth(handleGetPriorities))

	// Tags.
	g.GET("/api/v1/tags", auth(handleGetTags))
	g.POST("/api/v1/tags", perm(handleCreateTag, "tags:manage"))
	g.PUT("/api/v1/tags/{id}", perm(handleUpdateTag, "tags:manage"))
	g.DELETE("/api/v1/tags/{id}", perm(handleDeleteTag, "tags:manage"))
	g.POST("/api/v1/tags/import", perm(handleImportTags, "tags:manage"))
	g.GET("/api/v1/tags/import/status", perm(handleGetTagImportStatus, "tags:manage"))

	// Organizations.
	g.GET("/api/v1/organizations", auth(handleGetOrganizations))
	g.GET("/api/v1/organizations/{id}", perm(handleGetOrganization, "organizations:manage"))
	g.POST("/api/v1/organizations", perm(handleCreateOrganization, "organizations:manage"))
	g.PUT("/api/v1/organizations/{id}", perm(handleUpdateOrganization, "organizations:manage"))
	g.DELETE("/api/v1/organizations/{id}", perm(handleDeleteOrganization, "organizations:manage"))

	// Macros.
	g.GET("/api/v1/macros", auth(handleGetMacros))
	g.GET("/api/v1/macros/{id}", perm(handleGetMacro, "macros:manage"))
	g.POST("/api/v1/macros", perm(handleCreateMacro, "macros:manage"))
	g.PUT("/api/v1/macros/{id}", perm(handleUpdateMacro, "macros:manage"))
	g.DELETE("/api/v1/macros/{id}", perm(handleDeleteMacro, "macros:manage"))
	g.POST("/api/v1/conversations/{uuid}/macros/{id}/apply", auth(handleApplyMacro))

	// Agents.
	g.GET("/api/v1/agents/me", auth(handleGetCurrentAgent))
	g.PUT("/api/v1/agents/me", auth(clearsHCCache(handleUpdateCurrentAgent)))
	g.GET("/api/v1/agents/me/teams", auth(handleGetCurrentAgentTeams))
	g.PUT("/api/v1/agents/me/availability", auth(handleUpdateAgentAvailability))
	g.DELETE("/api/v1/agents/me/avatar", auth(clearsHCCache(handleDeleteCurrentAgentAvatar)))
	g.GET("/api/v1/agents/me/api-key", auth(handleGetCurrentAgentAPIKey))
	g.POST("/api/v1/agents/me/api-key", auth(handleGenerateCurrentAgentAPIKey))
	g.DELETE("/api/v1/agents/me/api-key", auth(handleRevokeCurrentAgentAPIKey))

	g.GET("/api/v1/agents/compact", auth(handleGetAgentsCompact))
	g.GET("/api/v1/agents", perm(handleGetAgents, "users:manage"))
	g.GET("/api/v1/agents/{id}", perm(handleGetAgent, "users:manage"))
	g.POST("/api/v1/agents", perm(handleCreateAgent, "users:manage"))
	g.PUT("/api/v1/agents/{id}", perm(clearsHCCache(handleUpdateAgent), "users:manage"))
	g.DELETE("/api/v1/agents/{id}", perm(clearsHCCache(handleDeleteAgent), "users:manage"))
	g.POST("/api/v1/agents/import", perm(clearsHCCache(handleImportAgents), "users:manage"))
	g.GET("/api/v1/agents/import/status", perm(handleGetAgentImportStatus, "users:manage"))
	g.POST("/api/v1/agents/{id}/api-key", perm(handleGenerateAPIKey, "users:manage"))
	g.DELETE("/api/v1/agents/{id}/api-key", perm(handleRevokeAPIKey, "users:manage"))
	g.POST("/api/v1/agents/reset-password", rateLimit(tryAuth(handleResetPassword), "auth"))
	g.POST("/api/v1/agents/set-password", rateLimit(tryAuth(handleSetPassword), "auth"))

	// Contacts.
	g.GET("/api/v1/contacts", perm(handleGetContacts, "contacts:read_all"))
	g.GET("/api/v1/contacts/{id}", perm(handleGetContact, "contacts:read"))
	g.PUT("/api/v1/contacts/{id}", perm(handleUpdateContact, "contacts:write"))
	g.DELETE("/api/v1/contacts/{id}", perm(handleDeleteContact, "contacts:delete"))
	g.POST("/api/v1/contacts/{id}/merge", perm(handleMergeContact, "contacts:merge"))
	g.PUT("/api/v1/contacts/{id}/organization", perm(handleSetContactOrganization, "contacts:write"))
	g.PUT("/api/v1/contacts/{id}/block", perm(handleBlockContact, "contacts:block"))
	g.GET("/api/v1/contacts/{id}/export", perm(handleExportContact, "contacts:export"))
	g.GET("/api/v1/contacts/{id}/conversations", perm(handleGetContactConversations, "contacts:read"))

	// Contact notes.
	g.GET("/api/v1/contacts/{id}/notes", perm(handleGetContactNotes, "contact_notes:read"))
	g.POST("/api/v1/contacts/{id}/notes", perm(handleCreateContactNote, "contact_notes:write"))
	g.DELETE("/api/v1/contacts/{id}/notes/{note_id}", perm(handleDeleteContactNote, "contact_notes:delete"))

	// Teams.
	g.GET("/api/v1/teams/compact", auth(handleGetTeamsCompact))
	g.GET("/api/v1/teams", perm(handleGetTeams, "teams:manage"))
	g.GET("/api/v1/teams/{id}", perm(handleGetTeam, "teams:manage"))
	g.POST("/api/v1/teams", perm(handleCreateTeam, "teams:manage"))
	g.PUT("/api/v1/teams/{id}", perm(handleUpdateTeam, "teams:manage"))
	g.DELETE("/api/v1/teams/{id}", perm(handleDeleteTeam, "teams:manage"))

	// Automations.
	g.GET("/api/v1/automations/rules", perm(handleGetAutomationRules, "automations:manage"))
	g.GET("/api/v1/automations/rules/{id}", perm(handleGetAutomationRule, "automations:manage"))
	g.POST("/api/v1/automations/rules", perm(handleCreateAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automations/rules/{id}/toggle", perm(handleToggleAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automations/rules/{id}", perm(handleUpdateAutomationRule, "automations:manage"))
	g.PUT("/api/v1/automations/rules/weights", perm(handleUpdateAutomationRuleWeights, "automations:manage"))
	g.PUT("/api/v1/automations/rules/execution-mode", perm(handleUpdateAutomationRuleExecutionMode, "automations:manage"))
	g.DELETE("/api/v1/automations/rules/{id}", perm(handleDeleteAutomationRule, "automations:manage"))

	// Inboxes.
	g.GET("/api/v1/inboxes", auth(handleGetInboxes))
	g.GET("/api/v1/inboxes/{id}", perm(handleGetInbox, "inboxes:manage"))
	g.POST("/api/v1/inboxes", perm(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/v1/inboxes/{id}", perm(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/v1/inboxes/{id}", perm(handleDeleteInbox, "inboxes:manage"))

	// OAuth endpoints for email inboxes.
	g.POST("/api/v1/inboxes/oauth/{provider}/authorize", perm(handleOAuthAuthorize, "inboxes:manage"))
	g.GET("/api/v1/inboxes/oauth/{provider}/callback", perm(handleOAuthCallback, "inboxes:manage"))

	// Roles.
	g.GET("/api/v1/roles", auth(handleGetRoles))
	g.GET("/api/v1/roles/{id}", perm(handleGetRole, "roles:manage"))
	g.POST("/api/v1/roles", perm(handleCreateRole, "roles:manage"))
	g.PUT("/api/v1/roles/{id}", perm(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/v1/roles/{id}", perm(handleDeleteRole, "roles:manage"))

	// Webhooks.
	g.GET("/api/v1/webhooks/compact", auth(handleGetWebhooksCompact))
	g.GET("/api/v1/webhooks", perm(handleGetWebhooks, "webhooks:manage"))
	g.GET("/api/v1/webhooks/{id}", perm(handleGetWebhook, "webhooks:manage"))
	g.POST("/api/v1/webhooks", perm(handleCreateWebhook, "webhooks:manage"))
	g.PUT("/api/v1/webhooks/{id}", perm(handleUpdateWebhook, "webhooks:manage"))
	g.DELETE("/api/v1/webhooks/{id}", perm(handleDeleteWebhook, "webhooks:manage"))
	g.PUT("/api/v1/webhooks/{id}/toggle", perm(handleToggleWebhook, "webhooks:manage"))
	g.POST("/api/v1/webhooks/{id}/test", perm(handleTestWebhook, "webhooks:manage"))

	// Context Links.
	g.GET("/api/v1/context-links", perm(handleGetContextLinks, "context_links:manage"))
	g.GET("/api/v1/context-links/active", auth(handleGetActiveContextLinks))
	g.GET("/api/v1/context-links/{id}", perm(handleGetContextLink, "context_links:manage"))
	g.POST("/api/v1/context-links", perm(handleCreateContextLink, "context_links:manage"))
	g.PUT("/api/v1/context-links/{id}", perm(handleUpdateContextLink, "context_links:manage"))
	g.DELETE("/api/v1/context-links/{id}", perm(handleDeleteContextLink, "context_links:manage"))
	g.PUT("/api/v1/context-links/{id}/toggle", perm(handleToggleContextLink, "context_links:manage"))
	g.GET("/api/v1/context-links/{id}/url", auth(handleGetContextLinkURL))

	// Reports.
	g.GET("/api/v1/reports/overview/sla", perm(handleOverviewSLA, "reports:manage"))
	g.GET("/api/v1/reports/overview/counts", perm(handleOverviewCounts, "reports:manage"))
	g.GET("/api/v1/reports/overview/charts", perm(handleOverviewCharts, "reports:manage"))
	g.GET("/api/v1/reports/overview/csat", perm(handleOverviewCSAT, "reports:manage"))
	g.GET("/api/v1/reports/overview/messages", perm(handleOverviewMessageVolume, "reports:manage"))
	g.GET("/api/v1/reports/overview/tags", perm(handleOverviewTagDistribution, "reports:manage"))

	// Templates.
	g.GET("/api/v1/templates", perm(handleGetTemplates, "templates:manage"))
	g.GET("/api/v1/templates/{id}", perm(handleGetTemplate, "templates:manage"))
	g.POST("/api/v1/templates", perm(handleCreateTemplate, "templates:manage"))
	g.PUT("/api/v1/templates/{id}", perm(handleUpdateTemplate, "templates:manage"))
	g.DELETE("/api/v1/templates/{id}", perm(handleDeleteTemplate, "templates:manage"))

	// Business hours.
	g.GET("/api/v1/business-hours", auth(handleGetBusinessHours))
	g.GET("/api/v1/business-hours/{id}", perm(handleGetBusinessHour, "business_hours:manage"))
	g.POST("/api/v1/business-hours", perm(handleCreateBusinessHours, "business_hours:manage"))
	g.PUT("/api/v1/business-hours/{id}", perm(handleUpdateBusinessHours, "business_hours:manage"))
	g.DELETE("/api/v1/business-hours/{id}", perm(handleDeleteBusinessHour, "business_hours:manage"))

	// SLAs.
	g.GET("/api/v1/sla", auth(handleGetSLAs))
	g.GET("/api/v1/sla/{id}", perm(handleGetSLA, "sla:manage"))
	g.POST("/api/v1/sla", perm(handleCreateSLA, "sla:manage"))
	g.PUT("/api/v1/sla/{id}", perm(handleUpdateSLA, "sla:manage"))
	g.DELETE("/api/v1/sla/{id}", perm(handleDeleteSLA, "sla:manage"))

	// AI completions.
	g.GET("/api/v1/ai/prompts", auth(handleGetAIPrompts))
	g.GET("/api/v1/ai/provider", perm(handleGetAIProvider, "ai:manage"))
	g.POST("/api/v1/ai/completion", auth(handleAICompletion))
	g.POST("/api/v1/ai/draft-reply", auth(handleAIDraftReply))
	g.PUT("/api/v1/ai/provider", perm(handleUpdateAIProvider, "ai:manage"))


	// AI provider config (completion / embedding).
	g.GET("/api/v1/ai/config/{type}", perm(handleGetAIConfig, "ai:manage"))
	g.PUT("/api/v1/ai/config/{type}", perm(handleUpdateAIConfig, "ai:manage"))
	g.POST("/api/v1/ai/config/{type}/test", perm(handleTestAIConfig, "ai:manage"))

	// AI custom tools.
	g.GET("/api/v1/ai/tools", perm(handleGetAITools, "ai:manage"))
	g.GET("/api/v1/ai/tools/{id}", perm(handleGetAITool, "ai:manage"))
	g.POST("/api/v1/ai/tools", perm(handleCreateAITool, "ai:manage"))
	g.PUT("/api/v1/ai/tools/{id}", perm(handleUpdateAITool, "ai:manage"))
	g.DELETE("/api/v1/ai/tools/{id}", perm(handleDeleteAITool, "ai:manage"))

	// AI knowledge base snippets.
	g.GET("/api/v1/ai/snippets", perm(handleGetAISnippets, "ai:manage"))
	g.POST("/api/v1/ai/snippets", perm(handleCreateAISnippet, "ai:manage"))
	g.POST("/api/v1/ai/snippets/import-url", perm(handleImportAISnippetFromURL, "ai:manage"))
	g.PUT("/api/v1/ai/snippets/{id}", perm(handleUpdateAISnippet, "ai:manage"))
	g.DELETE("/api/v1/ai/snippets/{id}", perm(handleDeleteAISnippet, "ai:manage"))

	// AI assistant: reply drafting + copilot chat.
	g.POST("/api/v1/ai/generate-reply", auth(handleAIGenerateReply))
	g.POST("/api/v1/ai/summarize", perm(handleAISummarizeConversation, "messages:write_private"))
	g.POST("/api/v1/ai/suggest-tags", auth(handleAISuggestTags))
	g.POST("/api/v1/ai/copilot", auth(handleAICopilot))
	g.GET("/api/v1/ai/copilot/messages", auth(handleGetCopilotMessages))
	g.DELETE("/api/v1/ai/copilot/messages", auth(handleClearCopilotMessages))

	// Autonomous AI agents (assistants).
	g.GET("/api/v1/ai/assistants/compact", auth(handleGetAIAssistantsCompact))
	g.GET("/api/v1/ai/assistants", perm(handleGetAIAssistants, "ai:manage"))
	g.GET("/api/v1/ai/assistants/{id}", perm(handleGetAIAssistant, "ai:manage"))
	g.POST("/api/v1/ai/assistants", perm(handleCreateAIAssistant, "ai:manage"))
	g.PUT("/api/v1/ai/assistants/{id}", perm(clearsHCCache(handleUpdateAIAssistant), "ai:manage"))
	g.DELETE("/api/v1/ai/assistants/{id}", perm(clearsHCCache(handleDeleteAIAssistant), "ai:manage"))
	g.POST("/api/v1/ai/assistants/{id}/preview", perm(handleAIAssistantPreview, "ai:manage"))
	g.GET("/api/v1/ai/assistants/{id}/stats", perm(handleGetAIAssistantStats, "ai:manage"))

	// AI FAQ learning: review queue for suggestions mined from resolved conversations + on/off setting.
	g.GET("/api/v1/ai/faq-suggestions", perm(handleGetAIFaqSuggestions, "ai:manage"))
	g.POST("/api/v1/ai/faq-suggestions/{id}/approve", perm(handleApproveAIFaqSuggestion, "ai:manage"))
	g.POST("/api/v1/ai/faq-suggestions/{id}/reject", perm(handleRejectAIFaqSuggestion, "ai:manage"))
	g.GET("/api/v1/ai/faq-learning", perm(handleGetAIFaqLearning, "ai:manage"))
	g.PUT("/api/v1/ai/faq-learning", perm(handleUpdateAIFaqLearning, "ai:manage"))

	// Help centers.
	g.GET("/api/v1/help-centers", perm(handleGetHelpCenters, "help_center:manage"))
	g.GET("/api/v1/help-centers/locales", perm(handleGetHelpCenterLocales, "help_center:manage"))
	g.GET("/api/v1/help-centers/{id}", perm(handleGetHelpCenter, "help_center:manage"))
	g.GET("/api/v1/help-centers/{id}/tree", perm(handleGetHelpCenterTree, "help_center:manage"))
	g.POST("/api/v1/help-centers", perm(clearsHCCache(handleCreateHelpCenter), "help_center:manage"))
	g.PUT("/api/v1/help-centers/{id}", perm(clearsHCCache(handleUpdateHelpCenter), "help_center:manage"))
	g.POST("/api/v1/help-centers/{id}/preview", perm(handleHelpCenterPreview, "help_center:manage"))
	g.PUT("/api/v1/help-centers/{id}/toggle", perm(clearsHCCache(handleToggleHelpCenterActive), "help_center:manage"))
	g.DELETE("/api/v1/help-centers/{id}", perm(clearsHCCache(handleDeleteHelpCenter), "help_center:manage"))
	g.GET("/api/v1/help-centers/{hc_id}/collections", perm(handleGetCollections, "help_center:manage"))
	g.POST("/api/v1/help-centers/{hc_id}/collections", perm(clearsHCCache(handleCreateCollection), "help_center:manage"))
	g.PUT("/api/v1/help-centers/{hc_id}/collections/{id}", perm(clearsHCCache(handleUpdateCollection), "help_center:manage"))
	g.DELETE("/api/v1/help-centers/{hc_id}/collections/{id}", perm(clearsHCCache(handleDeleteCollection), "help_center:manage"))
	g.PUT("/api/v1/help-centers/{hc_id}/collection-sort-order", perm(clearsHCCache(handleUpdateCollectionSortOrders), "help_center:manage"))
	g.PUT("/api/v1/collections/{id}/toggle", perm(clearsHCCache(handleToggleCollection), "help_center:manage"))
	g.PUT("/api/v1/collections/{col_id}/article-sort-order", perm(clearsHCCache(handleUpdateArticleSortOrders), "help_center:manage"))
	g.GET("/api/v1/collections/{col_id}/articles/{id}", perm(handleGetArticle, "help_center:manage"))
	g.POST("/api/v1/collections/{col_id}/articles", perm(clearsHCCache(handleCreateArticle), "help_center:manage"))
	g.DELETE("/api/v1/collections/{col_id}/articles/{id}", perm(clearsHCCache(handleDeleteArticle), "help_center:manage"))
	g.PUT("/api/v1/articles/{id}", perm(clearsHCCache(handleUpdateArticle), "help_center:manage"))
	g.PUT("/api/v1/articles/{id}/collection", perm(clearsHCCache(handleMoveArticle), "help_center:manage"))
	g.PUT("/api/v1/articles/{id}/status", perm(clearsHCCache(handleUpdateArticleStatus), "help_center:manage"))
	g.GET("/api/v1/help-centers/{id}/insights", perm(handleGetHelpCenterInsights, "help_center:manage"))

	// Public help center JSON API.
	g.GET("/api/v1/public/help-centers/{slug}/tree", rateLimit(handleGetPublicHelpCenterTree, "public"))
	g.GET("/api/v1/public/help-centers/{slug}/articles/{article_slug}", rateLimit(handleGetPublicHelpCenterArticle, "public"))
	g.GET("/api/v1/public/help-centers/{slug}/search", rateLimit(handlePublicHelpCenterSearch, "public"))
	g.POST("/api/v1/public/help-centers/{slug}/articles/{article_slug}/feedback", rateLimit(handleHelpCenterArticleFeedback, "public"))

	// Custom attributes.
	g.GET("/api/v1/custom-attributes", auth(handleGetCustomAttributes))
	g.POST("/api/v1/custom-attributes", perm(handleCreateCustomAttribute, "custom_attributes:manage"))
	g.GET("/api/v1/custom-attributes/{id}", perm(handleGetCustomAttribute, "custom_attributes:manage"))
	g.PUT("/api/v1/custom-attributes/{id}", perm(handleUpdateCustomAttribute, "custom_attributes:manage"))
	g.DELETE("/api/v1/custom-attributes/{id}", perm(handleDeleteCustomAttribute, "custom_attributes:manage"))

	// Actvity logs.
	g.GET("/api/v1/activity-logs", perm(handleGetActivityLogs, "activity_logs:manage"))

	// CSAT.
	g.POST("/api/v1/csat/{uuid}/response", rateLimit(handleSubmitCSATResponse, "public"))

	// User notifications.
	g.GET("/api/v1/notifications", auth(handleGetUserNotifications))
	g.GET("/api/v1/notifications/stats", auth(handleGetUserNotificationStats))
	g.PUT("/api/v1/notifications/{id}/read", auth(handleMarkNotificationAsRead))
	g.PUT("/api/v1/notifications/read-all", auth(handleMarkAllNotificationsAsRead))
	g.DELETE("/api/v1/notifications/{id}", auth(handleDeleteNotification))
	g.DELETE("/api/v1/notifications", auth(handleDeleteAllNotifications))

	// WebSocket.
	g.GET("/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Live chat widget websocket.
	g.GET("/widget/ws", rateLimit(handleWidgetWS, "widget"))

	// Widget APIs.
	g.GET("/api/v1/widget/chat/settings/launcher", rateLimit(validateWidgetInbox(handleGetChatLauncherSettings), "widget"))
	g.GET("/api/v1/widget/chat/settings", rateLimit(validateWidgetInbox(handleGetChatSettings), "widget"))
	g.POST("/api/v1/widget/chat/auth/exchange", rateLimit(validateWidgetInbox(handleAuthExchange), "widget"))
	g.GET("/api/v1/widget/chat/auth/me", rateLimit(widgetAuth(handleWidgetAuthMe), "widget"))
	g.POST("/api/v1/widget/chat/conversations/init", rateLimit(widgetAuth(handleChatInit), "widget"))
	g.GET("/api/v1/widget/chat/conversations", rateLimit(widgetAuth(handleGetConversations), "widget"))
	g.POST("/api/v1/widget/chat/conversations/{uuid}/update-last-seen", rateLimit(widgetAuth(handleChatUpdateLastSeen), "widget"))
	g.GET("/api/v1/widget/chat/conversations/{uuid}", rateLimit(widgetAuth(handleChatGetConversation), "widget"))
	g.POST("/api/v1/widget/chat/conversations/{uuid}/message", rateLimit(widgetAuth(handleChatSendMessage), "widget"))
	g.POST("/api/v1/widget/media/upload", rateLimit(widgetAuth(handleWidgetMediaUpload), "widget"))

	// getAndHead registers both methods: uptime checkers and link validators probe with HEAD.
	getAndHead := func(path string, h fastglue.FastRequestHandler) {
		g.GET(path, h)
		g.HEAD(path, h)
	}

	// Frontend pages.
	getAndHead("/", helpCenterHostHome(notAuthPage(serveIndexPage)))
	g.GET("/widget", validateWidgetInbox(serveWidgetIndexPage))
	g.GET("/widget/manifest.webmanifest", servePWAWidgetManifest)
	g.GET("/widget/sw.js", serveWidgetServiceWorker)
	g.GET("/inboxes/{all:*}", authPage(serveIndexPage))
	g.GET("/teams/{all:*}", authPage(serveIndexPage))
	g.GET("/views/{all:*}", authPage(serveIndexPage))
	g.GET("/admin/{all:*}", authPage(serveIndexPage))
	g.GET("/contacts/{all:*}", authPage(serveIndexPage))
	g.GET("/reports/{all:*}", authPage(serveIndexPage))
	g.GET("/account/{all:*}", authPage(serveIndexPage))
	g.GET("/reset-password", notAuthPage(serveIndexPage))
	g.GET("/set-password", notAuthPage(serveIndexPage))

	// PWA files. Served from the root so the service worker can control the
	// whole app scope and the manifest is discoverable at a stable URL.
	g.GET("/manifest.webmanifest", servePWAManifest)
	g.GET("/sw.js", serveServiceWorker)

	// Assets and static files.
	// FIXME: Reduce the number of routes.
	g.GET("/widget.js", serveWidgetJS)
	g.GET("/assets/{all:*}", serveFrontendStaticFiles)
	g.GET("/widget/assets/{all:*}", serveWidgetStaticFiles)
	g.GET("/images/{all:*}", serveFrontendStaticFiles)
	g.GET("/static/public/{all:*}", serveStaticFiles)

	// Public pages.
	getAndHead("/robots.txt", rateLimit(handleRobotsTxt, "public"))
	getAndHead("/sitemap.xml", rateLimit(handleSitemapIndex, "public"))
	getAndHead("/hc/{slug}", rateLimit(handleRedirectHelpCenterHome, "public"))
	getAndHead("/hc/{slug}/{locale}", rateLimit(cachedHCPage(handleShowHelpCenterHome), "public"))
	getAndHead("/hc/{slug}/{locale}/sitemap.xml", rateLimit(handleHelpCenterSitemap, "public"))
	getAndHead("/hc/{slug}/{locale}/search", rateLimit(cachedHCNoIndexPage(handleHelpCenterSearch), "public"))
	getAndHead("/hc/{slug}/{locale}/collections/{collection_slug}", rateLimit(cachedHCPage(handleShowHelpCenterCollection), "public"))
	getAndHead("/hc/{slug}/{locale}/articles/{article_slug}", rateLimit(countArticleView(cachedHCPage(handleShowHelpCenterArticle)), "public"))

	g.GET("/csat/{uuid}", rateLimit(handleShowCSAT, "public"))
	g.GET("/csat/{uuid}/widget", rateLimit(handleShowCSATWidget, "public"))
	g.POST("/csat/{uuid}", rateLimit(handleUpdateCSATResponse, "public"))

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

// serveWidgetIndexPage serves the widget index page of the application.
func serveWidgetIndexPage(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Prevent caching of the index page.
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")

	// CSP headers if trusted domains is set.
	if config, err := getWidgetConfig(r); err == nil && len(config.TrustedDomains) > 0 {
		csp := "frame-ancestors 'self' " + strings.Join(config.TrustedDomains, " ")
		r.RequestCtx.Response.Header.Set("Content-Security-Policy", csp)
	}

	// Serve the index.html file from the embedded filesystem.
	file, err := app.fs.Get(path.Join(widgetDir, "index.html"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.Response.SetBodyRaw(file.ReadBytes())

	return nil
}

// servePWAManifest serves the web app manifest used for PWA installation.
func servePWAManifest(r *fastglue.Request) error {
	app := r.Context.(*App)

	file, err := app.fs.Get(path.Join(frontendDir, "manifest.webmanifest"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "application/manifest+json")
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// serveServiceWorker serves the PWA service worker. It is served from the root
// path so it can control the entire app scope, with caching disabled so worker
// updates are picked up promptly.
func serveServiceWorker(r *fastglue.Request) error {
	app := r.Context.(*App)

	file, err := app.fs.Get(path.Join(frontendDir, "sw.js"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/javascript")
	r.RequestCtx.Response.Header.Set("Service-Worker-Allowed", "/")
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-cache")
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// servePWAWidgetManifest serves the chat widget's web app manifest. The widget
// is launched with an `inbox_id` query param, so it is templated into start_url
// to keep an installed widget pointed at the correct inbox.
func servePWAWidgetManifest(r *fastglue.Request) error {
	startURL := "/widget"
	if id := string(r.RequestCtx.QueryArgs().Peek("inbox_id")); id != "" {
		// Only accept a numeric inbox id to avoid injecting arbitrary values.
		if _, err := strconv.Atoi(id); err == nil {
			startURL = "/widget?inbox_id=" + id
		}
	}

	manifest := map[string]any{
		"name":             "Libredesk Chat",
		"short_name":       "Libredesk Chat",
		"description":      "Libredesk live chat.",
		"id":               "/widget",
		"start_url":        startURL,
		"scope":            "/widget",
		"display":          "standalone",
		"background_color": "#1a1a1d",
		"theme_color":      "#1a1a1d",
		"orientation":      "any",
		"icons": []map[string]any{
			{"src": "/images/pwa-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any"},
			{"src": "/images/pwa-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
			{"src": "/images/pwa-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
		},
	}

	body, err := json.Marshal(manifest)
	if err != nil {
		app := r.Context.(*App)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "application/manifest+json")
	r.RequestCtx.SetBody(body)
	return nil
}

// serveWidgetServiceWorker serves the widget service worker scoped to /widget.
func serveWidgetServiceWorker(r *fastglue.Request) error {
	app := r.Context.(*App)

	file, err := app.fs.Get(path.Join(widgetDir, "widget-sw.js"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/javascript")
	r.RequestCtx.Response.Header.Set("Service-Worker-Allowed", "/widget")
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-cache")
	r.RequestCtx.SetBody(file.ReadBytes())
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

// serveWidgetStaticFiles serves widget static assets from the embedded filesystem.
func serveWidgetStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	filePath := string(r.RequestCtx.Path())
	finalPath := filepath.Join(widgetDir, strings.TrimPrefix(filePath, "/widget"))

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

// serveWidgetJS serves the widget JavaScript file.
func serveWidgetJS(r *fastglue.Request) error {
	app := r.Context.(*App)

	r.RequestCtx.Response.Header.Set("Content-Type", "application/javascript")
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-cache")

	file, err := app.fs.Get("static/widget.js")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.T("validation.notFoundFile"), nil, envelope.NotFoundError)
	}

	r.RequestCtx.Response.SetBodyRaw(file.ReadBytes())
	return nil
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
