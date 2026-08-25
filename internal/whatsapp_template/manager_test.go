package whatsapp_template

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testdb"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const testInboxName = "wa-test"

var errAccount = &whatsapp.MetaAPIError{Message: "no account"}

type stubResolver struct{}

type failingResolver struct{}

func (stubResolver) WhatsAppAccount(inboxID int) (whatsapp.Account, error) {
	return whatsapp.Account{PhoneNumberID: "PN1", WABAID: "WABA1", AccessToken: "TOKEN"}, nil
}

func (failingResolver) WhatsAppAccount(inboxID int) (whatsapp.Account, error) {
	return whatsapp.Account{}, errAccount
}

func TestCreateAndFetch(t *testing.T) {
	m, _ := testManager(t, metaOK("111"))
	inboxID := seedInbox(t, m)

	created, err := m.Create(context.Background(), models.Template{
		InboxID:      inboxID,
		Name:         "order_update",
		Language:     "en_US",
		Category:     models.CategoryUtility,
		BodyContent:  "Hi {{1}}",
		SampleValues: json.RawMessage(`{"1":"Ravi"}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected an id")
	}
	// Meta accepted it, so the row carries its template id and waits on review.
	if created.MetaTemplateID.String != "111" || created.Status != models.StatusPending {
		t.Fatalf("unexpected created template: %+v", created)
	}
	// Empty JSON columns must default rather than land as NULL.
	if string(created.Buttons) != "[]" || string(created.SampleValues) == "" {
		t.Fatalf("unexpected json defaults: buttons=%s sample=%s", created.Buttons, created.SampleValues)
	}

	got, err := m.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Name != "order_update" {
		t.Fatalf("unexpected template: %+v", got)
	}

	byName, err := m.GetByName(inboxID, "order_update")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if byName.ID != created.ID {
		t.Fatalf("unexpected template: %+v", byName)
	}

	list, err := m.GetByInbox(inboxID)
	if err != nil {
		t.Fatalf("get by inbox: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one template, got %d", len(list))
	}
}

func TestCreateDuplicateNameAndLanguage(t *testing.T) {
	m, _ := testManager(t, metaOK("222"))
	inboxID := seedInbox(t, m)
	tmpl := models.Template{InboxID: inboxID, Name: "dupe", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi"}

	if _, err := m.Create(context.Background(), tmpl); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := m.Create(context.Background(), tmpl)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "exists") {
		t.Fatalf("expected a conflict, got %v", err)
	}
}

// The same name in another language is a separate template on Meta, so it must be allowed.
func TestCreateSameNameDifferentLanguage(t *testing.T) {
	m, _ := testManager(t, metaOK("333"))
	inboxID := seedInbox(t, m)
	base := models.Template{InboxID: inboxID, Name: "greeting", Category: models.CategoryUtility, BodyContent: "Hi"}

	base.Language = "en_US"
	if _, err := m.Create(context.Background(), base); err != nil {
		t.Fatalf("en create: %v", err)
	}
	base.Language = "mr"
	if _, err := m.Create(context.Background(), base); err != nil {
		t.Fatalf("mr create: %v", err)
	}
}

func TestCreateMarksRejectedWhenMetaRefuses(t *testing.T) {
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":{"message":"bad content","code":100,"error_user_msg":"Template violates policy"}}`))
	})
	inboxID := seedInbox(t, m)

	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "rejected_tmpl", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != models.StatusRejected || created.RejectionReason.String != "Template violates policy" {
		t.Fatalf("unexpected template: %+v", created)
	}

	stored, err := m.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.Status != models.StatusRejected {
		t.Fatalf("the rejection must be persisted, got %+v", stored)
	}
}

// A template Meta never accepted still needs sample values, so a missing one is a local rejection.
func TestCreateMarksRejectedWhenSubmissionCannotBeBuilt(t *testing.T) {
	m, _ := testManager(t, metaOK("444"))
	inboxID := seedInbox(t, m)

	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "no_samples", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi {{name}}",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != models.StatusRejected || !strings.Contains(created.RejectionReason.String, "could not build") {
		t.Fatalf("unexpected template: %+v", created)
	}
}

