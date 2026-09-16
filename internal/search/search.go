// Package search provides search functionality.
package search

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

const (
	maxPageSize = 100

	conversationResultOrder = "(conversations.reference_number = $1) DESC, conversations.last_message_at DESC NULLS LAST"
	messageResultOrder      = "conversation_messages.created_at DESC NULLS LAST"
)

var (
	//go:embed queries.sql
	efs embed.FS

	messageAllowedFields = []string{"created_at"}
)

// Manager is the search manager
type Manager struct {
	q               queries
	db              *sqlx.DB
	lo              *logf.Logger
	i18n            *i18n.I18n
	filterFields    dbutil.AllowedFields
	filterRenderers dbutil.FieldRenderers
	filterLocation  func() string
}

// Opts contains the options for creating a new search manager
type Opts struct {
	DB              *sqlx.DB
	Lo              *logf.Logger
	I18n            *i18n.I18n
	FilterFields    dbutil.AllowedFields
	FilterRenderers dbutil.FieldRenderers
	FilterLocation  func() string
}

// queries contains all the prepared queries
type queries struct {
	SearchConversations string     `query:"search-conversations"`
	SearchMessages      string     `query:"search-messages"`
	SearchContacts      *sqlx.Stmt `query:"search-contacts"`
}

// New creates a new search manager
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:               q,
		db:              opts.DB,
		lo:              opts.Lo,
		i18n:            opts.I18n,
		filterFields:    opts.FilterFields,
		filterRenderers: opts.FilterRenderers,
		filterLocation:  opts.FilterLocation,
	}, nil
}

// Conversations searches conversations the agent is allowed to read, returning the page and the total match count.
func (s *Manager) Conversations(query models.Query, scope models.ReadScope) ([]models.ConversationResult, int, error) {
	sql, args, err := s.buildQuery(s.q.SearchConversations, query, scope, conversationResultOrder, s.filterFields)
	if err != nil {
		return nil, 0, err
	}
	var results = make([]models.ConversationResult, 0)
	if err := s.db.Select(&results, sql, args...); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, 0, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	total := 0
	if len(results) > 0 {
		total = results[0].Total
	}
	return results, total, nil
}

// Messages searches messages in conversations the agent is allowed to read, returning the page and the total match count.
func (s *Manager) Messages(query models.Query, scope models.ReadScope) ([]models.MessageResult, int, error) {
	fields := dbutil.AllowedFields{"conversation_messages": messageAllowedFields}
	for model, f := range s.filterFields {
		fields[model] = f
	}
	sql, args, err := s.buildQuery(s.q.SearchMessages, query, scope, messageResultOrder, fields)
	if err != nil {
		return nil, 0, err
	}
	var results = make([]models.MessageResult, 0)
	if err := s.db.Select(&results, sql, args...); err != nil {
		s.lo.Error("error searching messages", "error", err)
		return nil, 0, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	total := 0
	if len(results) > 0 {
		total = results[0].Total
	}
	return results, total, nil
}

// Contacts searches contacts based on the query
func (s *Manager) Contacts(query string, limit int) ([]models.ContactResult, error) {
	var results = make([]models.ContactResult, 0)
	if err := s.q.SearchContacts.Select(&results, query, limit); err != nil {
		s.lo.Error("error searching contacts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return results, nil
}

func NormalizeQuery(query models.Query) models.Query {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > maxPageSize {
		query.PageSize = maxPageSize
	}
	if query.Filters == "" {
		query.Filters = "[]"
	}
	return query
}

func (s *Manager) buildQuery(base string, query models.Query, scope models.ReadScope, orderBy string, fields dbutil.AllowedFields) (string, []any, error) {
	query = NormalizeQuery(query)
	sql, args, err := dbutil.BuildFilterQuery(base, append([]any{query.Term}, scopeArgs(scope)...), query.Filters, fields, s.filterRenderers, s.filterLocation())
	if err != nil {
		s.lo.Error("error building search query", "error", err)
		return "", nil, envelope.NewError(envelope.InputError, s.i18n.T("globals.messages.invalidFilters"), nil)
	}
	sql += " ORDER BY " + orderBy
	sql += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, query.PageSize, dbutil.PageOffset(query.Page, query.PageSize))
	return sql, args, nil
}

func scopeArgs(scope models.ReadScope) []any {
	return []any{
		scope.UserID,
		scope.Read,
		scope.ReadAll,
		scope.ReadAssigned,
		scope.ReadTeamAll,
		scope.ReadTeamInbox,
		scope.ReadUnassigned,
		pq.Array(scope.TeamIDs),
	}
}
