// Package ssrf builds the shared outbound-request guard used across every
// admin-configured outbound URL (webhooks, OIDC discovery, AI provider and
// custom tool calls). It is off by default and meant to be enabled on
// multi-tenant/hosted deployments where those URLs come from untrusted tenants.
package ssrf

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/abhinavxd/ssrfguard"
	"github.com/zerodha/logf"
)

// Control matches net.Dialer.Control. A nil value means no guarding.
type Control = func(network, address string, c syscall.RawConn) error

// NewControl returns a net.Dialer.Control that blocks connections to private and
// reserved IP ranges, or nil when disabled. allowedCIDRs carve out exceptions
// (e.g. an operator's own internal host) that stay reachable while enabled.
func NewControl(enabled bool, allowedCIDRs []string, lo *logf.Logger) Control {
	if !enabled {
		return nil
	}
	var allowed []netip.Prefix
	for _, c := range allowedCIDRs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(c))
		if err != nil {
			lo.Warn("ignoring invalid ssrf `allowed_cidrs` entry", "entry", c, "error", err)
			continue
		}
		allowed = append(allowed, prefix)
	}
	return ssrfguard.New(allowed...).Control
}

// NewTransport clones http.DefaultTransport (keeping proxy and connection-pool defaults) and applies the SSRF dial guard; callers may override the returned transport's timeouts.
func NewTransport(control Control, dialTimeout time.Duration) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
		Control:   control,
	}).DialContext
	return t
}
