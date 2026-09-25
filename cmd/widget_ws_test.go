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

func TestWidgetWSClosesRejectedJoinAttempts(t *testing.T) {
	for _, paced := range []bool{false, true} {
		name := "rapid joins"
		if paced {
			name = "repeated rejected joins"
		}
		t.Run(name, func(t *testing.T) {
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
			attempts := wsMaxJoinAttempts
			if !paced {
				attempts = 1
			}
			for i := range attempts {
				if paced && i > 0 {
					time.Sleep(wsMinIntervalJoin + 10*time.Millisecond)
				}
				if err := conn.WriteJSON(WidgetMessage{Type: WidgetMsgTypeJoin, Data: json.RawMessage(`[]`)}); err != nil {
					t.Fatal(err)
				}
				conn.SetReadDeadline(time.Now().Add(time.Second))
				var msg WidgetMessage
				if err := conn.ReadJSON(&msg); err != nil || msg.Type != WidgetMsgTypeError {
					t.Fatalf("expected rejected join, type=%q err=%v", msg.Type, err)
				}
			}
			if !paced {
				conn.WriteJSON(WidgetMessage{Type: WidgetMsgTypeJoin, Data: json.RawMessage(`[]`)})
			}
			conn.SetReadDeadline(time.Now().Add(time.Second))
			if _, _, err := conn.ReadMessage(); err == nil {
				t.Fatal("connection remained open after rejected join limit")
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				t.Fatal("connection was not closed")
			}
		})
	}
}
