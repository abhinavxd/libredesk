package user

import (
	"fmt"
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/migrations"
)

func TestImageSenderPermissions(t *testing.T) {
	m, db := newTestManager(t)
	db.MustExec(`DROP TABLE user_image_permissions`)
	for range 2 {
		if err := migrations.V2_9_1(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var first, second int
	if err := db.Get(&first, `INSERT INTO users (type, email, first_name) VALUES ('agent', 'first@example.com', 'First') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&second, `INSERT INTO users (type, email, first_name) VALUES ('agent', 'second@example.com', 'Second') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := m.SetImageSender(first, "sender@example.com", true); err != nil {
			t.Fatal(err)
		}
	}
	got, err := m.GetImageSenders(first)
	if err != nil || !slices.Equal(got, []string{"sender@example.com"}) {
		t.Fatalf("unexpected permissions: %v, %v", got, err)
	}
	db.SetMaxOpenConns(1)
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	got, err = m.GetImageSendersTx(tx, first)
	tx.Rollback()
	if err != nil || !slices.Equal(got, []string{"sender@example.com"}) {
		t.Fatalf("transaction-bound permissions: %v, %v", got, err)
	}
	got, err = m.GetImageSenders(second)
	if err != nil || len(got) != 0 {
		t.Fatalf("trust crossed user boundary: %v, %v", got, err)
	}
	for i := 1; i < 100; i++ {
		if err := m.SetImageSender(first, fmt.Sprintf("sender%d@example.com", i), true); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.SetImageSender(first, "overflow@example.com", true); err == nil {
		t.Fatal("exceeded sender limit")
	}
	if err := m.SetImageSender(first, "sender@example.com", false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetImageSender(first, "overflow@example.com", true); err != nil {
		t.Fatal(err)
	}
	got, err = m.GetImageSenders(first)
	if err != nil || slices.Contains(got, "sender@example.com") || len(got) != 100 {
		t.Fatalf("failed to revoke: %v, %v", got, err)
	}
}
