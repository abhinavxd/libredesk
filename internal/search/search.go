// Package search provides search functionality.
package search

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager is the search manager
type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains the options for creating a new search manager
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains all the prepared queries
type queries struct {
	SearchConversationsByRefNum       *sqlx.Stmt `query:"search-conversations-by-reference-number"`
	SearchConversationsByContactEmail *sqlx.Stmt `query:"search-conversations-by-contact-email"`
	SearchMessages                    *sqlx.Stmt `query:"search-messages"`
	SearchContacts                    *sqlx.Stmt `query:"search-contacts"`
}

// New creates a new search manager
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo, i18n: opts.I18n}, nil
}

// Conversations searches conversations the agent is allowed to read.
func (s *Manager) Conversations(query string, scope models.ReadScope, limit int) ([]models.ConversationResult, error) {
	args := scopeArgs(scope)

	var refNumResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsByRefNum.Select(&refNumResults, append([]any{query}, args...)...); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var emailResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsByContactEmail.Select(&emailResults, append([]any{query}, append(args, limit)...)...); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	results := append(refNumResults, emailResults...)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// Messages searches messages in conversations the agent is allowed to read.
func (s *Manager) Messages(query string, scope models.ReadScope, limit int) ([]models.MessageResult, error) {
	var results = make([]models.MessageResult, 0)
	if err := s.q.SearchMessages.Select(&results, append([]any{query}, append(scopeArgs(scope), limit)...)...); err != nil {
		s.lo.Error("error searching messages", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return results, nil
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
