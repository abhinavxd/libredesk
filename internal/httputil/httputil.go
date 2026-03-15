package httputil

import (
	"net"
	"net/url"
	"strings"
)

// IsOriginTrusted checks if the given origin is trusted based on the trusted domains list
// Expects trustedDomains to be a list of domain strings, which can include wildcards.
// Like "*.example.com" or "example.com".
func IsOriginTrusted(origin string, trustedDomains []string) bool {
	if len(trustedDomains) == 0 {
		return false
	}

	originHost, originPort := parseHostPort(origin)
	if originHost == "" {
		return false
	}

	for _, trusted := range trustedDomains {
		trustedHost, trustedPort := parseTrustedDomain(trusted)
		if portMatches(originPort, trustedPort) && hostMatches(originHost, trustedHost) {
			return true
		}
	}

	return false
}

// parseHostPort extracts host and port from origin URL
func parseHostPort(origin string) (host, port string) {
	u, err := url.Parse(strings.ToLower(origin))
	if err != nil {
		return "", ""
	}
	
	host, port, _ = net.SplitHostPort(u.Host)
	if host == "" {
		host = u.Host
	}
	return host, port
}

// parseTrustedDomain extracts host and port from trusted domain entry
func parseTrustedDomain(domain string) (host, port string) {
	domain = strings.ToLower(domain)
	
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		u, err := url.Parse(domain)
		if err != nil {
			return "", ""
		}
		host, port, _ = net.SplitHostPort(u.Host)
		if host == "" {
			host = u.Host
		}
		return host, port
	}
	
	// Handle non-URL patterns (wildcards/domains)
	host, port, _ = net.SplitHostPort(domain)
	if host == "" {
		host = domain
	}
	return host, port
}

// portMatches checks if ports are compatible
func portMatches(originPort, trustedPort string) bool {
	if trustedPort == "" || trustedPort == originPort {
		return true
	}
	return false
}

// hostMatches checks if host matches trusted pattern
func hostMatches(origin, trusted string) bool {
	if trusted == origin {
		return true
	}
	
	if strings.HasPrefix(trusted, "*.") {
		base := trusted[2:]
		return origin == base || strings.HasSuffix(origin, "."+base)
	}
	
	return false
}