func TestCreateWithoutMetaClientStaysPending(t *testing.T) {
	m, _ := testManager(t, nil)
	m.client, m.resolver = nil, nil
	inboxID := seedInbox(t, m)

	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "offline_tmpl", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != models.StatusPending || created.MetaTemplateID.Valid {
		t.Fatalf("unexpected template: %+v", created)
	}
}

func TestCreateWhenAccountCannotBeResolved(t *testing.T) {
	m, _ := testManager(t, metaOK("555"))
	m.resolver = failingResolver{}
	inboxID := seedInbox(t, m)

	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "no_account", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != models.StatusRejected || !strings.Contains(created.RejectionReason.String, "resolve WhatsApp account") {
		t.Fatalf("unexpected template: %+v", created)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	m, _ := testManager(t, nil)
	if _, err := m.GetByID(9999999); err == nil {
		t.Fatal("expected a not-found error")
	}
}

func TestGetByNameNotFound(t *testing.T) {
	m, _ := testManager(t, nil)
	inboxID := seedInbox(t, m)
	if _, err := m.GetByName(inboxID, "missing"); err != ErrTemplateNotFound {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestGetApproved(t *testing.T) {
	m, _ := testManager(t, metaOK("666"))
	inboxID := seedInbox(t, m)
	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "approval_flow", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Pending is not sendable.
	if _, err := m.GetApproved(inboxID, "approval_flow", "en_US"); err == nil {
		t.Fatal("expected a not-approved error")
	}
	if _, err := m.GetApproved(inboxID, "nope", "en_US"); err != ErrTemplateNotFound {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}

	if err := m.HandleStatusUpdate(inboxID, created.MetaTemplateID.String, "approval_flow", "en_US", "APPROVED", "NONE"); err != nil {
		t.Fatalf("status update: %v", err)
	}
	approved, err := m.GetApproved(inboxID, "approval_flow", "en_US")
	if err != nil {
		t.Fatalf("get approved: %v", err)
	}
	if approved.Status != models.StatusApproved || approved.RejectionReason.Valid {
		t.Fatalf("unexpected template: %+v", approved)
	}
}

func TestHandleStatusUpdate(t *testing.T) {
	m, _ := testManager(t, metaOK("777"))
	inboxID := seedInbox(t, m)
	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "status_flow", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	tests := []struct {
		name       string
		metaID     string
		tmplName   string
		event      string
		reason     string
		wantStatus string
		wantReason string
	}{
		{"rejected by meta id", created.MetaTemplateID.String, "status_flow", "REJECTED", "INVALID_FORMAT", models.StatusRejected, "INVALID_FORMAT"},
		{"paused", created.MetaTemplateID.String, "status_flow", "PAUSED", "", models.StatusPaused, ""},
		{"disabled", created.MetaTemplateID.String, "status_flow", "DISABLED", "", models.StatusDisabled, ""},
		{"reinstated counts as approved", created.MetaTemplateID.String, "status_flow", "REINSTATED", "NONE", models.StatusApproved, ""},
		{"matched by name when the id is unknown", "does-not-exist", "status_flow", "REJECTED", "SCAM", models.StatusRejected, "SCAM"},
		{"matched by name when no id is sent", "", "status_flow", "APPROVED", "NONE", models.StatusApproved, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := m.HandleStatusUpdate(inboxID, tc.metaID, tc.tmplName, "en_US", tc.event, tc.reason); err != nil {
				t.Fatalf("status update: %v", err)
			}
			got, err := m.GetByID(created.ID)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Fatalf("expected status %s, got %s", tc.wantStatus, got.Status)
			}
			if got.RejectionReason.String != tc.wantReason {
				t.Fatalf("expected reason %q, got %q", tc.wantReason, got.RejectionReason.String)
			}
		})
	}
}

func TestHandleStatusUpdateIgnoresUnknownRowsAndEvents(t *testing.T) {
	m, _ := testManager(t, nil)
	inboxID := seedInbox(t, m)

	if err := m.HandleStatusUpdate(inboxID, "", "", "en_US", "APPROVED", ""); err == nil {
		t.Fatal("expected an error when the payload identifies no template")
	}
	// An event libredesk does not model must not touch the row or fail the delivery.
	if err := m.HandleStatusUpdate(inboxID, "1", "any", "en_US", "PENDING_DELETION", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := m.HandleStatusUpdate(inboxID, "unknown-id", "", "en_US", "APPROVED", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := m.HandleStatusUpdate(inboxID, "unknown-id", "unknown-name", "en_US", "APPROVED", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete(t *testing.T) {
	var deleted []string
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.URL.Query().Get("name"))
			w.Write([]byte(`{"success":true}`))
			return
		}
		metaOK("888")(w, r)
	})
	inboxID := seedInbox(t, m)
	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "deletable", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := m.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != "deletable" {
		t.Fatalf("expected the template to be deleted on Meta, got %v", deleted)
	}
	if _, err := m.GetByID(created.ID); err == nil {
		t.Fatal("expected the row to be gone")
	}
}

// The CSAT template is provisioned by libredesk, so deleting it would break resolved-conversation surveys.
func TestDeleteRejectsReservedTemplate(t *testing.T) {
	m, _ := testManager(t, metaOK("999"))
	inboxID := seedInbox(t, m)
	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: models.CSATTemplateName(inboxID), Language: "en_US", Category: models.CategoryUtility, BodyContent: "Rate us",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	err = m.Delete(context.Background(), created.ID)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("expected a reserved-template error, got %v", err)
	}
	if _, err := m.GetByID(created.ID); err != nil {
		t.Fatalf("the row must survive: %v", err)
	}
}

