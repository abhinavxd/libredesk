package template

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/abhinavxd/libredesk/internal/whatsapp/template/models"
	"github.com/volatiletech/null/v9"
)

func TestUpdateTemplate(t *testing.T) {
	for _, status := range []string{models.StatusApproved, models.StatusRejected, models.StatusPaused} {
		t.Run(status, func(t *testing.T) {
			var edit whatsapp.TemplateEdit
			m, db := testManager(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/message_templates") {
					metaOK("EDIT1")(w, r)
					return
				}
				if err := json.NewDecoder(r.Body).Decode(&edit); err != nil {
					t.Error(err)
				}
				writeJSON(w, map[string]bool{"success": true})
			})
			original, err := m.Create(t.Context(), models.Template{
				InboxID: seedInbox(t, db), Name: "editable", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Original",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := m.HandleStatusUpdate(original.InboxID, "EDIT1", original.Name, original.Language, status, ""); err != nil {
				t.Fatal(err)
			}
			desired := original
			desired.BodyContent = "Updated {{1}}"
			desired.HeaderType = null.StringFrom("TEXT")
			desired.HeaderContent = null.StringFrom("Order update")
			desired.FooterContent = null.StringFrom("Support")
			desired.SampleValues = json.RawMessage(`{"1":"Ruchika"}`)
			desired.Buttons = json.RawMessage(`[{"type":"QUICK_REPLY","text":"Thanks"}]`)
			updated, err := m.Update(t.Context(), original.ID, desired)
			if err != nil {
				t.Fatal(err)
			}
			if updated.ID != original.ID || updated.MetaTemplateID.String != "EDIT1" || updated.Status != models.StatusPending || updated.BodyContent != desired.BodyContent {
				t.Fatalf("unexpected update: %+v", updated)
			}
			if len(edit.Components) != 4 || edit.Category != "" || !updated.SupportsTextContent() {
				t.Fatalf("unexpected components or category: %+v", edit)
			}
			stored, err := m.GetByID(original.ID)
			if err != nil || stored.BodyContent != desired.BodyContent || string(stored.Buttons) != string(updated.Buttons) || stored.RejectionReason.Valid {
				t.Fatalf("update not persisted: %+v, %v", stored, err)
			}
		})
	}
}

func TestUpdateTemplateRejectsInvalidChanges(t *testing.T) {
	tests := []struct {
		name   string
		change func(*models.Template)
	}{
		{"inbox", func(v *models.Template) { v.InboxID++ }},
		{"name", func(v *models.Template) { v.Name = "renamed" }},
		{"language", func(v *models.Template) { v.Language = "mr" }},
		{"approved category", func(v *models.Template) { v.Category = models.CategoryMarketing }},
		{"media header", func(v *models.Template) { v.HeaderType = null.StringFrom("IMAGE") }},
		{"missing sample", func(v *models.Template) { v.BodyContent = "Hi {{name}}" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			m, db := testManager(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				metaOK("GUARD1")(w, r)
			})
			original, err := m.Create(t.Context(), models.Template{
				InboxID: seedInbox(t, db), Name: "unchanged", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Original",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := m.HandleStatusUpdate(original.InboxID, "GUARD1", original.Name, original.Language, "APPROVED", ""); err != nil {
				t.Fatal(err)
			}
			desired := original
			tc.change(&desired)
			if _, err := m.Update(t.Context(), original.ID, desired); err == nil {
				t.Fatal("expected invalid edit to fail")
			}
			stored, err := m.GetByID(original.ID)
			if err != nil || stored.BodyContent != "Original" || stored.Status != models.StatusApproved || calls != 1 {
				t.Fatalf("invalid edit changed saved template: %+v, calls=%d, error=%v", stored, calls, err)
			}
		})
	}
}

func TestUpdateTemplatePreservesContentOnMetaFailure(t *testing.T) {
	m, db := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/message_templates") {
			metaOK("REFUSED1")(w, r)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"error": map[string]any{"code": 100, "message": "Edit refused"}})
	})
	original, err := m.Create(t.Context(), models.Template{
		InboxID: seedInbox(t, db), Name: "refused", Language: "en_US", Category: models.CategoryUtility, BodyContent: "Original",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.HandleStatusUpdate(original.InboxID, "REFUSED1", original.Name, original.Language, "APPROVED", ""); err != nil {
		t.Fatal(err)
	}
	desired := original
	desired.BodyContent = "Not accepted"
	if _, err := m.Update(t.Context(), original.ID, desired); err == nil {
		t.Fatal("expected Meta error")
	}
	stored, err := m.GetByID(original.ID)
	if err != nil || stored.BodyContent != "Original" || stored.Status != models.StatusApproved {
		t.Fatalf("failed edit changed template: %+v, %v", stored, err)
	}
}

func TestSyncPreservesUnsupportedComponentTypes(t *testing.T) {
	m, db := testManager(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"data": []whatsapp.MetaTemplate{{
			ID: "CAROUSEL1", Name: "arbitrary_name", Language: "en_US", Category: "MARKETING", Status: "APPROVED",
			Components: []whatsapp.TemplateComponent{{Type: "BODY", Text: "Offers"}, {Type: "CAROUSEL"}},
		}}})
	})
	inboxID := seedInbox(t, db)
	if count, err := m.SyncFromMeta(t.Context(), inboxID); err != nil || count != 1 {
		t.Fatalf("sync: count=%d, error=%v", count, err)
	}
	stored, err := m.GetByName(inboxID, "arbitrary_name")
	if err != nil || len(stored.ComponentTypes) != 2 || stored.ComponentTypes[1] != "CAROUSEL" || stored.SupportsTextContent() {
		t.Fatalf("carousel requirements were lost: %+v, %v", stored, err)
	}
	if _, err := m.Update(t.Context(), stored.ID, stored); err == nil {
		t.Fatal("carousel must not be editable as text")
	}
}
