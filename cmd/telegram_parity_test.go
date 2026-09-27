package main

import (
	"encoding/json"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"strings"
	"sync"
	"testing"

	businesshours "github.com/abhinavxd/libredesk/internal/business_hours"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	"github.com/abhinavxd/libredesk/internal/telegram"
)

func TestTelegramCardsAndReplyReferences(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var message telegram.Message
	if err := json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"contact":{"first_name":"Shared","last_name":"Contact","phone_number":"+91 90000 00000"}}`), &message); err != nil {
		t.Fatal(err)
	}
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var contactJSON, firstUUID, convUUID string
	if err := db.QueryRow(`SELECT meta->'telegram_contact',uuid, (SELECT uuid FROM conversations WHERE id=conversation_id) FROM conversation_messages WHERE source_id=$1`, message.SourceID(rec.ID)).Scan(&contactJSON, &firstUUID, &convUUID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contactJSON, "90000") {
		t.Fatal(contactJSON)
	}
	var wg sync.WaitGroup
	ids := make(chan int, 10)
	for range 10 {
		wg.Go(func() {
			id, err := app.user.SaveSharedContact("Shared", "Contact", "+91 90000 00000")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- id
		})
	}
	wg.Wait()
	close(ids)
	var first int
	for id := range ids {
		if first == 0 {
			first = id
		}
		if first != id {
			t.Fatal("duplicate contacts")
		}
	}
	id, err := app.user.SaveSharedContact("Different", "Name", "919000000000")
	if err != nil || id != first {
		t.Fatalf("reused contact=%d %v", id, err)
	}
	var name string
	db.Get(&name, `SELECT first_name FROM users WHERE id=$1`, id)
	if name != "Shared" {
		t.Fatal("existing name overwritten")
	}
	if _, err := app.user.SaveSharedContact("", "", "no number"); err == nil {
		t.Fatal("empty phone accepted")
	}
	message.ID = 2
	message.Contact = nil
	message.Location = &telegram.Location{Latitude: 12.97, Longitude: 77.59}
	message.ReplyTo = &telegram.Message{ID: 1, Text: "Shared contact"}
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var target, location string
	db.QueryRow(`SELECT meta->'reply_to'->>'uuid',meta->'telegram_location' FROM conversation_messages WHERE source_id=$1`, message.SourceID(rec.ID)).Scan(&target, &location)
	if target != firstUUID || !strings.Contains(location, "12.97") {
		t.Fatalf("target=%s location=%s", target, location)
	}
	foreignID := telegram.SourceID(rec.ID, 42, 99)
	if err := app.conversation.RecordTelegramSend(firstUUID, []string{telegram.SourceID(rec.ID, 42, 1), foreignID}); err != nil {
		t.Fatal(err)
	}
	target, err = app.conversation.GetTelegramReplyUUID(foreignID, convUUID)
	if err != nil || target != firstUUID {
		t.Fatalf("album reference=%s %v", target, err)
	}
	target, err = app.conversation.GetTelegramReplyUUID(foreignID, "00000000-0000-0000-0000-000000000000")
	if err != nil || target != "" {
		t.Fatal("cross-conversation reference")
	}
	db.Close()
	if _, err := app.conversation.GetTelegramReplyUUID(foreignID, convUUID); err == nil {
		t.Fatal("database failure ignored")
	}
	if _, err := app.user.SaveSharedContact("", "", "12345"); err == nil {
		t.Fatal("database failure ignored")
	}
}

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

func TestTelegramCSATFeedbackAfterRating(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Hello"}`), &message)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	source := message.SourceID(rec.ID)
	db.MustExec(`INSERT INTO csat_responses (conversation_id) SELECT conversation_id FROM conversation_messages WHERE source_id=$1`, source)
	db.MustExec(`UPDATE conversation_messages m SET type='outgoing',meta=jsonb_build_object('is_csat',true,'csat_uuid',r.uuid::text) FROM csat_responses r WHERE r.conversation_id=m.conversation_id AND m.source_id=$1`, source)
	var uuid string
	db.Get(&uuid, `SELECT uuid FROM csat_responses`)
	if err := app.conversation.SubmitTelegramRating(source, 4); err != nil {
		t.Fatal(err)
	}
	survey, err := app.csat.Get(uuid)
	if err != nil || !survey.FeedbackPending() {
		t.Fatal("feedback unavailable after rating")
	}
	if err := app.csat.UpdateResponse(uuid, 1, "", nil); err == nil {
		t.Fatal("rating overwrite allowed")
	}
	if err := app.csat.UpdateResponse(uuid, 1, "Helpful reply", nil); err != nil {
		t.Fatal(err)
	}
	survey, err = app.csat.Get(uuid)
	if err != nil || survey.Rating != 4 || survey.Feedback.String != "Helpful reply" || survey.FeedbackPending() {
		t.Fatalf("survey=%+v %v", survey, err)
	}
	if err := app.csat.UpdateResponse(uuid, 5, "Overwrite", nil); err == nil {
		t.Fatal("feedback overwritten")
	}
	if err := app.conversation.SubmitTelegramRating(source, 5); err != nil {
		t.Fatal(err)
	}
	survey, _ = app.csat.Get(uuid)
	if survey.Rating != 4 {
		t.Fatal("callback overwrote completed survey")
	}
	db.MustExec(`UPDATE settings SET value='"https://new.example.com"' WHERE key='app.root_url'`)
	buttons, err := app.conversation.TelegramCSATButtons(uuid)
	if err != nil || len(buttons) != 6 || buttons[5].URL != "https://new.example.com/csat/"+uuid {
		t.Fatalf("buttons=%+v %v", buttons, err)
	}
}

