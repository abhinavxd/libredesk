// Package automation automatically evaluates and applies rules to conversations based on events like new conversations, updates, and time triggers,
// and performs some actions if they are true.
package automation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
	// MaxQueueSize defines the maximum size of the task queues.
	MaxQueueSize = 5000
)

// TaskType represents the type of conversation task.
type TaskType string

const (
	NewConversation    TaskType = "new"
	UpdateConversation TaskType = "update"
	TimeTrigger        TaskType = "time-trigger"
)

// ConversationTask represents a unit of work for processing conversations.
type ConversationTask struct {
	taskType         TaskType
	eventType        string
	conversationUUID string
}

type Engine struct {
	rules             []models.Rule
	rulesMu           sync.RWMutex
	q                 queries
	lo                *logf.Logger
	i18n              *i18n.I18n
	conversationStore conversationStore
	taskQueue         chan ConversationTask
	closed            bool
	closedMu          sync.RWMutex
	wg                sync.WaitGroup
}

type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

type conversationStore interface {
	ApplyAction(action models.RuleAction, conversation cmodels.Conversation, user umodels.User) error
	GetConversation(teamID int, uuid string) (cmodels.Conversation, error)
	GetConversationsCreatedAfter(time.Time) ([]cmodels.Conversation, error)
}

type queries struct {
	GetAll                  *sqlx.Stmt `query:"get-all"`
	GetRule                 *sqlx.Stmt `query:"get-rule"`
	InsertRule              *sqlx.Stmt `query:"insert-rule"`
	UpdateRule              *sqlx.Stmt `query:"update-rule"`
	DeleteRule              *sqlx.Stmt `query:"delete-rule"`
	ToggleRule              *sqlx.Stmt `query:"toggle-rule"`
	GetEnabledRules         *sqlx.Stmt `query:"get-enabled-rules"`
	UpdateRuleWeight        *sqlx.Stmt `query:"update-rule-weight"`
	UpdateRuleExecutionMode *sqlx.Stmt `query:"update-rule-execution-mode"`
}

// New initializes a new Engine.
func New(opt Opts) (*Engine, error) {
	var (
		q queries
		e = &Engine{
			lo:        opt.Lo,
			i18n:      opt.I18n,
			taskQueue: make(chan ConversationTask, MaxQueueSize),
		}
	)
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	e.q = q
	e.rules = e.queryRules()
	return e, nil
}

// SetConversationStore sets conversations store.
func (e *Engine) SetConversationStore(store conversationStore) {
	e.conversationStore = store
}

// ReloadRules reloads automation rules from DB.
func (e *Engine) ReloadRules() {
	e.rulesMu.Lock()
	defer e.rulesMu.Unlock()
	e.lo.Debug("reloading automation engine rules")
	e.rules = e.queryRules()
}

// Run starts the Engine with a worker pool to evaluate rules based on events.
func (e *Engine) Run(ctx context.Context, workerCount int) {
	// Spawn worker pool.
	for i := 0; i < workerCount; i++ {
		e.wg.Add(1)
		go e.worker(ctx)
	}

	// Hourly ticker for timed triggers.
	ticker := time.NewTicker(1 * time.Hour)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.lo.Info("queuing time triggers")
			e.taskQueue <- ConversationTask{taskType: TimeTrigger}
		}
	}
}

// worker processes tasks from the taskQueue until it's closed or context is done.
func (e *Engine) worker(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-e.taskQueue:
			if !ok {
				return
			}
			switch task.taskType {
			case NewConversation:
				e.handleNewConversation(task.conversationUUID)
			case UpdateConversation:
				e.handleUpdateConversation(task.conversationUUID, task.eventType)
			case TimeTrigger:
				e.handleTimeTrigger()
			}
		}
	}
}

// Close signals the Engine to stop accepting any more messages and waits for all workers to finish.
func (e *Engine) Close() {
	e.closedMu.Lock()
	defer e.closedMu.Unlock()
	if e.closed {
		return
	}
	e.closed = true
	close(e.taskQueue)
	// Wait for all workers.
	e.wg.Wait()
}