// Meta failing the delete must not leave the row behind in libredesk.
func TestDeleteContinuesWhenMetaFails(t *testing.T) {
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":{"message":"gone","code":100}}`))
			return
		}
		metaOK("1000")(w, r)
	})
	inboxID := seedInbox(t, m)
	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "stale", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := m.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := m.GetByID(created.ID); err == nil {
		t.Fatal("expected the row to be gone")
	}
}

func TestDeleteNotFound(t *testing.T) {
	m, _ := testManager(t, nil)
	if err := m.Delete(context.Background(), 9999999); err == nil {
		t.Fatal("expected a not-found error")
	}
}

func TestSyncFromMeta(t *testing.T) {
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"data": []map[string]any{
			{
				"id": "SYNC1", "name": "synced_one", "language": "en_US", "category": "MARKETING", "status": "APPROVED",
				"components": []map[string]any{
					{"type": "HEADER", "format": "IMAGE"},
					{"type": "BODY", "text": "Offer for {{1}}"},
					{"type": "FOOTER", "text": "Reply STOP to opt out"},
					{"type": "BUTTONS", "buttons": []map[string]any{{"type": "QUICK_REPLY", "text": "Tell me more"}}},
				},
			},
			{"id": "SYNC2", "name": "synced_two", "language": "mr", "category": "UTILITY", "status": "PENDING"},
		}})
	})
	inboxID := seedInbox(t, m)

	count, err := m.SyncFromMeta(context.Background(), inboxID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two templates, got %d", count)
	}
	got, err := m.GetByName(inboxID, "synced_one")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.HeaderType.String != "IMAGE" || got.BodyContent != "Offer for {{1}}" || got.FooterContent.String != "Reply STOP to opt out" {
		t.Fatalf("unexpected synced template: %+v", got)
	}

	// Syncing again must update rather than duplicate.
	if _, err := m.SyncFromMeta(context.Background(), inboxID); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	list, err := m.GetByInbox(inboxID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two rows after a repeat sync, got %d", len(list))
	}
}

