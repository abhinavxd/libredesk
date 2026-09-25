package media

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestPendingMediaOwnership(t *testing.T) {
	db := testutil.NewDB(t, "media_ownership")
	var q struct {
		Link  *sqlx.Stmt `query:"link-message-media"`
		Draft *sqlx.Stmt `query:"get-draft-inline-media"`
	}
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, i18n: testutil.NewI18n(t)}
	m.queries.LinkMessageMedia, m.queries.GetDraftInlineMedia = q.Link, q.Draft
	var owner, other int
	for _, id := range []*int{&owner, &other} {
		if err := db.Get(id, `INSERT INTO users (type, first_name, last_name) VALUES ('agent', 'Agent', '') RETURNING id`); err != nil {
			t.Fatal(err)
		}
	}
	for _, uploader := range []int{owner, other, 0} {
		var med models.Media
		if err := db.Get(&med, `INSERT INTO media (store, filename, content_type, content_id, size, model_type, uploaded_by) VALUES ('fs', 'draft.png', 'image/png', '', 1, 'messages', NULLIF($1, 0)) RETURNING id, uuid`, uploader); err != nil {
			t.Fatal(err)
		}
		_, err := m.GetDraftInlineMedia(med.UUID, 0 /** conversationID **/, owner)
		if (err == nil) != (uploader == owner) {
			t.Fatalf("draft access for uploader %d: %v", uploader, err)
		}
		tx, err := db.Beginx()
		if err != nil {
			t.Fatal(err)
		}
		if err := m.LinkMessageMediaTx(tx, 10 /** messageID **/, []models.Media{med, med}, []string{med.UUID, med.UUID}, owner); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		var linked null.Int
		if err := tx.Get(&linked, `SELECT model_id FROM media WHERE id = $1`, med.ID); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		tx.Rollback()
		if linked.Valid != (uploader == owner) {
			t.Fatalf("media linked for uploader %d: %v", uploader, linked)
		}
	}
}
