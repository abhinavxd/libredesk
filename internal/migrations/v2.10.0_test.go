package migrations

import (
	"github.com/abhinavxd/libredesk/internal/testutil"
	"testing"
)

func TestWidgetCampaignMigration(t *testing.T) {
	db := testutil.NewDB(t, "widget_migration")
	db.MustExec(`DROP TABLE widget_campaign_deliveries`)
	for range 2 {
		if err := V2_10_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'widget_campaign_deliveries'`); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("expected primary key and three indexes, got %d", count)
	}
}
