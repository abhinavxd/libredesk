package main

import (
	"fmt"
	"slices"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	smodels "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3

	maxConversationSearchLimit = 1000
	maxMessageSearchLimit      = 30
	maxContactSearchLimit      = 15
)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Conversations(q, scope, searchLimit(r, maxConversationSearchLimit))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Messages(q, scope, searchLimit(r, maxMessageSearchLimit))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app, _, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Contacts(q, searchLimit(r, maxContactSearchLimit))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

func searchInputs(r *fastglue.Request) (*App, amodels.User, string, error) {
	app := r.Context.(*App)
	user, _ := r.RequestCtx.UserValue("user").(amodels.User)
	q := string(r.RequestCtx.QueryArgs().Peek("query"))
	if len(q) < minSearchQueryLength {
		return app, user, "", envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil)
	}
	return app, user, q, nil
}

func searchLimit(r *fastglue.Request, max int) int {
	limit, err := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))
	if err != nil || limit < 1 || limit > max {
		return max
	}
	return limit
}

func readScope(app *App, agentID int) (smodels.ReadScope, error) {
	agent, err := app.user.GetAgentCachedOrLoad(agentID)
	if err != nil {
		return smodels.ReadScope{}, err
	}
	if !agent.Enabled {
		return smodels.ReadScope{}, nil
	}
	return smodels.ReadScope{
		UserID:         agent.ID,
		TeamIDs:        agent.Teams.IDs(),
		Read:           slices.Contains(agent.Permissions, authzmodels.PermConversationsRead),
		ReadAll:        slices.Contains(agent.Permissions, authzmodels.PermConversationsReadAll),
		ReadAssigned:   slices.Contains(agent.Permissions, authzmodels.PermConversationsReadAssigned),
		ReadTeamAll:    slices.Contains(agent.Permissions, authzmodels.PermConversationsReadTeamAll),
		ReadTeamInbox:  slices.Contains(agent.Permissions, authzmodels.PermConversationsReadTeamInbox),
		ReadUnassigned: slices.Contains(agent.Permissions, authzmodels.PermConversationsReadUnassigned),
	}, nil
}
