package template

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/template/models"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestTemplateWriteContracts(t *testing.T) {
	db := testutil.NewDB(t, "template_contracts")
	lo := logf.New(logf.Opts{Writer: io.Discard})
	m, err := New(&lo, db, nil, nil, nil, testutil.NewI18n(t))
	if err != nil {
		t.Fatal(err)
	}
	assertError := func(err error, typ string) {
		t.Helper()
		var e envelope.Error
		if !errors.As(err, &e) || e.ErrorType != typ {
			t.Fatalf("got %v, want %s", err, typ)
		}
	}
	good := models.Template{Name: strings.Repeat("é", 140), Type: TypeEmailOutgoing, Subject: null.StringFrom(strings.Repeat("é", 1000)), Body: "body"}
	row, err := m.Create(good)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []models.Template{
		{Name: strings.Repeat("x", 141), Type: TypeEmailOutgoing},
		{Name: "ok", Type: TypeEmailOutgoing, Subject: null.StringFrom(strings.Repeat("x", 1001))},
		{Name: "ok", Type: "email_notification", IsDefault: true},
		{Name: " ", Type: TypeEmailOutgoing},
	} {
		_, err = m.Create(invalid)
		assertError(err, envelope.InputError)
		_, err = m.Update(row.ID, invalid)
		assertError(err, envelope.InputError)
	}
	persisted, err := m.Get(row.ID)
	if err != nil || persisted.Name != good.Name || persisted.Subject != good.Subject {
		t.Fatalf("invalid write changed template: %+v %v", persisted, err)
	}
	_, err = m.Update(999999, good)
	assertError(err, envelope.NotFoundError)
	db.MustExec("UPDATE templates SET is_default = false")
	good.IsDefault = true
	first, err := m.Create(good)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Create(good)
	assertError(err, envelope.ConflictError)
	_, err = m.Update(row.ID, good)
	assertError(err, envelope.ConflictError)
	var defaultID int
	if err = db.Get(&defaultID, "SELECT id FROM templates WHERE is_default"); err != nil || defaultID != first.ID {
		t.Fatalf("default changed: %d %v", defaultID, err)
	}
}
