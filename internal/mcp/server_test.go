package mcp

import (
	"encoding/json"
	"testing"
)

func TestHandleInitializeAndTools(t *testing.T) {
	srv := Server{
		Name:    "libredesk",
		Version: "test",
		Tools: []Tool{{
			Name:        "ping_tool",
			Description: "echo",
			InputSchema: map[string]any{"type": "object"},
			Handler: func(args map[string]any) (any, error) {
				return map[string]any{"ok": true, "q": StrArg(args, "q")}, nil
			},
		}},
	}

	initRaw := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t"}}}`)
	resp := srv.Handle(initRaw)
	if resp == nil || resp.Error != nil {
		t.Fatalf("initialize: %+v", resp)
	}
	result, _ := resp.Result.(map[string]any)
	if result["protocolVersion"] != ProtocolVersion {
		t.Fatalf("protocolVersion = %v", result["protocolVersion"])
	}

	listRaw := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	list := srv.Handle(listRaw)
	listed, _ := list.Result.(map[string]any)
	tools, _ := listed["tools"].([]map[string]any)
	if len(tools) != 1 || tools[0]["name"] != "ping_tool" {
		t.Fatalf("tools/list = %#v", listed)
	}

	callRaw := []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ping_tool","arguments":{"q":"hi"}}}`)
	call := srv.Handle(callRaw)
	callResult, _ := call.Result.(map[string]any)
	content, _ := callResult["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatalf("tools/call = %#v", call.Result)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(content[0]["text"].(string)), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["q"] != "hi" {
		t.Fatalf("payload = %#v", payload)
	}

	if srv.Handle([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)) != nil {
		t.Fatal("notification should return nil")
	}
}
