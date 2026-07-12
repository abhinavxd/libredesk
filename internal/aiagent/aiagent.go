// Package aiagent runs autonomous AI assistants that reply to customers on conversations
// assigned to them, grounded only on the knowledge each assistant is given.
package aiagent

import (
	"database/sql"
	"embed"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai"
	"github.com/abhinavxd/libredesk/internal/aiagent/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const (
	// aiRunMaxSteps is the per-reply tool-calling budget, separate from the per-conversation max_turns cap.
	aiRunMaxSteps = 6
)

//go:embed queries.sql
var efs embed.FS

type queries struct {
	GetAssistants           *sqlx.Stmt `query:"get-assistants"`
	GetAssistant            *sqlx.Stmt `query:"get-assistant"`
	GetAssistantByUserID    *sqlx.Stmt `query:"get-assistant-by-user-id"`
	GetAssistantUserIDs     *sqlx.Stmt `query:"get-assistant-user-ids"`
	InsertAssistantUser     *sqlx.Stmt `query:"insert-assistant-user"`
	InsertAssistant         *sqlx.Stmt `query:"insert-assistant"`
	UpdateAssistant         *sqlx.Stmt `query:"update-assistant"`
	UpdateAssistantUser     *sqlx.Stmt `query:"update-assistant-user"`
	SoftDeleteAssistantUser *sqlx.Stmt `query:"soft-delete-assistant-user"`
	DeleteAssistant         *sqlx.Stmt `query:"delete-assistant"`
	GetAssistantTools       *sqlx.Stmt `query:"get-assistant-tools"`
	GetAllAssistantTools    *sqlx.Stmt `query:"get-all-assistant-tools"`
	DeleteAssistantTools    *sqlx.Stmt `query:"delete-assistant-tools"`
	InsertAssistantTool     *sqlx.Stmt `query:"insert-assistant-tool"`
	CountAITurns            *sqlx.Stmt `query:"count-ai-turns-since-assignment"`
	GetRecentContactConvos  *sqlx.Stmt `query:"get-recent-contact-conversations"`
	GetAssistantWindowStats *sqlx.Stmt `query:"get-assistant-window-stats"`
	GetAssistantExpectation *sqlx.Stmt `query:"get-assistant-expectation-by-user-id"`
	InsertAIAgentEvent      *sqlx.Stmt `query:"insert-ai-agent-event"`

	InsertFAQSuggestion           *sqlx.Stmt `query:"insert-faq-suggestion"`
	CountFAQByConversation        *sqlx.Stmt `query:"count-faq-suggestions-by-conversation"`
	GetFAQSuggestions             *sqlx.Stmt `query:"get-faq-suggestions"`
	GetFAQSuggestion              *sqlx.Stmt `query:"get-faq-suggestion"`
	UpdateFAQSuggestionStatus     *sqlx.Stmt `query:"update-faq-suggestion-status"`
	ApproveFAQSuggestionIfPending *sqlx.Stmt `query:"approve-faq-suggestion-if-pending"`
}

type Manager struct {
	q       queries
	db      *sqlx.DB
	lo      *logf.Logger
	i18n    *i18n.I18n
	ai      *ai.Manager
	convo   *conversation.Manager
	media   *media.Manager
	setting *setting.Manager

	queue    chan int
	inflight map[int]bool
	// pending marks a conversation that received a fresh event while its response was in flight, so
	// markDone re-enqueues it instead of dropping the follow-up.
	pending map[int]bool
	mu      sync.Mutex

	miningQueue    chan int
	miningInflight map[int]bool
	miningMu       sync.Mutex

	assistantUserIDs map[int]bool
	userIDsMu        sync.RWMutex
}

type Opts struct {
	DB        *sqlx.DB
	Lo        *logf.Logger
	I18n      *i18n.I18n
	QueueSize int
}

// New creates the AI agent manager.
func New(opts Opts, aiManager *ai.Manager, convo *conversation.Manager, mediaManager *media.Manager, settingManager *setting.Manager) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	m := &Manager{
		q:                q,
		db:               opts.DB,
		lo:               opts.Lo,
		i18n:             opts.I18n,
		ai:               aiManager,
		convo:            convo,
		media:            mediaManager,
		setting:          settingManager,
		queue:            make(chan int, opts.QueueSize),
		inflight:         map[int]bool{},
		pending:          map[int]bool{},
		miningQueue:      make(chan int, opts.QueueSize),
		miningInflight:   map[int]bool{},
		assistantUserIDs: map[int]bool{},
	}
	m.refreshAssistantUserIDs()
	return m, nil
}