func TestTelegramSaveContactHandler(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	app.authz, _ = authz.NewEnforcer(app.lo, app.i18n)
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"contact":{"first_name":"Shared","phone_number":"+123456789"}}`), &message)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var actorID int
	db.Get(&actorID, `SELECT id FROM users WHERE email='System'`)
	db.MustExec(`INSERT INTO roles (name,permissions) VALUES ('Test contact saver',ARRAY['conversations:read','conversations:read_all','contacts:write'])`)
	db.MustExec(`INSERT INTO user_roles (user_id,role_id) SELECT $1,id FROM roles WHERE name='Test contact saver'`, actorID)
	var uuid, cuuid string
	db.QueryRow(`SELECT m.uuid,c.uuid FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE m.source_id=$1`, message.SourceID(rec.ID)).Scan(&uuid, &cuuid)
	request := func(actor int, conv, id string) int {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("user", amodels.User{ID: actor})
		ctx.SetUserValue("cuuid", conv)
		ctx.SetUserValue("uuid", id)
		r := &fastglue.Request{RequestCtx: ctx, Context: app}
		if err := handleSaveTelegramContact(r); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(ctx.Response.Body()) {
			t.Fatal("missing envelope")
		}
		return ctx.Response.StatusCode()
	}
	if status := request(actorID, cuuid, uuid); status != 200 {
		t.Fatalf("save status=%d", status)
	}
	if request(999999, cuuid, uuid) == 200 {
		t.Fatal("unknown user allowed")
	}
	if request(actorID, "00000000-0000-0000-0000-000000000000", uuid) == 200 {
		t.Fatal("missing conversation allowed")
	}
	if request(actorID, cuuid, "00000000-0000-0000-0000-000000000000") == 200 {
		t.Fatal("missing message allowed")
	}
	db.MustExec(`UPDATE conversation_messages SET private=true WHERE uuid=$1`, uuid)
	if request(actorID, cuuid, uuid) != 403 {
		t.Fatal("private message allowed")
	}
	db.MustExec(`UPDATE conversation_messages SET private=false,meta='{}' WHERE uuid=$1`, uuid)
	if request(actorID, cuuid, uuid) != 400 {
		t.Fatal("missing contact allowed")
	}
	db.MustExec(`UPDATE conversation_messages SET meta='{"telegram_contact":"invalid"}' WHERE uuid=$1`, uuid)
	if request(actorID, cuuid, uuid) == 200 {
		t.Fatal("invalid contact accepted")
	}
	db.MustExec(`UPDATE conversation_messages SET meta='{"telegram_contact":{"phone_number":"invalid"}}' WHERE uuid=$1`, uuid)
	if request(actorID, cuuid, uuid) == 200 {
		t.Fatal("contact without phone digits accepted")
	}
	db.MustExec(`DELETE FROM user_roles WHERE user_id=$1`, actorID)
	app.user.InvalidateAgentCache(actorID)
	if request(actorID, cuuid, uuid) != 403 {
		t.Fatal("conversation access bypass")
	}
}

func TestTelegramInboxMessageSettings(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	app.businessHours, _ = businesshours.New(businesshours.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	for _, cfg := range []telegramChannel.Config{
		{GreetingMessage: strings.Repeat("x", 4097)},
		{Timezone: "Invalid/Zone"},
		{AwayMessage: "Closed"},
		{BusinessHoursID: -1},
		{AwayMessage: "Closed", BusinessHoursID: 999999},
	} {
		cfg.BotToken = "123:secret"
		rec.Config, _ = json.Marshal(cfg)
		if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err == nil {
			t.Fatalf("accepted %+v", cfg)
		}
	}
	var id int
	db.Get(&id, `INSERT INTO business_hours (name,is_always_open,hours,holidays) VALUES ('Always',true,'{}','[]') RETURNING id`)
	rec.Config, _ = json.Marshal(telegramChannel.Config{BotToken: "123:secret", BusinessHoursID: id, Timezone: "UTC"})
	if err := prepareTelegramInbox(t.Context(), app, &rec, 0); err == nil {
		t.Fatal("invalid getMe response accepted")
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

func TestTelegramReplyHTTPContract(t *testing.T) {
	app, db, rec := newTelegramIntegrationApp(t)
	app.authz, _ = authz.NewEnforcer(app.lo, app.i18n)
	var message telegram.Message
	json.Unmarshal([]byte(`{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"Question"}`), &message)
	if err := ingestTelegramMessage(t.Context(), app, rec, telegramChannel.Config{}, message); err != nil {
		t.Fatal(err)
	}
	var actorID int
	db.Get(&actorID, `SELECT id FROM users WHERE email='System'`)
	db.MustExec(`INSERT INTO roles (name,permissions) VALUES ('Telegram writer',ARRAY['conversations:read','conversations:read_all','messages:write'])`)
	db.MustExec(`INSERT INTO user_roles (user_id,role_id) SELECT $1,id FROM roles WHERE name='Telegram writer'`, actorID)
	var uuid, cuuid string
	db.QueryRow(`SELECT m.uuid,c.uuid FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE m.source_id=$1`, message.SourceID(rec.ID)).Scan(&uuid, &cuuid)
	for _, tc := range []struct {
		name, target string
		buttons      []telegram.Button
		status       int
	}{
		{name: "quoted buttons", target: uuid, buttons: []telegram.Button{{Text: "Yes", Data: "yes"}}, status: 200},
		{name: "missing quote", target: "00000000-0000-0000-0000-000000000000", status: 400},
		{name: "invalid link", buttons: []telegram.Button{{Text: "Unsafe", URL: "javascript:alert(1)"}}, status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.SetUserValue("user", amodels.User{ID: actorID})
			ctx.SetUserValue("cuuid", cuuid)
			ctx.Request.Header.SetContentType("application/json")
			body, _ := json.Marshal(messageReq{SenderType: "agent", Message: "Reply", ReplyToMessageUUID: tc.target, TelegramButtons: tc.buttons})
			ctx.Request.SetBody(body)
			if err := handleSendMessage(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
				t.Fatal(err)
			}
			if ctx.Response.StatusCode() != tc.status {
				t.Fatalf("status=%d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
			}
			if tc.status == 200 && !strings.Contains(string(ctx.Response.Body()), "reply_to") {
				t.Fatal("quote metadata missing")
			}
		})
	}
}
