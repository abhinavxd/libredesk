package sla

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/sla/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type stubTeamStore struct{}

type stubUserStore struct{}

type stubAppSettingsStore struct{}

type stubBusinessHrsStore struct{ bh bmodels.BusinessHours }

type appliedRow struct {
	ID          int          `db:"id"`
	Status      string       `db:"status"`
	FRDeadline  sql.NullTime `db:"first_response_deadline_at"`
	ResDeadline sql.NullTime `db:"resolution_deadline_at"`
	FRMetAt     sql.NullTime `db:"first_response_met_at"`
	FRBreached  sql.NullTime `db:"first_response_breached_at"`
	ResMetAt    sql.NullTime `db:"resolution_met_at"`
	ResBreached sql.NullTime `db:"resolution_breached_at"`
}

func (stubTeamStore) Get(id int) (tmodels.Team, error) { return tmodels.Team{}, nil }

func (stubUserStore) GetAgent(int, string) (umodels.User, error) { return umodels.User{}, nil }

func (stubAppSettingsStore) GetByPrefix(prefix string) (types.JSONText, error) {
	return types.JSONText(`{"app.business_hours_id":"1","app.timezone":"UTC"}`), nil
}

func (s stubBusinessHrsStore) Get(id int) (bmodels.BusinessHours, error) {
	return s.bh, nil
}

func TestApplySLASetsDeadlinesAndConversation(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, policy)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 1 || rows[0].Status != "pending" {
		t.Fatalf("expected one pending applied sla, got %+v", rows)
	}
	if !rows[0].FRDeadline.Valid || !rows[0].ResDeadline.Valid {
		t.Fatalf("expected both deadlines set, got %+v", rows[0])
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || !d.Time.Equal(rows[0].FRDeadline.Time) {
		t.Fatalf("expected conversation deadline = first response deadline, got %v vs %v", d, rows[0].FRDeadline)
	}
}

func TestApplySLAReplacesUntouchedPending(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "1h", "2h", "")
	p2 := insertPolicy(t, db, "p2", "3h", "4h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 1 || rows[0].Status != "pending" {
		t.Fatalf("expected old untouched sla deleted and one pending row, got %+v", rows)
	}
}

func TestApplySLAClosesSettledPendingAndCleansChildren(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	p2 := insertPolicy(t, db, "p2", "3h", "4h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	old := fetchApplied(t, db, conv)[0]
	db.MustExec(`UPDATE applied_slas SET first_response_met_at = NOW() WHERE id = $1`, old.ID)
	db.MustExec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at, status) VALUES ($1, $2, 'next_response', NOW() + INTERVAL '30 min', 'pending')`, old.ID, p1)
	db.MustExec(`INSERT INTO scheduled_sla_notifications (applied_sla_id, metric, notification_type, recipients, send_at) VALUES ($1, 'resolution', 'warning', '{1}', NOW() + INTERVAL '2h')`, old.ID)

	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 {
		t.Fatalf("expected settled sla kept plus new pending, got %+v", rows)
	}
	if rows[0].ID != old.ID || rows[0].Status == "pending" {
		t.Fatalf("expected old sla closed, got %+v", rows[0])
	}
	if rows[1].Status != "pending" {
		t.Fatalf("expected new pending sla, got %+v", rows[1])
	}
	events := queryInt(t, db, `SELECT COUNT(*) FROM sla_events WHERE applied_sla_id = $1 AND status = 'pending'`, old.ID)
	notifs := queryInt(t, db, `SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND processed_at IS NULL`, old.ID)
	if events != 0 || notifs != 0 {
		t.Fatalf("expected superseded sla children cleaned, got %d events and %d notifications", events, notifs)
	}
}

func TestApplySLAJudgesUnstampedBreachBeforeSupersede(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "30m", "", "")
	p2 := insertPolicy(t, db, "p2", "3h", "", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	old := fetchApplied(t, db, conv)[0]
	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '10 min' WHERE id = $1`, old.ID)

	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 || rows[0].ID != old.ID {
		t.Fatalf("expected old sla kept plus new pending, got %+v", rows)
	}
	if rows[0].Status != "breached" || !rows[0].FRBreached.Valid {
		t.Fatalf("expected incurred breach stamped and status breached, got %+v", rows[0])
	}
}

func TestApplySLAJudgesUnstampedReplyBeforeSupersede(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "1h", "", "")
	p2 := insertPolicy(t, db, "p2", "5m", "", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	old := fetchApplied(t, db, conv)[0]
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 || rows[0].ID != old.ID {
		t.Fatalf("expected old sla kept plus new pending, got %+v", rows)
	}
	if rows[0].Status != "met" || !rows[0].FRMetAt.Valid {
		t.Fatalf("expected reply stamped met and status met, got %+v", rows[0])
	}
}