// Meta is the source of truth, so a status change there overwrites the local one.
func TestSyncFromMetaOverwritesLocalStatus(t *testing.T) {
	status := "APPROVED"
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			metaOK("SYNC3")(w, r)
			return
		}
		writeJSON(w, map[string]any{"data": []map[string]any{
			{"id": "SYNC3", "name": "drifting", "language": "en_US", "category": "UTILITY", "status": status,
				"components": []map[string]any{{"type": "BODY", "text": "Hi"}}},
		}})
	})
	inboxID := seedInbox(t, m)
	created, err := m.Create(context.Background(), models.Template{
		InboxID: inboxID, Name: "drifting", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := m.SyncFromMeta(context.Background(), inboxID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got, _ := m.GetByID(created.ID); got.Status != models.StatusApproved {
		t.Fatalf("expected APPROVED after the sync, got %s", got.Status)
	}

	status = "PAUSED"
	if _, err := m.SyncFromMeta(context.Background(), inboxID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got, _ := m.GetByID(created.ID); got.Status != models.StatusPaused {
		t.Fatalf("expected PAUSED after the sync, got %s", got.Status)
	}
}

func TestSyncFromMetaFailures(t *testing.T) {
	t.Run("no client", func(t *testing.T) {
		m, _ := testManager(t, nil)
		m.client = nil
		if _, err := m.SyncFromMeta(context.Background(), 1); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("account cannot be resolved", func(t *testing.T) {
		m, _ := testManager(t, metaOK("x"))
		m.resolver = failingResolver{}
		if _, err := m.SyncFromMeta(context.Background(), 1); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("meta rejects the fetch", func(t *testing.T) {
		m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":{"message":"bad token","code":190}}`))
		})
		if _, err := m.SyncFromMeta(context.Background(), seedInbox(t, m)); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("a row that cannot be stored is skipped", func(t *testing.T) {
		m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"data": []map[string]any{
				{"id": "SYNC4", "name": strings.Repeat("x", 600), "language": "en_US", "category": "UTILITY", "status": "APPROVED"},
				{"id": "SYNC5", "name": "fine", "language": "en_US", "category": "UTILITY", "status": "APPROVED"},
			}})
		})
		count, err := m.SyncFromMeta(context.Background(), seedInbox(t, m))
		if err != nil {
			t.Fatalf("sync: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected the oversized row to be skipped, got count %d", count)
		}
	})
}

func TestEnsureReservedCreatesThenEdits(t *testing.T) {
	var (
		submitted []map[string]any
		edited    []string
	)
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/message_templates") && r.Method == http.MethodPost {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			submitted = append(submitted, body)
			writeJSON(w, map[string]any{"id": "CSAT1", "status": "PENDING"})
			return
		}
		if r.Method == http.MethodPost {
			edited = append(edited, r.URL.Path)
			writeJSON(w, map[string]bool{"success": true})
			return
		}
		writeJSON(w, map[string]any{"data": []any{}})
	})
	inboxID := seedInbox(t, m)
	name := models.CSATTemplateName(inboxID)

	desired := models.Template{
		InboxID:     inboxID,
		Name:        name,
		Language:    "en_US",
		Category:    models.CategoryUtility,
		BodyContent: "Rate us please",
		Buttons:     csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
	}
	if err := m.EnsureReserved(context.Background(), desired); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if len(submitted) != 1 {
		t.Fatalf("expected one submission, got %d", len(submitted))
	}

	// Identical content must not go back to Meta.
	if err := m.EnsureReserved(context.Background(), desired); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if len(submitted) != 1 || len(edited) != 0 {
		t.Fatalf("unchanged content must not be resubmitted: submits=%d edits=%d", len(submitted), len(edited))
	}

	// While the template is pending review Meta refuses edits, so libredesk must hold them back.
	changed := desired
	changed.BodyContent = "Rate us, please"
	if err := m.EnsureReserved(context.Background(), changed); err != nil {
		t.Fatalf("pending ensure: %v", err)
	}
	if len(edited) != 0 {
		t.Fatalf("expected no edit while pending, got %v", edited)
	}

	if err := m.HandleStatusUpdate(inboxID, "CSAT1", name, "en_US", "APPROVED", "NONE"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := m.EnsureReserved(context.Background(), changed); err != nil {
		t.Fatalf("approved ensure: %v", err)
	}
	if len(edited) != 1 || !strings.HasSuffix(edited[0], "/CSAT1") {
		t.Fatalf("expected an edit against the Meta template id, got %v", edited)
	}
	stored, err := m.GetByName(inboxID, name)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.BodyContent != "Rate us, please" || stored.Status != models.StatusPending {
		t.Fatalf("an edit must store the new copy and go back to pending: %+v", stored)
	}
}

// A language change is a different template on Meta, so it has to be created rather than edited.
func TestEnsureReservedCreatesPerLanguage(t *testing.T) {
	submits := 0
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		submits++
		writeJSON(w, map[string]any{"id": "CSAT" + string(rune('A'+submits)), "status": "PENDING"})
	})
	inboxID := seedInbox(t, m)
	desired := models.Template{
		InboxID: inboxID, Name: models.CSATTemplateName(inboxID), Category: models.CategoryUtility,
		BodyContent: "Rate us", Buttons: csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
	}

	desired.Language = "en_US"
	if err := m.EnsureReserved(context.Background(), desired); err != nil {
		t.Fatalf("en ensure: %v", err)
	}
	desired.Language = "mr"
	if err := m.EnsureReserved(context.Background(), desired); err != nil {
		t.Fatalf("mr ensure: %v", err)
	}
	if submits != 2 {
		t.Fatalf("expected a submission per language, got %d", submits)
	}
}

