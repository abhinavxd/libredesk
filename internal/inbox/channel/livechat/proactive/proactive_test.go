package proactive

import (
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
)

func TestConcurrentDeliveryAndOwnership(t *testing.T) {
	db := testutil.NewDB(t, "widget_delivery")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var inboxID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes(name,channel) VALUES ('Test','livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	ctx := Context{BrowserKey: uuid.NewString(), SessionKey: uuid.NewString(), Now: time.Now()}
	c := Campaign{ID: uuid.NewString(), Repeat: "once"}
	var wg sync.WaitGroup
	var deliveries []Delivery
	for range 12 {
		wg.Go(func() {
			unlock := m.Lock()
			defer unlock()
			history, err := m.History(inboxID, ctx)
			if err != nil {
				t.Error(err)
				return
			}
			if Suppression(c, ctx, history, 24*time.Hour) != "" {
				return
			}
			delivery, err := m.Reserve(inboxID, c, ctx, Snapshot{Message: "Hello"})
			if err != nil {
				t.Error(err)
				return
			}
			deliveries = append(deliveries, delivery)
		})
	}
	wg.Wait()
	if len(deliveries) != 1 {
		t.Fatalf("got %d reservations", len(deliveries))
	}
	d := deliveries[0]
	if _, err := m.Get(d.ID, inboxID, uuid.NewString(), 0); err == nil {
		t.Fatal("another browser accessed delivery")
	}
	if _, err := m.Get(d.ID, inboxID+1, ctx.BrowserKey, 0); err == nil {
		t.Fatal("another inbox accessed delivery")
	}
	for range 2 {
		for _, event := range []string{"displayed", "opened", "dismissed"} {
			if err := m.RecordEvent(d.ID, event); err != nil {
				t.Fatal(err)
			}
		}
	}
	stats, err := m.Stats(inboxID, ctx.Now.Add(-time.Minute), ctx.Now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 || stats[0].Displayed != 1 || stats[0].Opened != 1 || stats[0].Dismissed != 1 || stats[0].Replied != 0 {
		t.Fatalf("duplicate events inflated stats: %+v", stats)
	}
	var contactID int
	if err := db.Get(&contactID, `INSERT INTO users(type,first_name) VALUES ('contact','Test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	ctx.ContactID = contactID
	if _, err := m.History(inboxID, ctx); err != nil {
		t.Fatal(err)
	}
	ctx.BrowserKey = uuid.NewString()
	history, err := m.History(inboxID, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if Suppression(c, ctx, history, 24*time.Hour) != "repeat" {
		t.Fatal("identified contact lost delivery history")
	}
}