func TestApplySLAKeepsOverdueCountdownAsBreach(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "", "", "30m")
	p2 := insertPolicy(t, db, "p2", "1h", "", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	old := fetchApplied(t, db, conv)[0]
	db.MustExec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at, status) VALUES ($1, $2, 'next_response', NOW() - INTERVAL '5 min', 'pending')`, old.ID, p1)

	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 || rows[0].ID != old.ID {
		t.Fatalf("expected old sla kept plus new pending, got %+v", rows)
	}
	if rows[0].Status != "breached" {
		t.Fatalf("expected overdue countdown scored as breach, got %+v", rows[0])
	}
	events := queryInt(t, db, `SELECT COUNT(*) FROM sla_events WHERE applied_sla_id = $1 AND status = 'pending' AND breached_at IS NULL`, old.ID)
	if events != 1 {
		t.Fatalf("expected overdue countdown kept for the tick, got %d", events)
	}
}

func TestEvaluateMetAndClose(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW(), resolved_at = NOW() WHERE id = $1`, conv)
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "met" || !row.FRMetAt.Valid || !row.ResMetAt.Valid {
		t.Fatalf("expected sla met and closed, got %+v", row)
	}
}

func TestEvaluateBreachAndClose(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '2h', resolution_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "breached" || !row.FRBreached.Valid || !row.ResBreached.Valid {
		t.Fatalf("expected sla breached and closed, got %+v", row)
	}
}

func TestEvaluatePartiallyMet(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET resolution_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "partially_met" || !row.FRMetAt.Valid || !row.ResBreached.Valid {
		t.Fatalf("expected partially met, got %+v", row)
	}
}

func TestEvaluateLeavesUnsettledPending(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "pending" || !row.FRMetAt.Valid || row.ResMetAt.Valid {
		t.Fatalf("expected first response met but sla still pending, got %+v", row)
	}
}

func TestEvaluateSingleMetricPolicy(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "fr-only", "1h", "", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "met" || !row.FRMetAt.Valid {
		t.Fatalf("expected first-response-only sla closed as met, got %+v", row)
	}
}

func TestEvaluateResolutionOnlyPolicy(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "res-only", "", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE conversations SET resolved_at = NOW() WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "met" || !row.ResMetAt.Valid || row.FRMetAt.Valid {
		t.Fatalf("expected resolution-only sla closed as met, got %+v", row)
	}
}

func TestSweepSkipsNextResponseOnlyPolicy(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "nr-only", "", "", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	deadline, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0)
	if err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "pending" {
		t.Fatalf("expected next-response-only sla to stay pending, got %+v", row)
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || d.Time.Sub(deadline).Abs() > time.Millisecond {
		t.Fatalf("expected next response deadline kept on conversation, got %v want %v", d, deadline)
	}
}

func TestBreachSchedulesNotification(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	db.MustExec(`UPDATE sla_policies SET notifications = '[{"type":"breach","time_delay_type":"immediately","time_delay":"","recipients":["assigned_user"]}]' WHERE id = $1`, policy)
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	n := queryInt(t, db, `SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND notification_type = 'breach'`, fetchApplied(t, db, conv)[0].ID)
	if n != 1 {
		t.Fatalf("expected one scheduled breach notification, got %d", n)
	}
}

func TestNextResponseEventLifecycle(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != ErrUnmetSLAEventAlreadyExists {
		t.Fatalf("expected duplicate unmet event to be rejected, got %v", err)
	}
	if _, err := m.SetLatestSLAEventMetAt(appliedID, MetricNextResponse); err != nil {
		t.Fatalf("SetLatestSLAEventMetAt: %v", err)
	}
	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}

	if status := queryStr(t, db, `SELECT status FROM sla_events WHERE applied_sla_id = $1`, appliedID); status != "met" {
		t.Fatalf("expected event met, got %s", status)
	}
	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("expected new event allowed after previous met, got %v", err)
	}
}

func TestNextResponseEventBreachSchedulesNotification(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	db.MustExec(`UPDATE sla_policies SET notifications = '[{"type":"breach","time_delay_type":"immediately","time_delay":"","recipients":["assigned_user"]}]' WHERE id = $1`, policy)
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	db.MustExec(`UPDATE sla_events SET deadline_at = NOW() - INTERVAL '1h' WHERE applied_sla_id = $1`, appliedID)
	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}

	if status := queryStr(t, db, `SELECT status FROM sla_events WHERE applied_sla_id = $1`, appliedID); status != "breached" {
		t.Fatalf("expected event breached, got %s", status)
	}
	n := queryInt(t, db, `SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND metric = 'next_response' AND notification_type = 'breach'`, appliedID)
	if n != 1 {
		t.Fatalf("expected one scheduled next-response breach notification, got %d", n)
	}
}

