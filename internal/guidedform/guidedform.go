// Package guidedform runs guided, branching pre-chat question flows: a bot identity asks
// questions as ordinary chat messages, matches the visitor's answers against configured
// branches (regex), and once the flow completes hands the conversation off to an AI assistant,
// a team, or the unassigned queue - mirroring how internal/aiagent runs autonomous replies.
package guidedform

import (
	"database/sql"
	"embed"
	"strings"
	"sync"

	"github.com/abhinavxd/libredesk/internal/conversation"
	customAttribute "github.com/abhinavxd/libredesk/internal/custom_attribute"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	gmodels "github.com/abhinavxd/libredesk/internal/guidedform/models"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

//go:embed queries.sql
var efs embed.FS

type queries struct {
	GetForms              *sqlx.Stmt `query:"get-forms"`
	GetForm               *sqlx.Stmt `query:"get-form"`
	GetFormByUserID       *sqlx.Stmt `query:"get-form-by-user-id"`
	GetFormByInboxID      *sqlx.Stmt `query:"get-form-by-inbox-id"`
	GetFormBotUserIDs     *sqlx.Stmt `query:"get-form-bot-user-ids"`
	InsertFormUser        *sqlx.Stmt `query:"insert-form-user"`
	UpdateFormUser        *sqlx.Stmt `query:"update-form-user"`
	SoftDeleteFormUser    *sqlx.Stmt `query:"soft-delete-form-user"`
	InsertForm            *sqlx.Stmt `query:"insert-form"`
	UpdateForm            *sqlx.Stmt `query:"update-form"`
	DeleteForm            *sqlx.Stmt `query:"delete-form"`
	DisableOtherFormsOnInbox *sqlx.Stmt `query:"disable-other-forms-on-inbox"`
	UnassignFormBotConvos *sqlx.Stmt `query:"unassign-form-bot-conversations"`
	InsertGuidedFormEvent *sqlx.Stmt `query:"insert-guided-form-event"`
}

// Manager owns guided form configuration and runs the question/answer flow for conversations
// assigned to a guided-form bot identity.
type Manager struct {
	q               queries
	db              *sqlx.DB
	lo              *logf.Logger
	i18n            *i18n.I18n
	convo           *conversation.Manager
	customAttribute *customAttribute.Manager
	user            *user.Manager

	botUserIDs map[int]bool
	mu         sync.RWMutex
}

// Opts holds the options required to construct the guided form manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// New creates the guided form manager.
func New(opts Opts, convo *conversation.Manager, customAttributeManager *customAttribute.Manager, userManager *user.Manager) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	m := &Manager{
		q:               q,
		db:              opts.DB,
		lo:              opts.Lo,
		i18n:            opts.I18n,
		convo:           convo,
		customAttribute: customAttributeManager,
		user:            userManager,
		botUserIDs:      map[int]bool{},
	}
	if err := m.refreshBotUserIDs(); err != nil {
		return nil, err
	}
	return m, nil
}