// A template that was rejected before it reached Meta has no id to edit, so it is submitted afresh.
func TestEnsureReservedResubmitsWhenMetaIDIsMissing(t *testing.T) {
	submits := 0
	m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		submits++
		if submits == 1 {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":{"message":"nope","code":100}}`))
			return
		}
		writeJSON(w, map[string]any{"id": "CSAT9", "status": "PENDING"})
	})
	inboxID := seedInbox(t, m)
	desired := models.Template{
		InboxID: inboxID, Name: models.CSATTemplateName(inboxID), Language: "en_US", Category: models.CategoryUtility,
		BodyContent: "Rate us", Buttons: csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
	}

	if err := m.EnsureReserved(context.Background(), desired); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	stored, _ := m.GetByName(inboxID, desired.Name)
	if stored.Status != models.StatusRejected || stored.MetaTemplateID.Valid {
		t.Fatalf("expected a rejected row with no meta id: %+v", stored)
	}

	desired.BodyContent = "Rate us now"
	if err := m.EnsureReserved(context.Background(), desired); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	stored, _ = m.GetByName(inboxID, desired.Name)
	if stored.MetaTemplateID.String != "CSAT9" || stored.Status != models.StatusPending {
		t.Fatalf("expected a fresh submission: %+v", stored)
	}
}