func TestDeletePolicyClearsConversationDeadline(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	if d := conversationDeadline(t, db, conv); !d.Valid {
		t.Fatal("expected conversation deadline set after applying the policy")
	}

	if err := m.Delete(policy); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n := queryInt(t, db, `SELECT COUNT(*) FROM applied_slas WHERE conversation_id = $1`, conv); n != 0 {
		t.Fatalf("expected applied slas to cascade away, got %d", n)
	}
	if d := conversationDeadline(t, db, conv); d.Valid {
		t.Fatalf("expected conversation deadline cleared after policy delete, got %v", d.Time)
	}
}

func TestNextResponseMetClearsConversationDeadline(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if d := conversationDeadline(t, db, conv); !d.Valid {
		t.Fatal("expected conversation deadline to track the pending next-response event")
	}

	if _, err := m.SetLatestSLAEventMetAt(appliedID, MetricNextResponse); err != nil {
		t.Fatalf("SetLatestSLAEventMetAt: %v", err)
	}
	if d := conversationDeadline(t, db, conv); d.Valid {
		t.Fatalf("expected conversation deadline cleared once next response was met, got %v", d.Time)
	}
}

func TestNextResponseBreachClearsConversationDeadline(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)

	if _, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	db.MustExec(`UPDATE sla_events SET deadline_at = NOW() - INTERVAL '1h' WHERE applied_sla_id = $1`, appliedID)

	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}
	if d := conversationDeadline(t, db, conv); d.Valid {
		t.Fatalf("expected conversation deadline cleared once next response breached, got %v", d.Time)
	}
}

func TestSendNotificationSkipsMetMetric(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID
	db.MustExec(`UPDATE applied_slas SET first_response_met_at = NOW() WHERE id = $1`, appliedID)
	db.MustExec(`INSERT INTO scheduled_sla_notifications (applied_sla_id, metric, notification_type, recipients, send_at) VALUES ($1, 'first_response', 'breach', '{1}', NOW() - INTERVAL '1 min')`, appliedID)

	var pending []models.ScheduledSLANotification
	if err := m.q.GetScheduledSLANotifications.Select(&pending); err != nil {
		t.Fatalf("fetching scheduled notifications: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected one due notification, got %d", len(pending))
	}
	if err := m.SendNotification(pending[0]); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	if queryInt(t, db, `SELECT COUNT(*) FROM scheduled_sla_notifications WHERE id = $1 AND processed_at IS NOT NULL`, pending[0].ID) != 1 {
		t.Fatal("expected met-metric notification marked processed without sending")
	}
}

func TestPolicyMetricCombinations(t *testing.T) {
	cases := []struct {
		name           string
		fr, res, nr    string
		wantStatus     string
		wantRowsAfter  int
		settledOldGone bool
	}{
		{"fr-only", "1h", "", "", "met", 2, false},
		{"res-only", "", "2h", "", "met", 2, false},
		{"nr-only", "", "", "30m", "pending", 1, true},
		{"fr-res", "1h", "2h", "", "met", 2, false},
		{"fr-nr", "1h", "", "30m", "met", 2, false},
		{"res-nr", "", "2h", "30m", "met", 2, false},
		{"fr-res-nr", "1h", "2h", "30m", "met", 2, false},
	}
	m, db := newTestManager(t)
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := insertPolicy(t, db, tc.name, tc.fr, tc.res, tc.nr)
			conv := insertConversation(t, db, fmt.Sprintf("combo-%d", i))
			applySLA(t, m, conv, policy)

			row := fetchApplied(t, db, conv)[0]
			if row.FRDeadline.Valid != (tc.fr != "") || row.ResDeadline.Valid != (tc.res != "") {
				t.Fatalf("deadline validity mismatch for %s: %+v", tc.name, row)
			}
			if tc.nr != "" {
				if _, err := m.CreateNextResponseSLAEvent(conv, row.ID, policy, 0); err != nil {
					t.Fatalf("CreateNextResponseSLAEvent: %v", err)
				}
			}

			if tc.fr != "" {
				db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)
			}
			if tc.res != "" {
				db.MustExec(`UPDATE conversations SET resolved_at = NOW() WHERE id = $1`, conv)
			}
			if err := m.evaluatePendingSLAs(context.Background()); err != nil {
				t.Fatalf("evaluatePendingSLAs: %v", err)
			}
			row = fetchApplied(t, db, conv)[0]
			if row.Status != tc.wantStatus {
				t.Fatalf("expected status %s after evaluation, got %+v", tc.wantStatus, row)
			}

			next := insertPolicy(t, db, tc.name+"-next", "3h", "4h", "")
			applySLA(t, m, conv, next)
			rows := fetchApplied(t, db, conv)
			if len(rows) != tc.wantRowsAfter {
				t.Fatalf("expected %d rows after re-apply, got %+v", tc.wantRowsAfter, rows)
			}
			var pending int
			for _, r := range rows {
				if r.Status == "pending" {
					pending++
				}
			}
			if pending != 1 {
				t.Fatalf("expected exactly one pending row after re-apply, got %+v", rows)
			}
			if tc.settledOldGone && rows[0].ID == row.ID {
				t.Fatalf("expected untouched old row deleted on re-apply, got %+v", rows)
			}
			orphanEvents := queryInt(t, db, `SELECT COUNT(*) FROM sla_events WHERE applied_sla_id = $1 AND status = 'pending'`, row.ID)
			if orphanEvents != 0 {
				t.Fatalf("expected no pending events left on superseded sla, got %d", orphanEvents)
			}
		})
	}
}

