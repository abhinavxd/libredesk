// Package testutil provides a Postgres test harness for integration tests: start one with `make test-db` or set LIBREDESK_TEST_DB_DSN, else tests skip.
package testutil

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"

	_ "github.com/lib/pq"
)

const (
	dbDSNEnv     = "LIBREDESK_TEST_DB_DSN"
	dbDefaultDSN = "postgres://libredesk:libredesk@127.0.0.1:5433/libredesk?sslmode=disable&connect_timeout=3"
)

// NewDB provisions a fresh database named libredesk_test_<name> with schema.sql applied, skipping the test if Postgres is unreachable.
func NewDB(t *testing.T, name string) *sqlx.DB {
	t.Helper()

	dsn := os.Getenv(dbDSNEnv)
	if dsn == "" {
		dsn = dbDefaultDSN
	}
	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("test database unreachable, set %s or start the dev postgres: %v", dbDSNEnv, err)
	}
	defer admin.Close()

	dbName := "libredesk_test_" + name
	admin.MustExec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, dbName)
	admin.MustExec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName))
	admin.MustExec(fmt.Sprintf(`CREATE DATABASE %s`, dbName))

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parsing test DSN: %v", err)
	}
	u.Path = "/" + dbName
	db, err := sqlx.Connect("postgres", u.String())
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema, err := os.ReadFile(filepath.Join(repoRoot(t), "schema.sql"))
	if err != nil {
		t.Fatalf("reading schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("applying schema.sql: %v", err)
	}
	return db
}

// NewI18n loads the en-US translations.
func NewI18n(t *testing.T) *i18n.I18n {
	t.Helper()
	mgr, err := i18n.NewFromFile(filepath.Join(repoRoot(t), "i18n", "en-US.json"))
	if err != nil {
		t.Fatalf("loading i18n: %v", err)
	}
	return mgr
}

// repoRoot walks up from the working directory to the go.mod directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found in any parent directory")
		}
		dir = parent
	}
}
