package main

import (
	"fmt"
	"slices"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	searchmanager "github.com/abhinavxd/libredesk/internal/search"
	smodels "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3

	maxContactSearchLimit = 15
)

// handleSearchConversations searches conversations by term with optional list filters, paginated.
func handleSearchConversations(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, total, err := app.search.Conversations(q, scope)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(pageResults(results, total, q))
}

// handleSearchMessages searches messages by term with optional list filters, paginated.
func handleSearchMessages(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, total, err := app.search.Messages(q, scope)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(pageResults(results, total, q))
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app, _, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	limit, err := strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("limit")))
	if err != nil || limit < 1 || limit > maxContactSearchLimit {
		limit = maxContactSearchLimit
	}
	results, err := app.search.Contacts(q.Term, limit)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

func searchInputs(r *fastglue.Request) (*App, amodels.User, smodels.Query, error) {
	app := r.Context.(*App)
	user, _ := r.RequestCtx.UserValue("user").(amodels.User)
	term := string(r.RequestCtx.QueryArgs().Peek("query"))
	if len(term) < minSearchQueryLength {
		return app, user, smodels.Query{}, envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil)
	}
	page, pageSize := getPagination(r)
	query := searchmanager.NormalizeQuery(smodels.Query{
		Term:     term,
		Filters:  string(r.RequestCtx.QueryArgs().Peek("filters")),
		Page:     page,
		PageSize: pageSize,
	})
	return app, user, query, nil
}

func pageResults(results any, total int, q smodels.Query) envelope.PageResults {
	return envelope.PageResults{
		Results:    results,
		Total:      total,
		PerPage:    q.PageSize,
		TotalPages: (total + q.PageSize - 1) / q.PageSize,
		Page:       q.Page,
	}
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
