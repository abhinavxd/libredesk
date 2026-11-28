package main

import "testing"

func TestIsAllowedAgentOrigin(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		host    string
		rootURL string
		want    bool
	}{
		{name: "same host", origin: "https://desk.example.com", host: "desk.example.com", want: true},
		{name: "empty origin", origin: "", host: "desk.example.com", want: false},
		{name: "plain http", origin: "http://desk.example.com", host: "desk.example.com", want: false},
		{name: "other host", origin: "https://evil.example.com", host: "desk.example.com", want: false},
		{name: "localhost", origin: "http://localhost:9000", host: "127.0.0.1:9000", want: true},
		{name: "root url on another port", origin: "https://desk.example.com", host: "desk.example.com:8443", rootURL: "https://desk.example.com", want: true},
		{name: "other host with root url", origin: "https://evil.example.com", host: "desk.example.com:8443", rootURL: "https://desk.example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko.Delete("app.root_url")
			if tt.rootURL != "" {
				if err := ko.Set("app.root_url", tt.rootURL); err != nil {
					t.Fatalf("set config: %v", err)
				}
			}
			t.Cleanup(func() { ko.Delete("app.root_url") })

			if got := isAllowedAgentOrigin(tt.origin, tt.host); got != tt.want {
				t.Fatalf("isAllowedAgentOrigin(%q, %q) = %v, want %v", tt.origin, tt.host, got, tt.want)
			}
		})
	}
}

func TestInjectWSURL(t *testing.T) {
	page := []byte("<html><head><title>x</title></head><body></body></html>")

	ko.Delete("app.ws_url")
	if got := string(injectWSURL(page)); got != string(page) {
		t.Fatalf("page changed without app.ws_url: %s", got)
	}

	if err := ko.Set("app.ws_url", "wss://desk.example.com:8443/"); err != nil {
		t.Fatalf("set config: %v", err)
	}
	t.Cleanup(func() { ko.Delete("app.ws_url") })

	want := `<html><head><title>x</title><meta name="libredesk-ws-url" content="wss://desk.example.com:8443"></head><body></body></html>`
	if got := string(injectWSURL(page)); got != want {
		t.Fatalf("injectWSURL() = %s, want %s", got, want)
	}
}
