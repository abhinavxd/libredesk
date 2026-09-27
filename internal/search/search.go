// Package search provides search functionality.
package search

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	maxPageSize                  = 100
	maxConversationFirstPageSize = 1000
	maxMessageFirstPageSize      = 30
)

var (
	//go:embed queries.sql
	efs embed.FS

	messageAllowedFields = []string{"created_at"}
)

type cursorCondition func(string, models.Sort, int) (string, []any, error)

type conversationCursor struct {
	ReferenceMatch bool        `json:"reference_match"`
	Sort           models.Sort `json:"sort"`
	SortAt         *time.Time  `json:"sort_at"`
	ID             int         `json:"id"`
}

type messageCursor struct {
	Sort      models.Sort `json:"sort"`
	CreatedAt time.Time   `json:"created_at"`
	ID        int         `json:"id"`
}

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

func (s *Manager) Conversations(query models.Query, scope models.ReadScope) ([]models.ConversationResult, bool, string, error) {
	return s.searchConversations(query, scope, maxPageSize)
}

func (s *Manager) ConversationFirstPage(term string, scope models.ReadScope, limit int) ([]models.ConversationResult, error) {
	results, _, _, err := s.searchConversations(models.Query{Term: term, PageSize: limit}, scope, maxConversationFirstPageSize)
	return results, err
}

func (s *Manager) Messages(query models.Query, scope models.ReadScope) ([]models.MessageResult, bool, string, error) {
	return s.searchMessages(query, scope, maxPageSize)
}

func (s *Manager) MessageFirstPage(term string, scope models.ReadScope, limit int) ([]models.MessageResult, error) {
	results, _, _, err := s.searchMessages(models.Query{Term: term, PageSize: limit}, scope, maxMessageFirstPageSize)
	return results, err
}

