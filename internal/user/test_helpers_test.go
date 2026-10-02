package user

import (
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
	"testing"
)

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "mail_user")
	lo := logf.New(logf.Opts{})
	mgr, err := New(testutil.NewI18n(t), Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatalf("creating user manager: %v", err)
	}
	return mgr, db
}
