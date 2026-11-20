package main

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func healthStatus(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	ctx := &fasthttp.RequestCtx{}
	req := &fastglue.Request{RequestCtx: ctx, Context: &App{lo: &lo, db: db}}
	if err := handleHealthCheck(req); err != nil {
		t.Fatalf("health check: %v", err)
	}
	return ctx.Response.StatusCode()
}

// A database that does not answer must fail the health check.
func TestHealthCheckFailsWhenDatabaseIsDown(t *testing.T) {
	// sqlx.Open does not connect, so this only fails once the handler pings.
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 user=libredesk dbname=libredesk sslmode=disable connect_timeout=1")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if got := healthStatus(t, db); got != fasthttp.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusServiceUnavailable)
	}
}

func TestHealthCheckPassesWhenDatabaseAnswers(t *testing.T) {
	db := testutil.NewDB(t, "cmd_health")

	if got := healthStatus(t, db); got != fasthttp.StatusOK {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusOK)
	}
}
