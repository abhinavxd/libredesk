package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai"
	"github.com/jmoiron/sqlx/types"
)

const (
	contactHistoryDefaultLimit = 5
	contactHistoryMaxLimit     = 10
	contactHistoryScanCap      = 30
	contactHistorySnippetLen   = 200
)

var contactHistoryParams = types.JSONText(`{
	"type": "object",
	"properties": {
		"limit": {
			"type": "integer",
			"description": "Maximum number of past conversations to return (default 5, max 10)."
		}
	}
}`)

// contactHistoryTool lets the agent look up the current contact's previous conversations, filtered to what the requesting agent may access.
type contactHistoryTool struct {
	app         *App
	agentID     int
	contactID   int
	excludeUUID string
}

// contactHistoryTools returns the tool wrapped for the agent run, or nil when there is no real contact.
func contactHistoryTools(app *App, agentID, contactID int, excludeUUID string) []ai.Tool {
	if contactID <= 0 {
		return nil
	}
	return []ai.Tool{&contactHistoryTool{app: app, agentID: agentID, contactID: contactID, excludeUUID: excludeUUID}}
}

func (t *contactHistoryTool) Name() string { return ai.ToolContactHistory }

func (t *contactHistoryTool) Description() string {
	return "Get this customer's previous conversations (subject, date, last message). Call it only when the customer's history is relevant, e.g. a recurring issue or something discussed earlier."
}

func (t *contactHistoryTool) Parameters() types.JSONText { return contactHistoryParams }

func (t *contactHistoryTool) Execute(ctx context.Context, args string) (string, error) {
	limit := contactHistoryDefaultLimit
	if strings.TrimSpace(args) != "" {
		var in struct {
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal([]byte(args), &in); err == nil && in.Limit > 0 {
			limit = in.Limit
		}
	}
	if limit > contactHistoryMaxLimit {
		limit = contactHistoryMaxLimit
	}

	candidates, err := t.app.conversation.GetContactPreviousConversations(t.contactID, contactHistoryScanCap)
	if err != nil {
		return "", err
	}

	uuids := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c.UUID != t.excludeUUID {
			uuids = append(uuids, c.UUID)
		}
	}
	if len(uuids) == 0 {
		return "No previous conversations for this contact.", nil
	}

	authorized, err := t.app.conversation.FilterAuthorizedListUUIDs(t.agentID, uuids)
	if err != nil {
		return "", err
	}
	allowed := make(map[string]bool, len(authorized))
	for _, u := range authorized {
		allowed[u] = true
	}

	var b strings.Builder
	b.WriteString("Contact's previous conversations (most recent first):\n")
	n := 0
	for _, c := range candidates {
		if c.UUID == t.excludeUUID || !allowed[c.UUID] {
			continue
		}
		n++
		subject := strings.TrimSpace(c.Subject)
		if subject == "" {
			subject = "(no subject)"
		}
		fmt.Fprintf(&b, "[%d] %q - opened %s", n, subject, c.CreatedAt.Format("2006-01-02"))
		if msg := strings.TrimSpace(c.LastMessage.String); c.LastMessage.Valid && msg != "" {
			fmt.Fprintf(&b, ". Last message: %s", truncateText(msg, contactHistorySnippetLen))
		}
		b.WriteString("\n")
		if n >= limit {
			break
		}
	}
	if n == 0 {
		return "No previous conversations for this contact.", nil
	}
	return b.String(), nil
}

func truncateText(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
