package status

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/zerodha/logf"
)

func TestStatusWriteContracts(t *testing.T) {
	db := testutil.NewDB(t, "status_contracts")
	lo := logf.New(logf.Opts{Writer: io.Discard})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
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
	first, err := m.Create("custom", "open")
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Create(strings.Repeat("é", 25), "resolved")
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Create(first.Name, "resolved")
	assertError(err, envelope.ConflictError)
	_, err = m.Update(second.ID, first.Name, "open")
	assertError(err, envelope.ConflictError)
	persisted, err := m.Get(second.ID)
	if err != nil || persisted.Name != second.Name || persisted.Category != second.Category {
		t.Fatalf("conflict changed status: %+v %v", persisted, err)
	}
	_, err = m.Get(999999)
	assertError(err, envelope.NotFoundError)
	_, err = m.Update(999999, "missing", "open")
	assertError(err, envelope.NotFoundError)
	assertError(m.Delete(999999), envelope.NotFoundError)
	_, err = m.Update(1, "renamed", "open")
	assertError(err, envelope.InputError)
	assertError(m.Delete(1), envelope.InputError)
	if err = m.Delete(first.ID); err != nil {
		t.Fatal(err)
	}
	_, err = m.Get(first.ID)
	assertError(err, envelope.NotFoundError)
}
