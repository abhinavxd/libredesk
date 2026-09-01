// Package mcp implements a small JSON-RPC MCP server (initialize, tools/list, tools/call).
package mcp

import (
	"encoding/json"
	"fmt"
)

const ProtocolVersion = "2024-11-05"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     func(args map[string]any) (any, error) `json:"-"`
}

type Server struct {
	Name    string
	Version string
	Tools   []Tool
}

func (s *Server) Handle(raw []byte) *Response {
	if len(raw) == 0 {
		return nil
	}
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return &Response{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: "parse error"}}
	}
	if req.Method == "" || req.ID == nil {
		// Notifications (no id) are acknowledged with no body.
		return nil
	}
	switch req.Method {
	case "initialize":
		return s.result(req.ID, map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
		})
	case "ping":
		return s.result(req.ID, map[string]any{})
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.Tools))
		for _, t := range s.Tools {
			schema := t.InputSchema
			if schema == nil {
				schema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tools = append(tools, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": schema,
			})
		}
		return s.result(req.ID, map[string]any{"tools": tools})
	case "tools/call":
		return s.callTool(req)
	default:
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32601, Message: "method not found"}}
	}
}

func (s *Server) callTool(req Request) *Response {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32602, Message: "invalid params"}}
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}
	for _, t := range s.Tools {
		if t.Name != params.Name {
			continue
		}
		out, err := t.Handler(params.Arguments)
		if err != nil {
			return s.result(req.ID, map[string]any{
				"content": []map[string]any{{"type": "text", "text": err.Error()}},
				"isError": true,
			})
		}
		text, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return s.result(req.ID, map[string]any{
				"content": []map[string]any{{"type": "text", "text": fmt.Sprint(out)}},
			})
		}
		return s.result(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": string(text)}},
		})
	}
	return s.result(req.ID, map[string]any{
		"content": []map[string]any{{"type": "text", "text": "unknown tool: " + params.Name}},
		"isError": true,
	})
}

func (s *Server) result(id json.RawMessage, result any) *Response {
	return &Response{JSONRPC: "2.0", ID: id, Result: result}
}

func StrArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func IntArg(args map[string]any, key string, fallback int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return fallback
}
