package user

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordMismatchDoesNotLogApplicationError(t *testing.T) {
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs})
	manager := &Manager{lo: &lo}
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.verifyPassword([]byte("wrong"), string(hash)); err == nil {
		t.Fatal("incorrect password accepted")
	}
	if logs.Len() != 0 {
		t.Fatalf("ordinary password mismatch logged: %s", logs.String())
	}
	if err := manager.verifyPassword([]byte("correct"), "invalid stored hash"); err == nil {
		t.Fatal("corrupt password hash accepted")
	}
	if !strings.Contains(logs.String(), "level=error") {
		t.Fatal("corrupt stored hash has no application error log")
	}
}

func TestUserListCountMatchesFilteredPages(t *testing.T) {
	u, db := newTestManager(t)
	for n := 1; n <= 5; n++ {
		if _, err := db.Exec(`INSERT INTO users (type, first_name, last_name, email, created_at, deleted_at)
			VALUES ('contact', 'Contact', '', $1, '2026-01-01'::timestamptz + $2 * interval '1 minute',
			CASE WHEN $2 = 5 THEN NOW() END)`, fmt.Sprintf("contact%d@example.com", n), n); err != nil {
			t.Fatal(err)
		}
	}
	for _, email := range []any{nil, "System"} {
		if _, err := db.Exec(`INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Excluded', '', $1)`, email); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		name, order, filters string
		page, total          int
		want                 []string
	}{
		{"second page", "desc", "[]", 2, 4, []string{"contact2@example.com", "contact1@example.com"}},
		{"ascending", "asc", "[]", 1, 4, []string{"contact1@example.com", "contact2@example.com"}},
		{"exact email", "desc", `[{"model":"users","field":"email","operator":"equals","value":"contact3@example.com"}]`, 1, 1, []string{"contact3@example.com"}},
		{"date filter", "desc", `[{"model":"users","field":"created_at","operator":"equals","value":"2026-01-01"}]`, 1, 4, []string{"contact4@example.com", "contact3@example.com"}},
		{"empty page", "desc", "[]", 3, 0, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := u.GetAllUsers(tt.page, 2 /** pageSize **/, []string{"contact"}, tt.order, "users.created_at", tt.filters, "UTC")
			if err != nil {
				t.Fatal(err)
			}
			var emails []string
			for _, row := range rows {
				emails = append(emails, row.Email.String)
				if row.Total != tt.total {
					t.Fatalf("total = %d, want %d", row.Total, tt.total)
				}
			}
			if !slices.Equal(emails, tt.want) {
				t.Fatalf("emails = %v, want %v", emails, tt.want)
			}
		})
	}
}
