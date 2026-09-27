package main

import (
	"encoding/json"
	"testing"

	businesshours "github.com/abhinavxd/libredesk/internal/business_hours"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	"github.com/abhinavxd/libredesk/internal/telegram"
)

func TestTelegramGreetingAndAwayReplies(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var err error
	app.businessHours, err = businesshours.New(businesshours.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &message)
	cfg := telegramChannel.Config{GreetingMessage: "Welcome"}
	for range 2 {
		if err := ingestTelegramMessage(t.Context(), app, rec, cfg, message); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE meta->>'telegram_automatic_reply'='greeting'`)
	if count != 1 {
		t.Fatalf("greetings=%d", count)
	}
	var hoursID int
	db.Get(&hoursID, `INSERT INTO business_hours (name,is_always_open,hours,holidays) VALUES ('Closed',false,'{}','[]') RETURNING id`)
	cfg.AwayMessage = "We are closed"
	cfg.BusinessHoursID = hoursID
	cfg.Timezone = "Asia/Kolkata"
	message.ID++
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, message); err != nil {
		t.Fatal(err)
	}
	message.ID++
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, message); err != nil {
		t.Fatal(err)
	}
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE meta->>'telegram_automatic_reply'='away'`)
	if count != 1 {
		t.Fatalf("away replies=%d", count)
	}
	db.MustExec(`UPDATE conversation_messages SET created_at=NOW()-INTERVAL '2 days' WHERE meta->>'telegram_automatic_reply'='away'`)
	message.ID++
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, message); err != nil {
		t.Fatal(err)
	}
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE meta->>'telegram_automatic_reply'='away'`)
	if count != 2 {
		t.Fatalf("next day replies=%d", count)
	}
	db.MustExec(`UPDATE conversation_messages SET meta='{}' WHERE type='outgoing'`)
	message.ID++
	if err := ingestTelegramMessage(t.Context(), app, rec, cfg, message); err != nil {
		t.Fatal(err)
	}
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE type='outgoing'`)
	if count != 3 {
		t.Fatal("away interrupted active agent conversation")
	}
}

func TestTelegramAutomaticReplyFailures(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	app.businessHours, _ = businesshours.New(businesshours.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &message)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	source := message.SourceID(rec.ID)
	cfg := telegramChannel.Config{GreetingMessage: "Welcome"}
	if err := sendTelegramAutomaticReply(app, rec, cfg, "missing"); err != nil {
		t.Fatal(err)
	}
	cfg.AwayMessage = "Closed"
	cfg.BusinessHoursID = 999999
	if err := sendTelegramAutomaticReply(app, rec, cfg, source); err == nil {
		t.Fatal("missing schedule ignored")
	}
	db.Get(&cfg.BusinessHoursID, `INSERT INTO business_hours (name,is_always_open,hours,holidays) VALUES ('Broken schedule',false,'[]','[]') RETURNING id`)
	if err := sendTelegramAutomaticReply(app, rec, cfg, source); err == nil {
		t.Fatal("invalid schedule ignored")
	}
	cfg.AwayMessage = ""
	db.MustExec(`UPDATE users SET email='RenamedSystem' WHERE email='System'`)
	if err := sendTelegramAutomaticReply(app, rec, cfg, source); err == nil {
		t.Fatal("missing system actor ignored")
	}
	db.MustExec(`UPDATE users SET email='System' WHERE email='RenamedSystem'`)
	message.ID = 2
	message.Text = ""
	message.Venue = &telegram.Venue{Title: "Office", Address: "Example Street", Location: telegram.Location{Latitude: 12, Longitude: 77}}
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var title string
	db.Get(&title, `SELECT meta->'telegram_location'->>'title' FROM conversation_messages WHERE source_id=$1`, message.SourceID(rec.ID))
	if title != "Office" {
		t.Fatal("venue not stored")
	}
	db.Close()
	if err := sendTelegramAutomaticReply(app, rec, cfg, source); err == nil {
		t.Fatal("state lookup failure ignored")
	}
	if _, err := app.conversation.TelegramCSATButtons("survey"); err == nil {
		t.Fatal("root URL lookup failure ignored")
	}
}
