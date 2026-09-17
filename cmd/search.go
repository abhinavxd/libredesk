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

	maxConversationSearchLimit = 1000
	maxMessageSearchLimit      = 30
	maxContactSearchLimit      = 15
)

type searchPageResults struct {
	Results    any    `json:"results"`
	PerPage    int    `json:"per_page"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

func handleSearchConversations(r *fastglue.Request) error {
	app, user, term, err := searchTerm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.ConversationFirstPage(term, scope, searchLimit(r, maxConversationSearchLimit))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

func handleSearchMessages(r *fastglue.Request) error {
	app, user, term, err := searchTerm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.MessageFirstPage(term, scope, searchLimit(r, maxMessageSearchLimit))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

func handleSearchContacts(r *fastglue.Request) error {
	app, _, term, err := searchTerm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Contacts(term, searchLimit(r, maxContactSearchLimit))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

func handlePaginatedSearchConversations(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, hasMore, nextCursor, err := app.search.Conversations(q, scope)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(searchPageResults{
		Results:    results,
		PerPage:    q.PageSize,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	})
}

func handlePaginatedSearchMessages(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	scope, err := readScope(app, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, hasMore, nextCursor, err := app.search.Messages(q, scope)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(searchPageResults{
		Results:    results,
		PerPage:    q.PageSize,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	})
}

func searchInputs(r *fastglue.Request) (*App, amodels.User, smodels.Query, error) {
	app, user, term, err := searchTerm(r)
	if err != nil {
		return app, user, smodels.Query{}, err
	}
	_, pageSize := getPagination(r)
	query := searchmanager.NormalizeQuery(smodels.Query{
		Term:     term,
		Filters:  string(r.RequestCtx.QueryArgs().Peek("filters")),
		Cursor:   string(r.RequestCtx.QueryArgs().Peek("cursor")),
		Sort:     smodels.Sort(r.RequestCtx.QueryArgs().Peek("sort")),
		PageSize: pageSize,
	})
	return app, user, query, nil
}

func searchTerm(r *fastglue.Request) (*App, amodels.User, string, error) {
	app := r.Context.(*App)
	user, _ := r.RequestCtx.UserValue("user").(amodels.User)
	term := string(r.RequestCtx.QueryArgs().Peek("query"))
	if len(term) < minSearchQueryLength {
		return app, user, "", envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil)
	}
	return app, user, term, nil
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
