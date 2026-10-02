package media

import (
	"bytes"
	"testing"

	"github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/volatiletech/null/v9"
)

func TestUnlinkedBrandingSweepKeepsSavedAndPendingLogos(t *testing.T) {
	m, store := quotaFixture(t, "branding_sweep", 0)
	upload := func() models.Media {
		t.Helper()
		item, err := m.UploadAndInsert("logo.png", "image/png", "", null.StringFrom(models.ModelBranding), null.Int{}, bytes.NewReader([]byte("\x89PNG\r\n\x1a\n")), 8, null.StringFrom(models.DispositionInline), []byte("{}"), false)
		if err != nil {
			t.Fatal(err)
		}
		if item.Private {
			t.Fatal("logo stored as private media")
		}
		return item
	}
	saved, replaced, pending := upload(), upload(), upload()
	m.db.MustExec(`UPDATE media SET created_at = NOW() - INTERVAL '2 days' WHERE id IN ($1, $2)`, saved.ID, replaced.ID)
	m.db.MustExec(`UPDATE settings SET value = to_jsonb($1::text) WHERE key = 'app.logo_url'`, PublicURI+"/"+saved.UUID)

	m.deleteUnlinked()

	for _, tc := range []struct {
		item     models.Media
		wantKept bool
	}{{saved, true}, {pending, true}, {replaced, false}} {
		item, wantKept := tc.item, tc.wantKept
		var rows int
		if err := m.db.Get(&rows, `SELECT count(*) FROM media WHERE id = $1`, item.ID); err != nil {
			t.Fatal(err)
		}
		_, blob := store.blobs[item.UUID]
		if (rows == 1) != wantKept || blob != wantKept {
			t.Fatalf("media %d: row kept = %v, blob kept = %v, want %v", item.ID, rows == 1, blob, wantKept)
		}
	}
}
