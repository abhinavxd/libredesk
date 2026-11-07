package httputil

import (
	"net"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestIsIPBlocked(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		blocked  []string
		want     bool
	}{
		{"empty list", "1.2.3.4", nil, false},
		{"exact match", "10.0.0.1", []string{"10.0.0.1"}, true},
		{"no match", "10.0.0.2", []string{"10.0.0.1"}, false},
		{"CIDR match", "192.168.1.50", []string{"192.168.1.0/24"}, true},
		{"CIDR no match", "192.168.2.1", []string{"192.168.1.0/24"}, false},
		{"mixed list match IP", "10.0.0.1", []string{"192.168.0.0/16", "10.0.0.1"}, true},
		{"mixed list match CIDR", "192.168.5.5", []string{"192.168.0.0/16", "10.0.0.1"}, true},
		{"mixed list no match", "172.16.0.1", []string{"192.168.0.0/16", "10.0.0.1"}, false},
		{"invalid client IP", "not-an-ip", []string{"10.0.0.1"}, false},
		{"invalid entry skipped", "10.0.0.1", []string{"bad-entry", "10.0.0.1"}, true},
		{"IPv6 exact match", "::1", []string{"::1"}, true},
		{"IPv6 CIDR match", "2001:db8::1", []string{"2001:db8::/32"}, true},
		{"IPv6 no match", "2001:db9::1", []string{"2001:db8::/32"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIPBlocked(tt.clientIP, tt.blocked); got != tt.want {
				t.Errorf("IsIPBlocked(%q, %v) = %v, want %v", tt.clientIP, tt.blocked, got, tt.want)
			}
		})
	}
}

func TestValidateIPOrCIDR(t *testing.T) {
	tests := []struct {
		entry string
		want  bool
	}{
		{"10.0.0.1", true},
		{"192.168.1.0/24", true},
		{"::1", true},
		{"2001:db8::/32", true},
		{"not-valid", false},
		{"10.0.0.1/33", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			if got := ValidateIPOrCIDR(tt.entry); got != tt.want {
				t.Errorf("ValidateIPOrCIDR(%q) = %v, want %v", tt.entry, got, tt.want)
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
		want   string
	}{
		{"default", "", "203.0.113.1", "203.0.113.1"},
		{"railway", "X-Real-IP", "203.0.113.2", "203.0.113.2"},
		{"custom", "CF-Connecting-IP", "203.0.113.3", "203.0.113.3"},
		{"lowercase header", "x-real-ip", "203.0.113.2", "203.0.113.2"},
		{"IPv6", "X-Real-IP", "2001:db8::1", "2001:db8::1"},
		{"private IP", "X-Real-IP", "10.0.0.1", "10.0.0.1"},
		{"whitespace", "X-Real-IP", " 203.0.113.2 ", "203.0.113.2"},
		{"missing", "X-Real-IP", "", "192.0.2.1"},
		{"invalid", "X-Real-IP", "invalid", "192.0.2.1"},
		{"list", "X-Real-IP", "203.0.113.2, 203.0.113.3", "192.0.2.1"},
		{"duplicate IPs", "X-Real-IP", "203.0.113.2, 203.0.113.2", "192.0.2.1"},
		{"zero", "X-Real-IP", "0", "192.0.2.1"},
		{"negative", "X-Real-IP", "-1", "192.0.2.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.Init(&fasthttp.Request{}, &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9000}, nil /** logger **/)
			for _, header := range []string{"X-Client-IP", "X-Original-Forwarded-For", "X-Forwarded-For", "CF-Connecting-IP", "X-Real-IP"} {
				ctx.Request.Header.Set(header, "198.51.100.99")
			}
			header := tt.header
			if header == "" {
				header = "X-Client-IP"
			}
			ctx.Request.Header.Del(header)
			if tt.value != "" {
				ctx.Request.Header.Set(header, tt.value)
			}
			if got := ClientIP(&ctx, tt.header); got != tt.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