func TestSweepStatusLabels(t *testing.T) {
	cases := []struct {
		name              string
		fr, res           string
		frMet, frBreach   bool
		resMet, resBreach bool
		want              string
	}{
		{"both-met", "1h", "2h", true, false, true, false, "met"},
		{"both-breached", "1h", "2h", false, true, false, true, "breached"},
		{"fr-met-res-breached", "1h", "2h", true, false, false, true, "partially_met"},
		{"fr-breached-res-met", "1h", "2h", false, true, true, false, "partially_met"},
		{"fr-only-met", "1h", "", true, false, false, false, "met"},
		{"fr-only-breached", "1h", "", false, true, false, false, "breached"},
		{"res-only-met", "", "2h", false, false, true, false, "met"},
		{"res-only-breached", "", "2h", false, false, false, true, "breached"},
	}
	m, db := newTestManager(t)
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := insertPolicy(t, db, tc.name, tc.fr, tc.res, "")
			conv := insertConversation(t, db, fmt.Sprintf("label-%d", i))
			applySLA(t, m, conv, policy)
			db.MustExec(`UPDATE applied_slas SET
				first_response_met_at = CASE WHEN $2 THEN NOW() END,
				first_response_breached_at = CASE WHEN $3 THEN NOW() END,
				resolution_met_at = CASE WHEN $4 THEN NOW() END,
				resolution_breached_at = CASE WHEN $5 THEN NOW() END
				WHERE conversation_id = $1`, conv, tc.frMet, tc.frBreach, tc.resMet, tc.resBreach)

			if err := m.evaluatePendingSLAs(context.Background()); err != nil {
				t.Fatalf("evaluatePendingSLAs: %v", err)
			}
			row := fetchApplied(t, db, conv)[0]
			if row.Status != tc.want {
				t.Fatalf("expected %s, got %+v", tc.want, row)
			}
		})
	}
}

func TestSendNotificationSkipsResolvedConversation(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID
	db.MustExec(`UPDATE conversations SET status_id = (SELECT id FROM conversation_statuses WHERE category = 'resolved' LIMIT 1) WHERE id = $1`, conv)
	db.MustExec(`INSERT INTO scheduled_sla_notifications (applied_sla_id, metric, notification_type, recipients, send_at) VALUES ($1, 'first_response', 'breach', '{1}', NOW() - INTERVAL '1 min')`, appliedID)

	var pending []models.ScheduledSLANotification
	if err := m.q.GetScheduledSLANotifications.Select(&pending); err != nil {
		t.Fatalf("fetching scheduled notifications: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected one due notification, got %d", len(pending))
	}
	if err := m.SendNotification(pending[0]); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	if queryInt(t, db, `SELECT COUNT(*) FROM scheduled_sla_notifications WHERE id = $1 AND processed_at IS NOT NULL`, pending[0].ID) != 1 {
		t.Fatal("expected resolved-conversation notification marked processed without sending")
	}
}

func TestSweepRepairsConversationDeadline(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	db.MustExec(`UPDATE applied_slas SET first_response_met_at = NOW(), resolution_met_at = NOW() WHERE conversation_id = $1`, conv)
	db.MustExec(`UPDATE conversations SET next_sla_deadline_at = NOW() + INTERVAL '10 days' WHERE id = $1`, conv)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	if fetchApplied(t, db, conv)[0].Status != "met" {
		t.Fatal("expected sla closed")
	}
	if d := conversationDeadline(t, db, conv); d.Valid {
		t.Fatalf("expected sweep to clear the stale deadline (nothing left due), got %v", d)
	}
}