func TestEnsureReservedEditFailures(t *testing.T) {
	t.Run("meta rejects the edit", func(t *testing.T) {
		m, _ := testManager(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/message_templates") {
				writeJSON(w, map[string]any{"id": "CSATE", "status": "PENDING"})
				return
			}
			w.WriteHeader(400)
			w.Write([]byte(`{"error":{"message":"cannot edit","code":100,"error_user_msg":"Edit refused"}}`))
		})
		inboxID := seedInbox(t, m)
		name := models.CSATTemplateName(inboxID)
		desired := models.Template{
			InboxID: inboxID, Name: name, Language: "en_US", Category: models.CategoryUtility,
			BodyContent: "Rate us", Buttons: csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
		}
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := m.HandleStatusUpdate(inboxID, "CSATE", name, "en_US", "APPROVED", ""); err != nil {
			t.Fatalf("approve: %v", err)
		}
		desired.BodyContent = "Rate us again"
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		stored, _ := m.GetByName(inboxID, name)
		if stored.Status != models.StatusRejected || stored.RejectionReason.String != "Edit refused" {
			t.Fatalf("expected the refusal to be recorded: %+v", stored)
		}
	})

	t.Run("account cannot be resolved", func(t *testing.T) {
		m, _ := testManager(t, metaOK("CSATF"))
		inboxID := seedInbox(t, m)
		name := models.CSATTemplateName(inboxID)
		desired := models.Template{
			InboxID: inboxID, Name: name, Language: "en_US", Category: models.CategoryUtility,
			BodyContent: "Rate us", Buttons: csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
		}
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := m.HandleStatusUpdate(inboxID, "CSATF", name, "en_US", "APPROVED", ""); err != nil {
			t.Fatalf("approve: %v", err)
		}
		m.resolver = failingResolver{}
		desired.BodyContent = "Rate us again"
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		stored, _ := m.GetByName(inboxID, name)
		if stored.Status != models.StatusRejected {
			t.Fatalf("expected a rejected row: %+v", stored)
		}
	})

	t.Run("submission cannot be built", func(t *testing.T) {
		m, _ := testManager(t, metaOK("CSATG"))
		inboxID := seedInbox(t, m)
		name := models.CSATTemplateName(inboxID)
		desired := models.Template{
			InboxID: inboxID, Name: name, Language: "en_US", Category: models.CategoryUtility,
			BodyContent: "Rate us", Buttons: csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
		}
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := m.HandleStatusUpdate(inboxID, "CSATG", name, "en_US", "APPROVED", ""); err != nil {
			t.Fatalf("approve: %v", err)
		}
		// A body placeholder with no sample value cannot be submitted.
		desired.BodyContent = "Rate us {{name}}"
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		stored, _ := m.GetByName(inboxID, name)
		if stored.Status != models.StatusRejected || !strings.Contains(stored.RejectionReason.String, "could not build") {
			t.Fatalf("expected a build failure to be recorded: %+v", stored)
		}
	})

	t.Run("without a meta client the row is still updated", func(t *testing.T) {
		m, _ := testManager(t, metaOK("CSATH"))
		inboxID := seedInbox(t, m)
		name := models.CSATTemplateName(inboxID)
		desired := models.Template{
			InboxID: inboxID, Name: name, Language: "en_US", Category: models.CategoryUtility,
			BodyContent: "Rate us", Buttons: csatButtons("Rate us", "https://desk.test/csat/{{1}}"),
		}
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := m.HandleStatusUpdate(inboxID, "CSATH", name, "en_US", "APPROVED", ""); err != nil {
			t.Fatalf("approve: %v", err)
		}
		m.client = nil
		desired.BodyContent = "Rate us offline"
		if err := m.EnsureReserved(context.Background(), desired); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		stored, _ := m.GetByName(inboxID, name)
		if stored.BodyContent != "Rate us offline" {
			t.Fatalf("expected the local copy to be updated: %+v", stored)
		}
	})
}

