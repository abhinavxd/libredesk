package resourcepolicy

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/idna"
)

const (
	BlockAll                   = "block_all"
	Allowlist                  = "allowlist"
	LoadOnReceipt              = "load_on_receipt"
	MaxDomains                 = 100
	DefaultMaxCacheBytes int64 = 10 << 30
	MaxCacheBytes        int64 = (1 << 53) - 1 // Largest integer exactly representable by the settings UI.
)

type Config struct {
	MaxCacheBytes  int64    `json:"max_cache_bytes"`
	Mode           string   `json:"mode"`
	AllowedDomains []string `json:"allowed_domains"`
}

// Update contains only the fields an editor owns; omitted fields are preserved.
type Update struct {
	MaxCacheBytes  *int64    `json:"max_cache_bytes,omitempty"`
	Mode           *string   `json:"mode,omitempty"`
	AllowedDomains *[]string `json:"allowed_domains,omitempty"`
}

func NormalizeUpdate(update Update) (Update, error) {
	if update.MaxCacheBytes == nil && update.Mode == nil && update.AllowedDomains == nil {
		return Update{}, fmt.Errorf("at least one policy field is required")
	}
	cfg := Default()
	if update.MaxCacheBytes != nil {
		cfg.MaxCacheBytes = *update.MaxCacheBytes
	}
	if update.Mode != nil {
		cfg.Mode = *update.Mode
	}
	if update.AllowedDomains != nil {
		cfg.AllowedDomains = *update.AllowedDomains
	}
	cfg, err := Normalize(cfg)
	if err != nil {
		return Update{}, err
	}
	if update.AllowedDomains != nil {
		update.AllowedDomains = &cfg.AllowedDomains
	}
	return update, nil
}

type Policy struct {
	mode  string
	hosts map[string]struct{}
}

func Default() Config {
	return Config{Mode: LoadOnReceipt, AllowedDomains: []string{}, MaxCacheBytes: DefaultMaxCacheBytes}
}

func Blocked() Config {
	return Config{Mode: BlockAll, AllowedDomains: []string{}, MaxCacheBytes: DefaultMaxCacheBytes}
}

func Normalize(cfg Config) (Config, error) {
	if cfg.MaxCacheBytes <= 0 || cfg.MaxCacheBytes > MaxCacheBytes {
		return Blocked(), fmt.Errorf("maximum image cache size must be a positive whole number of bytes no greater than %d", MaxCacheBytes)
	}
	if cfg.Mode != BlockAll && cfg.Mode != Allowlist && cfg.Mode != LoadOnReceipt {
		return Blocked(), fmt.Errorf("mode must be block_all, allowlist or load_on_receipt")
	}
	if len(cfg.AllowedDomains) > MaxDomains {
		return Blocked(), fmt.Errorf("at most %d domains are allowed", MaxDomains)
	}
	out := Config{MaxCacheBytes: cfg.MaxCacheBytes, Mode: cfg.Mode, AllowedDomains: make([]string, 0, len(cfg.AllowedDomains))}
	for _, domain := range cfg.AllowedDomains {
		host := strings.ToLower(strings.TrimSpace(domain))
		if !validHost(host) {
			return Blocked(), fmt.Errorf("allowed domains must be exact ASCII DNS names without schemes, ports, wildcards or trailing dots")
		}
		out.AllowedDomains = append(out.AllowedDomains, host)
	}
	slices.Sort(out.AllowedDomains)
	out.AllowedDomains = slices.Compact(out.AllowedDomains)
	return out, nil
}

func New(cfg Config) (Policy, error) {
	// Callers constructing only a URL policy need not specify a storage budget.
	if cfg.MaxCacheBytes == 0 {
		cfg.MaxCacheBytes = DefaultMaxCacheBytes
	}
	cfg, err := Normalize(cfg)
	if err != nil {
		return Policy{}, err
	}
	p := Policy{mode: cfg.Mode, hosts: make(map[string]struct{}, len(cfg.AllowedDomains))}
	for _, domain := range cfg.AllowedDomains {
		p.hosts[domain] = struct{}{}
	}
	return p, nil
}

func (p Policy) AllowsURL(raw string) bool {
	if !p.PermitsFetching() || !ValidImageURL(raw) {
		return false
	}
	if p.mode == LoadOnReceipt {
		return true
	}
	u, _ := url.Parse(raw)
	_, allowed := p.hosts[strings.ToLower(u.Hostname())]
	return allowed
}

func ValidImageURL(raw string) bool {
	if len(raw) > 8192 {
		return false
	}
	for _, c := range raw {
		if c <= 0x20 || c == 0x7f || c == '\\' {
			return false
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if !validHost(host) {
		return false
	}
	authority := strings.ToLower(u.Host)
	if authority != host && authority != host+":443" {
		return false
	}
	return true
}

func validHost(host string) bool {
	if len(host) > 253 || !strings.Contains(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	tld := labels[len(labels)-1]
	if tld[0] < 'a' || tld[0] > 'z' {
		return false
	}
	ascii, err := idna.Lookup.ToASCII(host)
	return err == nil && ascii == host
}

func (p Policy) PermitsConsent() bool {
	return p.mode == Allowlist
}

func (p Policy) PermitsFetching() bool {
	return p.mode == Allowlist || p.mode == LoadOnReceipt
}