// Contacts searches contacts based on the query
func (s *Manager) Contacts(query string, limit int) ([]models.ContactResult, error) {
	var results = make([]models.ContactResult, 0)
	if err := s.q.SearchContacts.Select(&results, dbutil.ContainsPattern(query), limit); err != nil {
		s.lo.Error("error searching contacts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return results, nil
}

func NormalizeQuery(query models.Query) models.Query {
	return normalizeQuery(query, maxPageSize)
}

func (s *Manager) searchConversations(query models.Query, scope models.ReadScope, pageSizeLimit int) ([]models.ConversationResult, bool, string, error) {
	query = normalizeQuery(query, pageSizeLimit)
	orderBy, err := conversationResultOrder(query.Sort)
	if err != nil {
		return nil, false, "", envelope.NewError(envelope.InputError, s.i18n.T("globals.messages.invalidFilters"), nil)
	}
	sql, args, err := s.buildQuery(s.q.SearchConversations, query, scope, orderBy, s.filterFields, conversationCursorCondition)
	if err != nil {
		return nil, false, "", err
	}
	var results = make([]models.ConversationResult, 0)
	if err := s.db.Select(&results, sql, args...); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, false, "", envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	results, hasMore := trimResults(results, query.PageSize)
	if !hasMore {
		return results, false, "", nil
	}
	nextCursor, err := encodeCursor(conversationCursor{
		ReferenceMatch: results[len(results)-1].ReferenceMatch,
		Sort:           query.Sort,
		SortAt:         conversationSortAt(results[len(results)-1], query.Sort),
		ID:             results[len(results)-1].ID,
	})
	if err != nil {
		s.lo.Error("error encoding conversation search cursor", "error", err)
		return nil, false, "", envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return results, true, nextCursor, nil
}

func (s *Manager) searchMessages(query models.Query, scope models.ReadScope, pageSizeLimit int) ([]models.MessageResult, bool, string, error) {
	query = normalizeQuery(query, pageSizeLimit)
	orderBy, err := messageResultOrder(query.Sort)
	if err != nil {
		return nil, false, "", envelope.NewError(envelope.InputError, s.i18n.T("globals.messages.invalidFilters"), nil)
	}
	fields := dbutil.AllowedFields{"conversation_messages": messageAllowedFields}
	for model, f := range s.filterFields {
		fields[model] = f
	}
	sql, args, err := s.buildQuery(s.q.SearchMessages, query, scope, orderBy, fields, messageCursorCondition)
	if err != nil {
		return nil, false, "", err
	}
	var results = make([]models.MessageResult, 0)
	if err := s.db.Select(&results, sql, args...); err != nil {
		s.lo.Error("error searching messages", "error", err)
		return nil, false, "", envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	results, hasMore := trimResults(results, query.PageSize)
	if !hasMore {
		return results, false, "", nil
	}
	nextCursor, err := encodeCursor(messageCursor{
		Sort:      query.Sort,
		CreatedAt: results[len(results)-1].CreatedAt,
		ID:        results[len(results)-1].ID,
	})
	if err != nil {
		s.lo.Error("error encoding message search cursor", "error", err)
		return nil, false, "", envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return results, true, nextCursor, nil
}

func (s *Manager) buildQuery(base string, query models.Query, scope models.ReadScope, orderBy string, fields dbutil.AllowedFields, cursorWhere cursorCondition) (string, []any, error) {
	baseArgs := append([]any{query.Term}, scopeArgs(scope)...)
	baseArgs = append(baseArgs, dbutil.ContainsPattern(query.Term))
	sql, args, err := dbutil.BuildFilterQuery(base, baseArgs, query.Filters, fields, s.filterRenderers, s.filterLocation())
	if err != nil {
		s.lo.Error("error building search query", "error", err)
		return "", nil, envelope.NewError(envelope.InputError, s.i18n.T("globals.messages.invalidFilters"), nil)
	}
	if query.Cursor != "" {
		condition, cursorArgs, err := cursorWhere(query.Cursor, query.Sort, len(args)+1)
		if err != nil {
			s.lo.Error("error decoding search cursor", "error", err)
			return "", nil, envelope.NewError(envelope.InputError, s.i18n.T("globals.messages.invalidFilters"), nil)
		}
		sql += " AND " + condition
		args = append(args, cursorArgs...)
	}
	sql += " ORDER BY " + orderBy
	sql += fmt.Sprintf(" LIMIT $%d", len(args)+1)
	args = append(args, query.PageSize+1)
	return sql, args, nil
}

func normalizeQuery(query models.Query, pageSizeLimit int) models.Query {
	if query.PageSize < 1 || query.PageSize > pageSizeLimit {
		query.PageSize = pageSizeLimit
	}
	if query.Filters == "" {
		query.Filters = "[]"
	}
	if query.Sort == "" {
		query.Sort = models.SortNewest
	}
	return query
}

func conversationResultOrder(sort models.Sort) (string, error) {
	switch sort {
	case models.SortNewest:
		return "(conversations.reference_number = $1) DESC, conversations.last_message_at DESC NULLS LAST, conversations.id DESC", nil
	case models.SortOldest:
		return "(conversations.reference_number = $1) DESC, conversations.last_message_at ASC NULLS LAST, conversations.id ASC", nil
	case models.SortStartedFirst:
		return "(conversations.reference_number = $1) DESC, conversations.created_at ASC, conversations.id ASC", nil
	case models.SortStartedLast:
		return "(conversations.reference_number = $1) DESC, conversations.created_at DESC, conversations.id DESC", nil
	default:
		return "", fmt.Errorf("invalid conversation sort: %q", sort)
	}
}

func messageResultOrder(sort models.Sort) (string, error) {
	switch sort {
	case models.SortNewest:
		return "conversation_messages.created_at DESC, conversation_messages.id DESC", nil
	case models.SortOldest:
		return "conversation_messages.created_at ASC, conversation_messages.id ASC", nil
	default:
		return "", fmt.Errorf("invalid message sort: %q", sort)
	}
}

func conversationCursorCondition(raw string, sort models.Sort, firstPlaceholder int) (string, []any, error) {
	var cursor conversationCursor
	if err := decodeCursor(raw, &cursor); err != nil || cursor.ID < 1 || cursor.Sort != sort {
		return "", nil, fmt.Errorf("invalid conversation cursor")
	}

	switch sort {
	case models.SortNewest:
		condition := fmt.Sprintf(
			"((conversations.reference_number = $1), (conversations.last_message_at IS NOT NULL), COALESCE(conversations.last_message_at, '-infinity'::timestamptz), conversations.id) < ($%d::boolean, $%d::boolean, COALESCE($%d::timestamptz, '-infinity'::timestamptz), $%d)",
			firstPlaceholder,
			firstPlaceholder+1,
			firstPlaceholder+2,
			firstPlaceholder+3,
		)
		return condition, []any{cursor.ReferenceMatch, cursor.SortAt != nil, cursor.SortAt, cursor.ID}, nil
	case models.SortOldest:
		condition := fmt.Sprintf(
			"((conversations.reference_number != $1), (conversations.last_message_at IS NULL), COALESCE(conversations.last_message_at, 'infinity'::timestamptz), conversations.id) > ($%d::boolean, $%d::boolean, COALESCE($%d::timestamptz, 'infinity'::timestamptz), $%d)",
			firstPlaceholder,
			firstPlaceholder+1,
			firstPlaceholder+2,
			firstPlaceholder+3,
		)
		return condition, []any{!cursor.ReferenceMatch, cursor.SortAt == nil, cursor.SortAt, cursor.ID}, nil
	case models.SortStartedFirst, models.SortStartedLast:
		if cursor.SortAt == nil {
			return "", nil, fmt.Errorf("invalid conversation cursor")
		}
		operator := ">"
		referenceExpression := "(conversations.reference_number != $1)"
		referenceValue := !cursor.ReferenceMatch
		if sort == models.SortStartedLast {
			operator = "<"
			referenceExpression = "(conversations.reference_number = $1)"
			referenceValue = cursor.ReferenceMatch
		}
		condition := fmt.Sprintf(
			"(%s, conversations.created_at, conversations.id) %s ($%d::boolean, $%d::timestamptz, $%d)",
			referenceExpression,
			operator,
			firstPlaceholder,
			firstPlaceholder+1,
			firstPlaceholder+2,
		)
		return condition, []any{referenceValue, cursor.SortAt, cursor.ID}, nil
	default:
		return "", nil, fmt.Errorf("invalid conversation cursor")
	}
}

func messageCursorCondition(raw string, sort models.Sort, firstPlaceholder int) (string, []any, error) {
	var cursor messageCursor
	if err := decodeCursor(raw, &cursor); err != nil || cursor.ID < 1 || cursor.CreatedAt.IsZero() || cursor.Sort != sort {
		return "", nil, fmt.Errorf("invalid message cursor")
	}
	operator := "<"
	if sort == models.SortOldest {
		operator = ">"
	} else if sort != models.SortNewest {
		return "", nil, fmt.Errorf("invalid message cursor")
	}
	condition := fmt.Sprintf(
		"(conversation_messages.created_at, conversation_messages.id) %s ($%d::timestamptz, $%d)",
		operator,
		firstPlaceholder,
		firstPlaceholder+1,
	)
	return condition, []any{cursor.CreatedAt, cursor.ID}, nil
}

func encodeCursor(cursor any) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeCursor(raw string, cursor any) error {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cursor)
}

func nullTimePointer(value null.Time) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func conversationSortAt(result models.ConversationResult, sort models.Sort) *time.Time {
	if sort == models.SortStartedFirst || sort == models.SortStartedLast {
		return &result.CreatedAt
	}
	return nullTimePointer(result.LastMessageAt)
}

func trimResults[T any](results []T, pageSize int) ([]T, bool) {
	if len(results) <= pageSize {
		return results, false
	}
	return results[:pageSize], true
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