func TestApplySLADeadlinesHonorBusinessHours(t *testing.T) {
	m, db := newTestManagerBH(t, bmodels.BusinessHours{
		ID:       1,
		Holidays: types.JSONText(`[]`),
		Hours:    types.JSONText(`{"Tuesday":{"open":"09:00","close":"17:00"}}`),
	})
	policy := insertPolicy(t, db, "p1", "1h", "8h", "")
	conv := insertConversation(t, db, "c1")
	start := time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC)
	if _, err := m.ApplySLA(start, conv, 0, policy); err != nil {
		t.Fatalf("ApplySLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	wantFR := time.Date(2023, 10, 10, 11, 0, 0, 0, time.UTC)
	wantRes := time.Date(2023, 10, 17, 10, 0, 0, 0, time.UTC)
	if !row.FRDeadline.Time.UTC().Equal(wantFR) {
		t.Fatalf("expected first response deadline %v, got %v", wantFR, row.FRDeadline.Time.UTC())
	}
	if !row.ResDeadline.Time.UTC().Equal(wantRes) {
		t.Fatalf("expected resolution deadline to spill into next working day %v, got %v", wantRes, row.ResDeadline.Time.UTC())
	}
}

func TestConcurrentApplySLA(t *testing.T) {
	m, db := newTestManager(t)
	conv := insertConversation(t, db, "c1")
	policies := make([]int, 8)
	for i := range policies {
		policies[i] = insertPolicy(t, db, fmt.Sprintf("p%d", i), "1h", "2h", "")
	}

	var wg sync.WaitGroup
	var successes atomic.Int32
	for _, p := range policies {
		wg.Add(1)
		go func(policyID int) {
			defer wg.Done()
			if _, err := m.ApplySLA(time.Now(), conv, 0, policyID); err == nil {
				successes.Add(1)
			}
		}(p)
	}
	wg.Wait()

	if successes.Load() == 0 {
		t.Fatal("expected at least one concurrent ApplySLA to succeed")
	}
	pending := queryInt(t, db, `SELECT COUNT(*) FROM applied_slas WHERE conversation_id = $1 AND status = 'pending'`, conv)
	total := queryInt(t, db, `SELECT COUNT(*) FROM applied_slas WHERE conversation_id = $1`, conv)
	if pending != 1 {
		t.Fatalf("expected exactly one pending row after concurrent applies, got %d pending of %d total", pending, total)
	}
}

func TestWarningNotificationSchedule(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "", "")
	db.MustExec(`UPDATE sla_policies SET notifications = '[{"type":"warning","time_delay_type":"before","time_delay":"10m","recipients":["assigned_user"]}]' WHERE id = $1`, policy)
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	row := fetchApplied(t, db, conv)[0]
	var sendAt time.Time
	if err := db.QueryRow(`SELECT send_at FROM scheduled_sla_notifications WHERE applied_sla_id = $1 AND notification_type = 'warning'`, row.ID).Scan(&sendAt); err != nil {
		t.Fatalf("fetching warning notification: %v", err)
	}
	want := row.FRDeadline.Time.Add(-10 * time.Minute)
	if sendAt.Sub(want).Abs() > time.Millisecond {
		t.Fatalf("expected warning at deadline minus 10m (%v), got %v", want, sendAt)
	}

	short := insertPolicy(t, db, "p2", "1m", "", "")
	db.MustExec(`UPDATE sla_policies SET notifications = '[{"type":"warning","time_delay_type":"before","time_delay":"10m","recipients":["assigned_user"]}]' WHERE id = $1`, short)
	conv2 := insertConversation(t, db, "c2")
	applySLA(t, m, conv2, short)
	if n := queryInt(t, db, `SELECT COUNT(*) FROM scheduled_sla_notifications WHERE applied_sla_id = $1`, fetchApplied(t, db, conv2)[0].ID); n != 0 {
		t.Fatalf("expected past-dated warning skipped, got %d scheduled", n)
	}
}

