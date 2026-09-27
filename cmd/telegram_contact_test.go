package main

import (
	"encoding/json"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

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
