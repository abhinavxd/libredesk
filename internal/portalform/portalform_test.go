package portalform

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/portalform/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestRenderHeaderBlock(t *testing.T) {
	if got := RenderHeaderBlock(nil); got != "" {
		t.Fatalf("empty lines: got %q", got)
	}
	got := RenderHeaderBlock([][2]string{{"Client ID", "BQA921"}, {"Category", "Account modification"}})
	want := "Client ID: BQA921\nCategory: Account modification\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidate(t *testing.T) {
	m := &Manager{i18n: testutil.NewI18n(t)}

	field := func(mut func(*models.Field)) models.Field {
		f := models.Field{Key: "client_id", Label: "Client ID", Type: models.FieldTypeText, Target: models.TargetMessage}
		mut(&f)
		return f
	}

	cases := []struct {
		name    string
		form    models.Form
		wantErr bool
	}{
		{"no name", models.Form{}, true},
		{"message field", models.Form{Name: "Form", Fields: []models.Field{field(func(*models.Field) {})}}, false},
		{"bad key", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) { f.Key = "Client ID" })}}, true},
		{"duplicate key", models.Form{Name: "Form", Fields: []models.Field{field(func(*models.Field) {}), field(func(*models.Field) {})}}, true},
		{"no label", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) { f.Label = " " })}}, true},
		{"unknown type", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) { f.Type = "range" })}}, true},
		{"select without options", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) { f.Type = models.FieldTypeSelect })}}, true},
		{"select with options", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) {
			f.Type = models.FieldTypeSelect
			f.Options = []string{"a"}
		})}}, false},
		{"attribute without key", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) { f.Target = models.TargetAttribute })}}, true},
		{"unknown target", models.Form{Name: "Form", Fields: []models.Field{field(func(f *models.Field) { f.Target = "email" })}}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := m.validate(c.form)
			if (err != nil) != c.wantErr {
				t.Fatalf("wantErr %v, got %v", c.wantErr, err)
			}
		})
	}
}

func TestValidateClearsAttributeKeyOnMessageField(t *testing.T) {
	m := &Manager{i18n: testutil.NewI18n(t)}
	form := models.Form{Name: "Form", Fields: []models.Field{
		{Key: "client_id", Label: "Client ID", Type: models.FieldTypeText, Target: models.TargetMessage, AttributeKey: "stale"},
	}}
	if _, err := m.validate(form); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if form.Fields[0].AttributeKey != "" {
		t.Fatalf("attribute key not cleared: %q", form.Fields[0].AttributeKey)
	}
}

func TestValidateFieldCap(t *testing.T) {
	m := &Manager{i18n: testutil.NewI18n(t)}
	fields := make([]models.Field, maxFields+1)
	for i := range fields {
		fields[i] = models.Field{Key: "f" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Label: "L", Type: models.FieldTypeText, Target: models.TargetMessage}
	}
	if _, err := m.validate(models.Form{Name: "Form", Fields: fields}); err == nil {
		t.Fatal("want error over the field cap")
	}
}