func TestSendNotificationsStopsOnCancel(t *testing.T) {
	m, _ := newTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.SendNotifications(ctx); err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPolicyCRUD(t *testing.T) {
	m, _ := newTestManager(t)
	created, err := m.Create("p1", "desc", null.StringFrom("1h"), null.StringFrom("2h"), null.StringFrom("30m"), models.SlaNotifications{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := m.Update(created.ID, "p1-renamed", "desc", null.StringFrom("3h"), null.StringFrom("4h"), null.String{}, models.SlaNotifications{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := m.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "p1-renamed" || got.FirstResponseTime.String != "3h" {
		t.Fatalf("expected updated policy, got %+v", got)
	}
	all, err := m.GetAll()
	if err != nil || len(all) != 2 {
		t.Fatalf("expected created policy plus the seeded default from GetAll, got %v %v", all, err)
	}
	if err := m.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := m.Get(created.ID); err == nil {
		t.Fatal("expected Get to fail after delete")
	}
}

func TestScoredNextResponseHistorySurvivesReapply(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "nr-only", "", "", "30m")
	p2 := insertPolicy(t, db, "p2", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, p1)
	oldID := fetchApplied(t, db, conv)[0].ID

	if _, err := m.CreateNextResponseSLAEvent(conv, oldID, p1, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if _, err := m.SetLatestSLAEventMetAt(oldID, MetricNextResponse); err != nil {
		t.Fatalf("SetLatestSLAEventMetAt: %v", err)
	}
	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}

	applySLA(t, m, conv, p2)

	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 || rows[0].ID != oldID || rows[0].Status == "pending" {
		t.Fatalf("expected scored nr-only sla closed and kept, got %+v", rows)
	}
	if n := queryInt(t, db, `SELECT COUNT(*) FROM sla_events WHERE applied_sla_id = $1 AND status = 'met'`, oldID); n != 1 {
		t.Fatalf("expected met next-response event history kept, got %d", n)
	}
}

func TestMetButUntickedEventSurvivesReapply(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "", "", "30m")
	p2 := insertPolicy(t, db, "p2", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, p1)
	oldID := fetchApplied(t, db, conv)[0].ID

	if _, err := m.CreateNextResponseSLAEvent(conv, oldID, p1, 0); err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	if _, err := m.SetLatestSLAEventMetAt(oldID, MetricNextResponse); err != nil {
		t.Fatalf("SetLatestSLAEventMetAt: %v", err)
	}

	applySLA(t, m, conv, p2)

	if n := queryInt(t, db, `SELECT COUNT(*) FROM sla_events WHERE applied_sla_id = $1 AND met_at IS NOT NULL`, oldID); n != 1 {
		t.Fatalf("expected met-but-unticked event kept through re-apply, got %d", n)
	}
	if err := m.evaluatePendingSLAEvents(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAEvents: %v", err)
	}
	if status := queryStr(t, db, `SELECT status FROM sla_events WHERE applied_sla_id = $1`, oldID); status != "met" {
		t.Fatalf("expected surviving event flipped to met by next tick, got %s", status)
	}
}

func TestSweepKeepsLiveNextResponseCountdown(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	appliedID := fetchApplied(t, db, conv)[0].ID

	nrDeadline, err := m.CreateNextResponseSLAEvent(conv, appliedID, policy, 0)
	if err != nil {
		t.Fatalf("CreateNextResponseSLAEvent: %v", err)
	}
	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '2h', resolution_deadline_at = NOW() - INTERVAL '1h' WHERE id = $1`, appliedID)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	if row := fetchApplied(t, db, conv)[0]; row.Status != "breached" {
		t.Fatalf("expected row closed breached, got %+v", row)
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || d.Time.Sub(nrDeadline).Abs() > time.Millisecond {
		t.Fatalf("expected live next-response countdown kept as conversation deadline, got %v want %v", d, nrDeadline)
	}
}

func TestDeadlineRecomputeUsesLatestSLA(t *testing.T) {
	m, db := newTestManager(t)
	p1 := insertPolicy(t, db, "p1", "1h", "2h", "")
	p2 := insertPolicy(t, db, "p2", "10h", "20h", "")
	conv := insertConversation(t, db, "c1")

	applySLA(t, m, conv, p1)
	db.MustExec(`UPDATE applied_slas SET first_response_met_at = NOW() WHERE conversation_id = $1`, conv)
	applySLA(t, m, conv, p2)
	rows := fetchApplied(t, db, conv)
	if len(rows) != 2 {
		t.Fatalf("expected history plus current row, got %+v", rows)
	}

	if _, err := m.q.UpdateConversationNextSLADeadline.Exec(pq.Array([]int{conv})); err != nil {
		t.Fatalf("UpdateConversationNextSLADeadline: %v", err)
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || d.Time.Sub(rows[1].FRDeadline.Time).Abs() > time.Millisecond {
		t.Fatalf("expected deadline from the latest sla %v, got %v", rows[1].FRDeadline.Time, d)
	}
}

func TestDeadlineRecomputeSkipsRepliedFirstResponse(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	row := fetchApplied(t, db, conv)[0]

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)
	if _, err := m.q.UpdateConversationNextSLADeadline.Exec(pq.Array([]int{conv})); err != nil {
		t.Fatalf("UpdateConversationNextSLADeadline: %v", err)
	}

	d := conversationDeadline(t, db, conv)
	if !d.Valid || d.Time.Sub(row.ResDeadline.Time).Abs() > time.Millisecond {
		t.Fatalf("expected resolution deadline %v after a first reply, got %v", row.ResDeadline.Time, d)
	}
}

func TestDeadlineRecomputeKeepsUnrepliedFirstResponse(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	row := fetchApplied(t, db, conv)[0]

	if _, err := m.q.UpdateConversationNextSLADeadline.Exec(pq.Array([]int{conv})); err != nil {
		t.Fatalf("UpdateConversationNextSLADeadline: %v", err)
	}

	d := conversationDeadline(t, db, conv)
	if !d.Valid || d.Time.Sub(row.FRDeadline.Time).Abs() > time.Millisecond {
		t.Fatalf("expected first response deadline %v with no reply yet, got %v", row.FRDeadline.Time, d)
	}
}

func TestQuietTickChangesNothing(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)
	before := fetchApplied(t, db, conv)[0]
	beforeDeadline := conversationDeadline(t, db, conv)

	for i := 0; i < 3; i++ {
		if err := m.evaluatePendingSLAs(context.Background()); err != nil {
			t.Fatalf("evaluatePendingSLAs: %v", err)
		}
	}

	after := fetchApplied(t, db, conv)[0]
	if after != before {
		t.Fatalf("expected quiet ticks to change nothing, before %+v after %+v", before, after)
	}
	if d := conversationDeadline(t, db, conv); d != beforeDeadline {
		t.Fatalf("expected conversation deadline untouched, got %v want %v", d, beforeDeadline)
	}
}

func TestSkippedRowPickedUpWhenDeadlinePasses(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}
	if row := fetchApplied(t, db, conv)[0]; row.Status != "pending" || row.FRBreached.Valid {
		t.Fatalf("expected row untouched while nothing is due, got %+v", row)
	}

	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '1 min' WHERE conversation_id = $1`, conv)
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}
	if row := fetchApplied(t, db, conv)[0]; !row.FRBreached.Valid || row.Status != "pending" {
		t.Fatalf("expected previously skipped row breached once due and still pending on resolution, got %+v", row)
	}

	db.MustExec(`UPDATE conversations SET resolved_at = NOW() WHERE id = $1`, conv)
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}
	if row := fetchApplied(t, db, conv)[0]; row.Status != "partially_met" || !row.ResMetAt.Valid {
		t.Fatalf("expected row settled across ticks, got %+v", row)
	}
}