// GetForms returns all guided forms.
func (m *Manager) GetForms() ([]gmodels.Form, error) {
	var rows []gmodels.Form
	if err := m.q.GetForms.Select(&rows); err != nil {
		m.lo.Error("error fetching guided forms", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for i := range rows {
		if err := rows[i].UnmarshalSteps(); err != nil {
			m.lo.Error("error unmarshalling guided form steps", "form_id", rows[i].ID, "error", err)
		}
	}
	return rows, nil
}

// GetForm returns one guided form by id.
func (m *Manager) GetForm(id int) (gmodels.Form, error) {
	return m.getFormWith(m.q.GetForm, id)
}

// GetFormByUserID resolves the guided form whose bot identity is the given user.
func (m *Manager) GetFormByUserID(userID int) (gmodels.Form, error) {
	return m.getFormWith(m.q.GetFormByUserID, userID)
}

// GetFormByInboxID returns the enabled guided form configured for an inbox, if any.
func (m *Manager) GetFormByInboxID(inboxID int) (gmodels.Form, error) {
	return m.getFormWith(m.q.GetFormByInboxID, inboxID)
}

func (m *Manager) getFormWith(stmt *sqlx.Stmt, arg int) (gmodels.Form, error) {
	var f gmodels.Form
	if err := stmt.Get(&f, arg); err != nil {
		if err == sql.ErrNoRows {
			return f, err
		}
		m.lo.Error("error fetching guided form", "error", err)
		return f, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := f.UnmarshalSteps(); err != nil {
		m.lo.Error("error unmarshalling guided form steps", "form_id", f.ID, "error", err)
	}
	return f, nil
}

// CreateForm creates the bot identity user and the form config in one transaction.
func (m *Manager) CreateForm(f gmodels.Form) (gmodels.Form, error) {
	if err := m.validate(&f); err != nil {
		return gmodels.Form{}, err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if f.Enabled {
		if _, err := tx.Stmtx(m.q.DisableOtherFormsOnInbox).Exec(f.InboxID, 0); err != nil {
			m.lo.Error("error disabling other guided forms on inbox", "inbox_id", f.InboxID, "error", err)
			return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}

	var userID int
	if err := tx.Stmtx(m.q.InsertFormUser).QueryRow(f.Name).Scan(&userID); err != nil {
		m.lo.Error("error creating guided form bot user", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var id int
	if err := tx.Stmtx(m.q.InsertForm).QueryRow(userID, f.Name, f.InboxID, f.Enabled, f.StartStepID, f.StepsRaw, f.OnCompleteAction, f.OnCompleteAssistantID, f.OnCompleteTeamID, f.CompletionMessage).Scan(&id); err != nil {
		m.lo.Error("error creating guided form", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing guided form", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.refreshBotUserIDs(); err != nil {
		m.lo.Error("error refreshing guided form bot user ids cache", "error", err)
	}
	return m.GetForm(id)
}

// UpdateForm updates an existing guided form's config and bot identity name.
func (m *Manager) UpdateForm(id int, f gmodels.Form) (gmodels.Form, error) {
	if err := m.validate(&f); err != nil {
		return gmodels.Form{}, err
	}
	existing, err := m.GetForm(id)
	if err != nil {
		return gmodels.Form{}, err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if f.Enabled {
		if _, err := tx.Stmtx(m.q.DisableOtherFormsOnInbox).Exec(f.InboxID, id); err != nil {
			m.lo.Error("error disabling other guided forms on inbox", "inbox_id", f.InboxID, "error", err)
			return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}

	if _, err := tx.Stmtx(m.q.UpdateForm).Exec(id, f.Name, f.InboxID, f.Enabled, f.StartStepID, f.StepsRaw, f.OnCompleteAction, f.OnCompleteAssistantID, f.OnCompleteTeamID, f.CompletionMessage); err != nil {
		m.lo.Error("error updating guided form", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.UpdateFormUser).Exec(existing.UserID, f.Name); err != nil {
		m.lo.Error("error updating guided form bot user", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing guided form update", "error", err)
		return gmodels.Form{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return m.GetForm(id)
}

// DeleteForm removes the form config and soft-deletes its bot identity user, moving any
// in-flight conversations to the form's fallback team (or unassigning them).
func (m *Manager) DeleteForm(id int) (int, error) {
	f, err := m.GetForm(id)
	if err != nil {
		return 0, err
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting transaction", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.DeleteForm).Exec(id); err != nil {
		m.lo.Error("error deleting guided form", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.SoftDeleteFormUser).Exec(f.UserID); err != nil {
		m.lo.Error("error soft-deleting guided form bot user", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.UnassignFormBotConvos).Exec(f.UserID, f.OnCompleteTeamID); err != nil {
		m.lo.Error("error unassigning deleted guided form conversations", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing guided form delete", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.refreshBotUserIDs(); err != nil {
		m.lo.Error("error refreshing guided form bot user ids cache", "error", err)
	}
	return f.UserID, nil
}

func (m *Manager) validate(f *gmodels.Form) error {
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", m.i18n.T("globals.terms.name")), nil)
	}
	if f.InboxID == 0 {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if len(f.Steps) == 0 {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	ids := map[string]bool{}
	for _, s := range f.Steps {
		if s.ID == "" || s.Question == "" {
			return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		ids[s.ID] = true
	}
	if f.StartStepID == "" {
		f.StartStepID = f.Steps[0].ID
	}
	if !ids[f.StartStepID] {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for _, s := range f.Steps {
		if s.DefaultNextStepID != "" && !ids[s.DefaultNextStepID] {
			return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		for _, b := range s.Branches {
			if !ids[b.NextStepID] {
				return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			if _, err := compileBranchPattern(b.Pattern); err != nil {
				return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.invalidFields", "name", "branch pattern"), nil)
			}
		}
	}
	if err := m.validateGraph(*f); err != nil {
		return err
	}
	switch f.OnCompleteAction {
	case gmodels.CompleteActionTeam:
		if !f.OnCompleteTeamID.Valid {
			return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	case gmodels.CompleteActionAssistant:
		if !f.OnCompleteAssistantID.Valid {
			return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	case gmodels.CompleteActionUnassign:
		// No target required.
	default:
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := f.MarshalSteps(); err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// validateGraph checks the step transition graph (branches, explicit defaults, and the
// natural fall-through-to-next-in-order edge) for two mistakes that are easy to make by hand
// and otherwise only surface as a broken conversation later: a step that can never be reached
// from the start step, and a cycle that would loop forever without ever completing the form.
func (m *Manager) validateGraph(f gmodels.Form) error {
	edges := make(map[string][]string, len(f.Steps))
	for i, s := range f.Steps {
		var targets []string
		for _, b := range s.Branches {
			targets = append(targets, b.NextStepID)
		}
		if s.DefaultNextStepID != "" {
			targets = append(targets, s.DefaultNextStepID)
		} else if !s.EndsForm && i+1 < len(f.Steps) {
			targets = append(targets, f.Steps[i+1].ID)
		}
		edges[s.ID] = targets
	}

	// Reachability: BFS from the start step over every edge.
	reachable := map[string]bool{f.StartStepID: true}
	queue := []string{f.StartStepID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, next := range edges[id] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}
	for _, s := range f.Steps {
		if !reachable[s.ID] {
			return envelope.NewError(envelope.InputError, m.i18n.Ts("admin.guidedForms.errors.unreachableStep", "step", s.ID), nil)
		}
	}

	// Cycle detection: DFS with a recursion-stack marker, following the same edges.
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[string]int, len(f.Steps))
	var stack []string
	var dfs func(id string) error
	dfs = func(id string) error {
		state[id] = visiting
		stack = append(stack, id)
		for _, next := range edges[id] {
			switch state[next] {
			case visiting:
				loop := append(append([]string{}, stack...), next)
				return envelope.NewError(envelope.InputError, m.i18n.Ts("admin.guidedForms.errors.cycle", "path", strings.Join(loop, " -> ")), nil)
			case unvisited:
				if err := dfs(next); err != nil {
					return err
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
		return nil
	}
	for _, s := range f.Steps {
		if state[s.ID] == unvisited {
			if err := dfs(s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) refreshBotUserIDs() error {
	var ids []int
	if err := m.q.GetFormBotUserIDs.Select(&ids); err != nil {
		m.lo.Error("error loading guided form bot user ids", "error", err)
		return err
	}
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	m.mu.Lock()
	m.botUserIDs = set
	m.mu.Unlock()
	return nil
}

func (m *Manager) isFormBotUser(userID int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.botUserIDs[userID]
}

func (m *Manager) recordEvent(formID, conversationID int, eventType string) {
	if _, err := m.q.InsertGuidedFormEvent.Exec(formID, conversationID, eventType); err != nil {
		m.lo.Error("error recording guided form event", "type", eventType, "error", err)
	}
}
