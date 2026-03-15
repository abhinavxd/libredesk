package httputil

import "testing"

func TestIsIPBlocked(t *testing.T) {
	tests := []struct {
		name      string
		clientIP  string
		blocked   []string
		want      bool
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
