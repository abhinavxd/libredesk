package main

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestWidgetWSClosesAfterJoinLimit(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	app := &App{lo: &lo}
	server := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		_ = handleWidgetWS(&fastglue.Request{RequestCtx: ctx, Context: app})
	}}
	go server.Serve(listener)
	t.Cleanup(func() { server.Shutdown() })
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/widget/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	join := WidgetMessage{Type: WidgetMsgTypeJoin, Data: json.RawMessage(`[]`)}
	for range wsMaxJoinsPerConn {
		if err := conn.WriteJSON(join); err != nil {
			t.Fatal(err)
		}
		conn.SetReadDeadline(time.Now().Add(time.Second))
		var msg WidgetMessage
		if err := conn.ReadJSON(&msg); err != nil || msg.Type != WidgetMsgTypeError {
			t.Fatalf("expected rejected join, type=%q err=%v", msg.Type, err)
		}
	}
	conn.WriteJSON(join)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
		t.Fatalf("expected policy violation close after join limit, got %v", err)
	}
}