func TestSweepIsolatedPerConversation(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	settling := insertConversation(t, db, "c1")
	bystander := insertConversation(t, db, "c2")
	applySLA(t, m, settling, policy)
	applySLA(t, m, bystander, policy)
	bystanderBefore := fetchApplied(t, db, bystander)[0]
	bystanderDeadline := conversationDeadline(t, db, bystander)

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW(), resolved_at = NOW() WHERE id = $1`, settling)
	if err := m.evaluatePendingSLAs(context.Background()); err != nil {
		t.Fatalf("evaluatePendingSLAs: %v", err)
	}

	if row := fetchApplied(t, db, settling)[0]; row.Status != "met" {
		t.Fatalf("expected settling conversation closed, got %+v", row)
	}
	if row := fetchApplied(t, db, bystander)[0]; row != bystanderBefore {
		t.Fatalf("expected bystander conversation untouched, before %+v after %+v", bystanderBefore, row)
	}
	if d := conversationDeadline(t, db, bystander); d != bystanderDeadline {
		t.Fatalf("expected bystander deadline untouched, got %v want %v", d, bystanderDeadline)
	}
}

func TestEvaluateConversationStampsMetOnReply(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "pending" || !row.FRMetAt.Valid || row.FRBreached.Valid {
		t.Fatalf("expected first response met and sla still pending, got %+v", row)
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || !d.Time.Equal(row.ResDeadline.Time) {
		t.Fatalf("expected cached deadline to move to resolution deadline, got %+v", d)
	}
}

func TestEvaluateConversationStampsBreachOnLateReply(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '2h' WHERE conversation_id = $1`, conv)
	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if !row.FRBreached.Valid || row.FRMetAt.Valid {
		t.Fatalf("expected first response breached on late reply, got %+v", row)
	}
}

func TestEvaluateConversationStampsBreachOnSilence(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE applied_slas SET first_response_deadline_at = NOW() - INTERVAL '1h' WHERE conversation_id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if !row.FRBreached.Valid || row.FRMetAt.Valid {
		t.Fatalf("expected first response breached with no reply, got %+v", row)
	}
}

func TestEvaluateConversationStampsResolutionMet(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW(), resolved_at = NOW(),
		status_id = (SELECT id FROM conversation_statuses WHERE category = 'resolved' LIMIT 1) WHERE id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if !row.FRMetAt.Valid || !row.ResMetAt.Valid {
		t.Fatalf("expected both metrics met, got %+v", row)
	}
	if d := conversationDeadline(t, db, conv); d.Valid {
		t.Fatalf("expected NULL cached deadline on resolved conversation, got %+v", d)
	}
}

func TestEvaluateConversationIsIdempotent(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}
	first := fetchApplied(t, db, conv)[0]
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA second call: %v", err)
	}
	second := fetchApplied(t, db, conv)[0]
	if !first.FRMetAt.Time.Equal(second.FRMetAt.Time) {
		t.Fatalf("expected met_at unchanged on second call, got %v then %v", first.FRMetAt.Time, second.FRMetAt.Time)
	}
}

