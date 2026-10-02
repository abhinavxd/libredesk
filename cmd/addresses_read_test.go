package main

import (
	"encoding/json"
	"fmt"
	"testing"

	amodels "github.com/jakedolan443/fernmail/internal/auth/models"
	"github.com/jakedolan443/fernmail/internal/conversation"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jakedolan443/fernmail/internal/user"
	"github.com/jakedolan443/fernmail/internal/ws"
	wsmodels "github.com/jakedolan443/fernmail/internal/ws/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestMarkAddressReadHTTPAndPersonalBroadcast(t *testing.T) {
	db := testutil.NewDB(t, "mark_address_read_http")
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	users, err := user.New(i18n, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	conversations, err := conversation.New(nil, i18n, nil, nil, nil, nil, nil, nil, nil, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	hub := ws.NewHub(&lo, users)
	app := &App{user: users, conversation: conversations, wsHub: hub, i18n: i18n, lo: &lo}
	var reader, addressID int
	if err := db.Get(&reader, `INSERT INTO users(type,email,first_name,last_name) VALUES('agent','read@example.test','Reader','') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name='Agent'`, reader)
	db.MustExec(`INSERT INTO inboxes(name,channel) VALUES('Mailbox','email')`)
	if err := db.Get(&addressID, `INSERT INTO email_addresses(inbox_id,address) SELECT id,'mail@example.test' FROM inboxes RETURNING id`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO email_address_users(address_id,user_id) VALUES($1,$2)`, addressID, reader)
	self := &ws.Client{ID: reader, Hub: hub, Send: make(chan wsmodels.WSMessage, 8)}
	teammate := &ws.Client{ID: reader + 1, Hub: hub, Send: make(chan wsmodels.WSMessage, 8)}
	hub.AddClient(self)
	hub.AddClient(teammate)
	request := func(id string) *fasthttp.RequestCtx {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Init(&fasthttp.Request{}, nil, nil)
		ctx.SetUserValue("user", amodels.User{ID: reader})
		ctx.SetUserValue("id", id)
		if err := handleMarkAddressRead(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
	ctx := request(fmt.Sprint(addressID))
	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("response=%s", ctx.Response.Body())
	}
	var response struct {
		Data addressReadResponse `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.AddressID != addressID || response.Data.MarkedAt.IsZero() {
		t.Fatalf("response=%+v", response)
	}
	if len(self.Send) != 1 || len(teammate.Send) != 0 {
		t.Fatal("read event was not scoped to current user's sessions")
	}
	var event struct {
		Type string              `json:"type"`
		Data addressReadResponse `json:"data"`
	}
	json.Unmarshal((<-self.Send).Data, &event)
	if event.Type != "address_read" || event.Data.AddressID != addressID {
		t.Fatalf("event=%+v", event)
	}
	for _, id := range []string{"0", "-1", "nope", "99999999999999999999999"} {
		if got := request(id).Response.StatusCode(); got != 400 {
			t.Fatalf("id=%q status=%d", id, got)
		}
	}
	db.MustExec(`DELETE FROM email_address_users WHERE address_id=$1`, addressID)
	if got := request(fmt.Sprint(addressID)).Response.StatusCode(); got != 403 {
		t.Fatalf("denied address status=%d", got)
	}
	if len(self.Send) != 0 || len(teammate.Send) != 0 {
		t.Fatal("failed mutation emitted a read event")
	}
}
