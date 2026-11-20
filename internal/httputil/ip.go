package httputil

import (
	"cmp"
	"net"
	"strings"

	"github.com/valyala/fasthttp"
)

// IsIPBlocked checks if the given IP address matches any entry in the blocked list.
// Entries can be individual IPs ("10.0.0.1") or CIDR ranges ("192.168.1.0/24").
func IsIPBlocked(clientIP string, blockedIPs []string) bool {
	if len(blockedIPs) == 0 {
		return false
	}
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	for _, entry := range blockedIPs {
		if _, network, err := net.ParseCIDR(entry); err == nil {
			if network.Contains(ip) {
				return true
			}
			continue
		}
		if blocked := net.ParseIP(entry); blocked != nil {
			if blocked.Equal(ip) {
				return true
			}
		}
	}
	return false
}

// ValidateIPOrCIDR checks if a string is a valid IP address or CIDR range.
func ValidateIPOrCIDR(entry string) bool {
	if net.ParseIP(entry) != nil {
		return true
	}
	_, _, err := net.ParseCIDR(entry)
	return err == nil
}

// ClientIP reads only the configured proxy header, falling back to the connection IP.
func ClientIP(ctx *fasthttp.RequestCtx, header string) string {
	value := strings.TrimSpace(string(ctx.Request.Header.Peek(cmp.Or(header, "X-Client-IP"))))
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return ctx.RemoteIP().String()
}