// Every method has to surface a database failure instead of pretending the write happened.
func TestDatabaseFailuresSurface(t *testing.T) {
	m := managerOnClosedDB(t)

	if _, err := m.GetByInbox(1); err == nil {
		t.Error("GetByInbox must fail")
	}
	if _, err := m.GetByID(1); err == nil {
		t.Error("GetByID must fail")
	}
	if _, err := m.GetByName(1, "x"); err == nil {
		t.Error("GetByName must fail")
	}
	if _, err := m.GetApproved(1, "x", "en_US"); err == nil {
		t.Error("GetApproved must fail")
	}
	if _, err := m.Create(context.Background(), models.Template{InboxID: 1, Name: "x", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Hi"}); err == nil {
		t.Error("Create must fail")
	}
	if err := m.EnsureReserved(context.Background(), models.Template{InboxID: 1, Name: "x", Language: "en_US", BodyContent: "Hi"}); err == nil {
		t.Error("EnsureReserved must fail")
	}
	if err := m.Delete(context.Background(), 1); err == nil {
		t.Error("Delete must fail")
	}
	if err := m.HandleStatusUpdate(1, "id", "name", "en_US", "APPROVED", ""); err == nil {
		t.Error("HandleStatusUpdate by meta id must fail")
	}
	if err := m.HandleStatusUpdate(1, "", "name", "en_US", "APPROVED", ""); err == nil {
		t.Error("HandleStatusUpdate by name must fail")
	}
	if _, err := m.SyncFromMeta(context.Background(), 1); err != nil {
		t.Errorf("a sync whose rows cannot be stored still reports what it fetched: %v", err)
	}
}

func TestSubmitErrReason(t *testing.T) {
	if got := submitErrReason(nil); got != "" {
		t.Fatalf("expected an empty reason, got %q", got)
	}
	if got := submitErrReason(&whatsapp.MetaAPIError{Message: "raw", UserMsg: "friendly"}); got != "friendly" {
		t.Fatalf("expected the user message, got %q", got)
	}
	if got := submitErrReason(&whatsapp.MetaAPIError{Message: "raw"}); got != "raw" {
		t.Fatalf("expected the raw message, got %q", got)
	}
	if got := submitErrReason(context.Canceled); got != context.Canceled.Error() {
		t.Fatalf("unexpected reason %q", got)
	}
}

func TestButtonsSurfaceEqualHandlesMalformedJSON(t *testing.T) {
	if !buttonsSurfaceEqual(json.RawMessage(`not json`), json.RawMessage(`also not json`)) {
		t.Fatal("two unreadable button sets must compare equal so a reserved template is not resubmitted forever")
	}
	if buttonsSurfaceEqual(json.RawMessage(`not json`), csatButtons("Rate us", "https://x.test/{{1}}")) {
		t.Fatal("an unreadable set must not match a real one")
	}
}

func testManager(t *testing.T, handler http.HandlerFunc) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testdb.New(t, testInboxName)

	var client *whatsapp.Client
	if handler != nil {
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)
		client = whatsapp.New(testLogger())
		client.SetBaseURL(srv.URL)
	} else {
		client = whatsapp.New(testLogger())
	}

	m, err := New(Opts{Lo: testLogger(), DB: db, I18n: testI18n(t), Client: client, Resolver: stubResolver{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m, db
}

func seedInbox(t *testing.T, m *Manager) int {
	t.Helper()
	var id int
	db := testdb.New(t, testInboxName)
	if err := db.QueryRow(`INSERT INTO inboxes (channel, config, "name", enabled) VALUES ('whatsapp', '{}'::jsonb, $1, true) RETURNING id`,
		"wa-"+t.Name()).Scan(&id); err != nil {
		t.Fatalf("seeding an inbox: %v", err)
	}
	return id
}

func csatButtons(text, url string) json.RawMessage {
	raw, _ := json.Marshal([]map[string]any{{"type": "URL", "text": text, "url": url, "example": []string{strings.ReplaceAll(url, "{{1}}", "example")}}})
	return raw
}

func metaOK(templateID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"id": templateID, "status": "PENDING", "category": "UTILITY"})
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func testLogger() *logf.Logger {
	l := logf.New(logf.Opts{Level: logf.FatalLevel})
	return &l
}

func testI18n(t *testing.T) *i18n.I18n {
	t.Helper()
	i, err := i18n.New([]byte(`{"_.code":"en","_.name":"English","globals.messages.somethingWentWrong":"Something went wrong","globals.messages.notFound":"Not found","globals.messages.errorAlreadyExists":"Already exists"}`))
	if err != nil {
		t.Fatalf("i18n: %v", err)
	}
	return i
}

// managerOnClosedDB builds a manager whose statements are prepared and then invalidated.
func managerOnClosedDB(t *testing.T) *Manager {
	t.Helper()
	testdb.New(t, testInboxName)
	db, err := sqlx.Connect("postgres", strings.Replace(os.Getenv("LIBREDESK_TEST_DB_DSN"), "/libredesk?", "/libredesk_test_"+testInboxName+"?", 1))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	srv := httptest.NewServer(metaOK("CLOSED"))
	t.Cleanup(srv.Close)
	client := whatsapp.New(testLogger())
	client.SetBaseURL(srv.URL)

	m, err := New(Opts{Lo: testLogger(), DB: db, I18n: testI18n(t), Client: client, Resolver: stubResolver{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return m
}