func TestEvaluateConversationNoChangeStillRecomputes(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "p1", "1h", "2h", "")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	db.MustExec(`UPDATE conversations SET next_sla_deadline_at = NOW() - INTERVAL '5h' WHERE id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.FRMetAt.Valid || row.FRBreached.Valid {
		t.Fatalf("expected no stamps with future deadlines, got %+v", row)
	}
	d := conversationDeadline(t, db, conv)
	if !d.Valid || !d.Time.Equal(row.FRDeadline.Time) {
		t.Fatalf("expected cached deadline repaired to first response deadline, got %+v", d)
	}
}

func TestEvaluateConversationNoPendingRowStillRecomputes(t *testing.T) {
	m, db := newTestManager(t)
	conv := insertConversation(t, db, "c1")

	db.MustExec(`UPDATE conversations SET next_sla_deadline_at = NOW() + INTERVAL '1h' WHERE id = $1`, conv)
	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	if d := conversationDeadline(t, db, conv); d.Valid {
		t.Fatalf("expected stale cached deadline cleared with no applied sla, got %+v", d)
	}
}

func TestEvaluateConversationNextResponseOnlyRecomputes(t *testing.T) {
	m, db := newTestManager(t)
	policy := insertPolicy(t, db, "nr-only", "", "", "30m")
	conv := insertConversation(t, db, "c1")
	applySLA(t, m, conv, policy)

	appliedID := fetchApplied(t, db, conv)[0].ID
	db.MustExec(`INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at)
		VALUES ($1, $2, 'next_response', NOW() + INTERVAL '30m')`, appliedID, policy)
	db.MustExec(`UPDATE conversations SET next_sla_deadline_at = NULL WHERE id = $1`, conv)

	if err := m.EvaluateConversationSLA(conv); err != nil {
		t.Fatalf("EvaluateConversationSLA: %v", err)
	}

	row := fetchApplied(t, db, conv)[0]
	if row.Status != "pending" || row.FRMetAt.Valid || row.FRBreached.Valid || row.ResMetAt.Valid || row.ResBreached.Valid {
		t.Fatalf("expected next-response-only row untouched, got %+v", row)
	}
	if d := conversationDeadline(t, db, conv); !d.Valid {
		t.Fatalf("expected cached deadline recomputed from pending event, got %+v", d)
	}
}

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	return newTestManagerBH(t, bmodels.BusinessHours{ID: 1, IsAlwaysOpen: true})
}

func newTestManagerBH(t *testing.T, bh bmodels.BusinessHours) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "sla")
	lo := logf.New(logf.Opts{})
	mgr, err := New(
		Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)},
		stubTeamStore{},
		stubAppSettingsStore{},
		stubBusinessHrsStore{bh: bh},
		nil,
		stubUserStore{},
		nil,
	)
	if err != nil {
		t.Fatalf("creating sla manager: %v", err)
	}
	return mgr, db
}

func queryInt(t *testing.T, db *sqlx.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("querying %q: %v", query, err)
	}
	return n
}

func queryStr(t *testing.T, db *sqlx.DB, query string, args ...any) string {
	t.Helper()
	var s string
	if err := db.QueryRow(query, args...).Scan(&s); err != nil {
		t.Fatalf("querying %q: %v", query, err)
	}
	return s
}

func insertPolicy(t *testing.T, db *sqlx.DB, name, fr, res, nr string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO sla_policies (name, description, first_response_time, resolution_time, next_response_time) VALUES ($1, '', $2, $3, $4) RETURNING id`,
		name, fr, res, nr).Scan(&id)
	if err != nil {
		t.Fatalf("inserting sla policy: %v", err)
	}
	return id
}

func insertConversation(t *testing.T, db *sqlx.DB, ref string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`
		WITH contact AS (
			INSERT INTO users (type, email, first_name) VALUES ('contact', $1 || '@example.com', 'C') RETURNING id
		), inbox AS (
			INSERT INTO inboxes (channel, config, name) VALUES ('email', '{}', 'inb-' || $1) RETURNING id
		)
		INSERT INTO conversations (contact_id, inbox_id, status_id, reference_number, subject)
		SELECT contact.id, inbox.id, (SELECT id FROM conversation_statuses WHERE category != 'resolved' LIMIT 1), $1, 'subject'
		FROM contact, inbox RETURNING id`, ref).Scan(&id)
	if err != nil {
		t.Fatalf("inserting conversation: %v", err)
	}
	return id
}

func fetchApplied(t *testing.T, db *sqlx.DB, conversationID int) []appliedRow {
	t.Helper()
	var rows []appliedRow
	err := db.Select(&rows, `SELECT id, status, first_response_deadline_at, resolution_deadline_at,
		first_response_met_at, first_response_breached_at, resolution_met_at, resolution_breached_at
		FROM applied_slas WHERE conversation_id = $1 ORDER BY id`, conversationID)
	if err != nil {
		t.Fatalf("fetching applied slas: %v", err)
	}
	return rows
}

func conversationDeadline(t *testing.T, db *sqlx.DB, conversationID int) sql.NullTime {
	t.Helper()
	var d sql.NullTime
	if err := db.QueryRow(`SELECT next_sla_deadline_at FROM conversations WHERE id = $1`, conversationID).Scan(&d); err != nil {
		t.Fatalf("fetching conversation deadline: %v", err)
	}
	return d
}

func applySLA(t *testing.T, m *Manager, conversationID, policyID int) {
	t.Helper()
	if _, err := m.ApplySLA(time.Now(), conversationID, 0, policyID); err != nil {
		t.Fatalf("ApplySLA: %v", err)
	}
}