// GetAssistants returns all assistants with their knowledge links.
func (m *Manager) GetAssistants() ([]models.Assistant, error) {
	assistants := []models.Assistant{}
	if err := m.q.GetAssistants.Select(&assistants); err != nil {
		m.lo.Error("error fetching assistants", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	toolsByAssistant, err := m.allAssistantTools()
	if err != nil {
		return nil, err
	}
	for i := range assistants {
		ids := toolsByAssistant[assistants[i].ID]
		if ids == nil {
			ids = []int{}
		}
		assistants[i].ToolIDs = ids
	}
	return assistants, nil
}

// allAssistantTools returns tool ids grouped by assistant id in a single query.
func (m *Manager) allAssistantTools() (map[int][]int, error) {
	var rows []struct {
		AssistantID int `db:"assistant_id"`
		ToolID      int `db:"tool_id"`
	}
	if err := m.q.GetAllAssistantTools.Select(&rows); err != nil {
		m.lo.Error("error fetching assistant tools", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	out := make(map[int][]int)
	for _, r := range rows {
		out[r.AssistantID] = append(out[r.AssistantID], r.ToolID)
	}
	return out, nil
}

// GetAssistant returns one assistant with its knowledge links.
func (m *Manager) GetAssistant(id int) (models.Assistant, error) {
	var a models.Assistant
	if err := m.q.GetAssistant.Get(&a, id); err != nil {
		if err == sql.ErrNoRows {
			return a, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching assistant", "error", err)
		return a, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	toolIDs, err := m.getTools(a.ID)
	if err != nil {
		return a, err
	}
	a.ToolIDs = toolIDs
	return a, nil
}

// GetAssistantStats returns windowed metrics over the last rangeDays with rates and trend deltas
// versus the previous equal-length window.
func (m *Manager) GetAssistantStats(id, rangeDays int) (models.AssistantStats, error) {
	a, err := m.GetAssistant(id)
	if err != nil {
		return models.AssistantStats{}, err
	}
	if rangeDays <= 0 {
		rangeDays = 30
	}
	if rangeDays > 90 {
		rangeDays = 90
	}
	now := time.Now()
	cur, err := m.statWindow(a, now.AddDate(0, 0, -rangeDays), now)
	if err != nil {
		return models.AssistantStats{}, err
	}
	prev, err := m.statWindow(a, now.AddDate(0, 0, -2*rangeDays), now.AddDate(0, 0, -rangeDays))
	if err != nil {
		return models.AssistantStats{}, err
	}
	return buildStats(rangeDays, cur, prev), nil
}

func (m *Manager) statWindow(a models.Assistant, start, end time.Time) (models.StatWindow, error) {
	var w models.StatWindow
	if err := m.q.GetAssistantWindowStats.Get(&w, a.UserID, a.ID, start, end); err != nil {
		m.lo.Error("error fetching assistant stats", "error", err)
		return w, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return w, nil
}

// recordEvent logs a handoff/resolve marker so rate and reopen metrics are queryable.
func (m *Manager) recordEvent(assistantID, conversationID int, eventType string) {
	if _, err := m.q.InsertAIAgentEvent.Exec(assistantID, conversationID, eventType); err != nil {
		m.lo.Error("error recording ai agent event", "type", eventType, "error", err)
	}
}

// GetAssistantByUserID resolves the assistant whose identity is the given user.
func (m *Manager) GetAssistantByUserID(userID int) (models.Assistant, error) {
	var a models.Assistant
	if err := m.q.GetAssistantByUserID.Get(&a, userID); err != nil {
		return a, err
	}
	toolIDs, err := m.getTools(a.ID)
	if err != nil {
		return a, err
	}
	a.ToolIDs = toolIDs
	return a, nil
}

// AssistantExpectation returns the assistant's widget note, empty if none.
func (m *Manager) AssistantExpectation(userID int) string {
	var expectation string
	if err := m.q.GetAssistantExpectation.Get(&expectation, userID); err != nil {
		return ""
	}
	return expectation
}

// CreateAssistant creates the assistant's identity user and config in one transaction.
func (m *Manager) CreateAssistant(a models.Assistant) (models.Assistant, error) {
	if err := m.validate(&a); err != nil {
		return models.Assistant{}, err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	var userID int
	if err := tx.Stmtx(m.q.InsertAssistantUser).QueryRow(a.Name).Scan(&userID); err != nil {
		m.lo.Error("error creating assistant user", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var id int
	if err := tx.Stmtx(m.q.InsertAssistant).QueryRow(userID, a.Description, a.Instructions, a.Guardrails, a.Tone, a.ResponseLength, a.MaxTurns, a.FallbackTeamID, a.Enabled, a.Expectation).Scan(&id); err != nil {
		m.lo.Error("error creating assistant", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := insertTools(tx, m.q.InsertAssistantTool, id, a.ToolIDs); err != nil {
		m.lo.Error("error linking assistant tools", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing assistant", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.refreshAssistantUserIDs()
	return m.GetAssistant(id)
}

// UpdateAssistant updates an assistant's config, identity user, and knowledge links.
func (m *Manager) UpdateAssistant(id int, a models.Assistant) (models.Assistant, error) {
	if err := m.validate(&a); err != nil {
		return models.Assistant{}, err
	}
	existing, err := m.GetAssistant(id)
	if err != nil {
		return models.Assistant{}, err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.UpdateAssistant).Exec(id, a.Description, a.Instructions, a.Guardrails, a.Tone, a.ResponseLength, a.MaxTurns, a.FallbackTeamID, a.Enabled, a.Expectation); err != nil {
		m.lo.Error("error updating assistant", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.UpdateAssistantUser).Exec(existing.UserID, a.Name); err != nil {
		m.lo.Error("error updating assistant user", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.DeleteAssistantTools).Exec(id); err != nil {
		m.lo.Error("error clearing assistant tools", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := insertTools(tx, m.q.InsertAssistantTool, id, a.ToolIDs); err != nil {
		m.lo.Error("error linking assistant tools", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing assistant update", "error", err)
		return models.Assistant{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.refreshAssistantUserIDs()
	return m.GetAssistant(id)
}

// DeleteAssistant removes the assistant config and soft-deletes its identity user so past messages keep its name.
func (m *Manager) DeleteAssistant(id int) error {
	a, err := m.GetAssistant(id)
	if err != nil {
		return err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.DeleteAssistant).Exec(id); err != nil {
		m.lo.Error("error deleting assistant", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.SoftDeleteAssistantUser).Exec(a.UserID); err != nil {
		m.lo.Error("error soft-deleting assistant user", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing assistant delete", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.refreshAssistantUserIDs()
	return nil
}

func (m *Manager) getTools(assistantID int) ([]int, error) {
	ids := []int{}
	if err := m.q.GetAssistantTools.Select(&ids, assistantID); err != nil {
		m.lo.Error("error fetching assistant tools", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return ids, nil
}

func (m *Manager) validate(a *models.Assistant) error {
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", m.i18n.T("globals.terms.name")), nil)
	}
	if a.Tone == "" {
		a.Tone = "professional"
	}
	if a.ResponseLength == "" {
		a.ResponseLength = "balanced"
	}
	if !slices.Contains(models.Tones, a.Tone) || !slices.Contains(models.ResponseLengths, a.ResponseLength) {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if a.MaxTurns <= 0 || a.MaxTurns > 20 {
		a.MaxTurns = 6
	}
	return nil
}

func (m *Manager) refreshAssistantUserIDs() {
	var ids []int
	if err := m.q.GetAssistantUserIDs.Select(&ids); err != nil {
		m.lo.Error("error loading assistant user ids", "error", err)
		return
	}
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	m.userIDsMu.Lock()
	m.assistantUserIDs = set
	m.userIDsMu.Unlock()
}

func (m *Manager) isAssistantUser(userID int) bool {
	m.userIDsMu.RLock()
	defer m.userIDsMu.RUnlock()
	return m.assistantUserIDs[userID]
}

func buildStats(rangeDays int, cur, prev models.StatWindow) models.AssistantStats {
	s := models.AssistantStats{
		RangeDays:      rangeDays,
		Conversations:  cur.Conversations,
		Replies:        cur.Replies,
		Handoffs:       cur.Handoffs,
		Resolved:       cur.Resolves,
		Reopened:       cur.Reopened,
		ResolutionRate: pct(cur.Resolves, cur.Conversations),
		HandoffRate:    pct(cur.Handoffs, cur.Conversations),
		ReopenRate:     pct(cur.Reopened, cur.Resolves),
		Depth:          round1(ratio(cur.Replies, cur.Conversations)),
		CSATCount:      cur.CSATCount,
		CSATAvg:        cur.CSATAvg,
		CSATPositive:   cur.CSATPositive,
	}
	s.Trends = map[string]float64{
		"conversations":   pctChange(cur.Conversations, prev.Conversations),
		"replies":         pctChange(cur.Replies, prev.Replies),
		"resolution_rate": round1(s.ResolutionRate - pct(prev.Resolves, prev.Conversations)),
		"handoff_rate":    round1(s.HandoffRate - pct(prev.Handoffs, prev.Conversations)),
		"reopen_rate":     round1(s.ReopenRate - pct(prev.Reopened, prev.Resolves)),
		"csat_avg":        round1(cur.CSATAvg - prev.CSATAvg),
	}
	return s
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return round1(float64(n) / float64(d) * 100)
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func pctChange(cur, prev int) float64 {
	if prev == 0 {
		return 0
	}
	return round1(float64(cur-prev) / float64(prev) * 100)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func insertTools(tx *sqlx.Tx, stmt *sqlx.Stmt, assistantID int, toolIDs []int) error {
	for _, id := range toolIDs {
		if id == 0 {
			continue
		}
		if _, err := tx.Stmtx(stmt).Exec(assistantID, id); err != nil {
			return err
		}
	}
	return nil
}