// GetAllRules retrieves all rules of a specific type.
func (e *Engine) GetAllRules(typ []byte) ([]models.RuleRecord, error) {
	var rules = make([]models.RuleRecord, 0)
	if err := e.q.GetAll.Select(&rules, typ); err != nil {
		e.lo.Error("error fetching rules", "error", err)
		return rules, envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorFetching", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	return rules, nil
}

// GetRule retrieves a rule by ID.
func (e *Engine) GetRule(id int) (models.RuleRecord, error) {
	var rule models.RuleRecord
	if err := e.q.GetRule.Get(&rule, id); err != nil {
		if err == sql.ErrNoRows {
			return rule, envelope.NewError(envelope.InputError, e.i18n.Ts("globals.messages.notFound", "name", e.i18n.Ts("globals.entities.rule")), nil)
		}
		e.lo.Error("error fetching rule", "error", err)
		return rule, envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorFetching", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	return rule, nil
}

// ToggleRule toggles the active status of a rule by ID.
func (e *Engine) ToggleRule(id int) error {
	if _, err := e.q.ToggleRule.Exec(id); err != nil {
		e.lo.Error("error toggling rule", "error", err)
		return envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorUpdating", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// UpdateRule updates an existing rule.
func (e *Engine) UpdateRule(id int, rule models.RuleRecord) error {
	if rule.Events == nil {
		rule.Events = pq.StringArray{}
	}
	if _, err := e.q.UpdateRule.Exec(id, rule.Name, rule.Description, rule.Type, rule.Events, rule.Rules, rule.Enabled); err != nil {
		e.lo.Error("error updating rule", "error", err)
		return envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorUpdating", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// CreateRule creates a new rule.
func (e *Engine) CreateRule(rule models.RuleRecord) error {
	if rule.Events == nil {
		rule.Events = pq.StringArray{}
	}
	if _, err := e.q.InsertRule.Exec(rule.Name, rule.Description, rule.Type, rule.Events, rule.Rules); err != nil {
		e.lo.Error("error creating rule", "error", err)
		return envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorCreating", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// DeleteRule deletes a rule by ID.
func (e *Engine) DeleteRule(id int) error {
	if _, err := e.q.DeleteRule.Exec(id); err != nil {
		e.lo.Error("error deleting rule", "error", err)
		return envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorDeleting", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// UpdateRuleWeights updates the weights of the automation rules.
func (e *Engine) UpdateRuleWeights(weights map[int]int) error {
	for id, weight := range weights {
		if _, err := e.q.UpdateRuleWeight.Exec(id, weight); err != nil {
			e.lo.Error("error updating rule weight", "error", err)
			return envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorUpdating", "name", e.i18n.Ts("globals.entities.rule")), nil)
		}
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// UpdateRuleExecutionMode updates the execution mode for a type of rule.
func (e *Engine) UpdateRuleExecutionMode(ruleType, mode string) error {
	if _, err := e.q.UpdateRuleExecutionMode.Exec(ruleType, mode); err != nil {
		e.lo.Error("error updating rule execution mode", "error", err)
		return envelope.NewError(envelope.GeneralError, e.i18n.Ts("globals.messages.errorUpdating", "name", e.i18n.Ts("globals.entities.rule")), nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// EvaluateNewConversationRules enqueues a new conversation for rule evaluation.
func (e *Engine) EvaluateNewConversationRules(conversationUUID string) {
	e.closedMu.RLock()
	defer e.closedMu.RUnlock()
	if e.closed {
		return
	}
	select {
	case e.taskQueue <- ConversationTask{
		taskType:         NewConversation,
		conversationUUID: conversationUUID,
	}:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateNewConversationRules: newConversationQ is full, unable to enqueue conversation")
	}
}

// EvaluateConversationUpdateRules enqueues an updated conversation for rule evaluation.
func (e *Engine) EvaluateConversationUpdateRules(conversationUUID string, eventType string) {
	if eventType == "" {
		e.lo.Error("error evaluating conversation update rules: eventType is empty")
		return
	}
	e.closedMu.RLock()
	defer e.closedMu.RUnlock()
	if e.closed {
		return
	}
	select {
	case e.taskQueue <- ConversationTask{
		taskType:         UpdateConversation,
		conversationUUID: conversationUUID,
		eventType:        eventType,
	}:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateConversationUpdateRules: updateConversationQ is full, unable to enqueue conversation")
	}
}

// handleNewConversation handles new conversation events.
func (e *Engine) handleNewConversation(conversationUUID string) {
	e.lo.Debug("handling new conversation", "uuid", conversationUUID)
	conversation, err := e.conversationStore.GetConversation(0, conversationUUID)
	if err != nil {
		e.lo.Error("error fetching conversation for new event", "uuid", conversationUUID, "error", err)
		return
	}
	rules := e.filterRulesByType(models.RuleTypeNewConversation, "")
	if len(rules) == 0 {
		e.lo.Warn("no rules to evaluate for new conversation", "uuid", conversationUUID)
		return
	}
	e.evalConversationRules(rules, conversation)
}

// handleUpdateConversation handles update conversation events with specific eventType.
func (e *Engine) handleUpdateConversation(conversationUUID, eventType string) {
	e.lo.Debug("handling update conversation", "uuid", conversationUUID, "event_type", eventType)
	conversation, err := e.conversationStore.GetConversation(0, conversationUUID)
	if err != nil {
		e.lo.Error("error fetching conversation for update event", "uuid", conversationUUID, "error", err)
		return
	}
	rules := e.filterRulesByType(models.RuleTypeConversationUpdate, eventType)
	if len(rules) == 0 {
		e.lo.Warn("no rules to evaluate for conversation update", "uuid", conversationUUID, "event_type", eventType)
		return
	}
	e.evalConversationRules(rules, conversation)
}

// handleTimeTrigger handles time trigger events.
func (e *Engine) handleTimeTrigger() {
	e.lo.Debug("handling time triggers")
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	conversations, err := e.conversationStore.GetConversationsCreatedAfter(thirtyDaysAgo)
	if err != nil {
		e.lo.Error("error fetching conversations for time trigger", "error", err)
		return
	}
	rules := e.filterRulesByType(models.RuleTypeTimeTrigger, "")
	if len(rules) == 0 {
		e.lo.Warn("no rules to evaluate for time trigger")
		return
	}
	e.lo.Debug("fetched conversations for evaluating time triggers", "conversations_count", len(conversations), "rules_count", len(rules))
	for _, c := range conversations {
		// Fetch entire conversation.
		conversation, err := e.conversationStore.GetConversation(0, c.UUID)
		if err != nil {
			e.lo.Error("error fetching conversation for time trigger", "uuid", c.UUID, "error", err)
			continue
		}
		e.evalConversationRules(rules, conversation)
	}
}

// queryRules fetches automation rules from the database.
func (e *Engine) queryRules() []models.Rule {
	var (
		rules         []models.RuleRecord
		filteredRules []models.Rule
	)
	err := e.q.GetEnabledRules.Select(&rules)
	if err != nil {
		e.lo.Error("error fetching automation rules", "error", err)
		return filteredRules
	}

	for _, rule := range rules {
		var rulesBatch []models.Rule
		if err := json.Unmarshal([]byte(rule.Rules), &rulesBatch); err != nil {
			e.lo.Error("error unmarshalling rule JSON", "error", err)
			continue
		}
		// Set values from DB.
		for i := range rulesBatch {
			rulesBatch[i].Type = rule.Type
			rulesBatch[i].Events = rule.Events
			rulesBatch[i].ExecutionMode = rule.ExecutionMode
		}
		filteredRules = append(filteredRules, rulesBatch...)
	}
	return filteredRules
}

// filterRulesByType filters rules by type and event.
func (e *Engine) filterRulesByType(ruleType, eventType string) []models.Rule {
	e.rulesMu.RLock()
	defer e.rulesMu.RUnlock()

	var filteredRules []models.Rule
	for _, rule := range e.rules {
		if rule.Type == ruleType && (eventType == "" || slices.Contains(rule.Events, eventType)) {
			filteredRules = append(filteredRules, rule)
		}
	}
	return filteredRules
}
