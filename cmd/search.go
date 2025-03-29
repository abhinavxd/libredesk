package main

import (
	"fmt"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3
)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	wrapper := func(query string) (interface{}, error) {
		return app.search.Conversations(query)
	}
	return handleSearch(r, wrapper)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	app := r.Context.(*App)
	wrapper := func(query string) (interface{}, error) {
		return app.search.Messages(query)
	}
	return handleSearch(r, wrapper)
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app := r.Context.(*App)
	wrapper := func(query string) (interface{}, error) {
		return app.search.Contacts(query)
	}
	return handleSearch(r, wrapper)
}

// handleSearch searches for the given query using the provided search function.
func handleSearch(r *fastglue.Request, searchFunc func(string) (interface{}, error)) error {
	var (
		app = r.Context.(*App)
		q   = string(r.RequestCtx.QueryArgs().Peek("query"))
	)

	if len(q) < minSearchQueryLength {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil))
	}

	results, err := searchFunc(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}
