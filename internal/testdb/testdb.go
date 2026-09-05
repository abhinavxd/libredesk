// Package testdb loads schema.sql into a test database, or skips the test when LIBREDESK_TEST_DB_DSN is unset (see `make test-db`).
package testdb

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const dsnEnvVar = "LIBREDESK_TEST_DB_DSN"

var (
	mu      sync.Mutex
	loaded  = map[string]*sqlx.DB{}
	skipMsg = fmt.Sprintf("set %s to run database tests (see `make test-db`)", dsnEnvVar)
)

// New loads schema.sql into a database named after suffix. Each package needs its own, since schema.sql drops every table.
func New(t testing.TB, suffix string) *sqlx.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(dsnEnvVar))
	if dsn == "" {
		t.Skip(skipMsg)
	}

	mu.Lock()
	defer mu.Unlock()
	if db, ok := loaded[suffix]; ok {
		return db
	}

	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("%s: %v", skipMsg, err)
	}
	defer admin.Close()

	name := "libredesk_test_" + suffix
	if _, err := admin.Exec(`CREATE DATABASE ` + pq(name)); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("creating test database %s: %v", name, err)
	}

	db, err := sqlx.Connect("postgres", swapDatabase(t, dsn, name))
	if err != nil {
		t.Fatalf("connecting to test database %s: %v", name, err)
	}
	if _, err := db.Exec(schema(t)); err != nil {
		t.Fatalf("loading schema.sql: %v", err)
	}

	loaded[suffix] = db
	return db
}

func swapDatabase(t testing.TB, dsn, name string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parsing %s: %v", dsnEnvVar, err)
	}
	u.Path = "/" + name
	return u.String()
}

func schema(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "schema.sql")
		if _, err := os.Stat(candidate); err == nil {
			raw, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatalf("reading %s: %v", candidate, err)
			}
			return string(raw)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find schema.sql above the test's directory")
		}
		dir = parent
	}
}

func pq(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
