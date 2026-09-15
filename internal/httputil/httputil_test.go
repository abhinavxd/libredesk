package httputil

import "testing"

func TestIsValidHTTPURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "HTTPS", url: "https://desk.example.com", want: true},
		{name: "HTTP localhost", url: "http://localhost:9000", want: true},
		{name: "path", url: "https://desk.example.com/app", want: true},
		{name: "empty", url: "", want: false},
		{name: "missing host", url: "https:", want: false},
		{name: "relative", url: "/app", want: false},
		{name: "unsupported scheme", url: "javascript:alert(1)", want: false},
		{name: "attribute injection", url: `https://desk.example.com" onclick="alert(1)`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidHTTPURL(tt.url); got != tt.want {
				t.Fatalf("IsValidHTTPURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
