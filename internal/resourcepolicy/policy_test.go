package resourcepolicy

import (
	"slices"
	"strings"
	"testing"
)

func TestAllowsURL(t *testing.T) {
	p, err := New(Config{Mode: Allowlist, AllowedDomains: []string{"images.example.com", "xn--bcher-kva.example"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"https://images.example.com/pixel?user=123",
		"https://IMAGES.EXAMPLE.COM/image.png",
		"https://images.example.com:443/image.png",
		"https://xn--bcher-kva.example/image.png",
	} {
		if !p.AllowsURL(raw) {
			t.Errorf("blocked %q", raw)
		}
	}
	for _, raw := range []string{
		"", "/uploads/image.png", "//images.example.com/image.png",
		"http://images.example.com/image.png", "javascript:alert(1)",
		"data:image/png;base64,abc", "blob:https://images.example.com/id", "cid:image",
		"https://images.example.com.evil.test/x", "https://sub.images.example.com/x",
		"https://example.com/x", "https://evil.test/images.example.com",
		"https://images.example.com@evil.test/x", "https://evil.test@images.example.com/x",
		"https://images.example.com:444/x", "https://images.example.com:/x",
		"https://images.example.com:0443/x", "https://images.example.com./x",
		"https://images%2eexample.com/x", "https://images.example.com\\@evil.test/x",
		"https://images.example.com\n/x", " https://images.example.com/x",
		"https://images.example.com/x\t", "https://images.example.com/x\x00",
		"https:images.example.com/x", "https:///images.example.com/x",
		"https://bücher.example/x", "https://127.0.0.1/x", "https://2130706433/x",
		"https://0x7f000001/x", "https://127.1/x", "https://[::1]/x",
		"https://images.example.com/%zz", "https://images.example.com/" + strings.Repeat("a", 8192),
	} {
		if p.AllowsURL(raw) {
			t.Errorf("allowed %q", raw)
		}
	}
}

func TestPolicyFailsClosed(t *testing.T) {
	for _, cfg := range []Config{
		{}, Blocked(),
		{Mode: Allowlist},
		{Mode: BlockAll, AllowedDomains: []string{"images.example.com"}},
		{Mode: "allow", AllowedDomains: []string{"images.example.com"}},
		{Mode: Allowlist, AllowedDomains: []string{"images.example.com", "*.example.com"}},
	} {
		p, _ := New(cfg)
		if p.AllowsURL("https://images.example.com/image.png") {
			t.Errorf("config %#v allowed a resource", cfg)
		}
	}
	if (Policy{}).AllowsURL("https://images.example.com/image.png") {
		t.Fatal("zero policy allowed a resource")
	}
}

func TestNormalize(t *testing.T) {
	cfg, err := Normalize(Config{Mode: Allowlist, AllowedDomains: []string{" IMAGES.Example.com ", "images.example.com", "a.example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.AllowedDomains, []string{"a.example.com", "images.example.com"}) {
		t.Fatalf("unexpected domains: %v", cfg.AllowedDomains)
	}
	for _, host := range []string{
		"", "*", "*.example.com", "https://example.com", "example.com/path",
		"example.com:443", "example.com.", ".example.com", "a..example.com",
		"example.com@evil.test", "example.com?x", "example.com#x", "example.com\\evil",
		"-a.example", "a-.example", "a_b.example", "a.123", "a.0x7f", "localhost",
		"127.0.0.1", "127.1", "0177.0.0.1", "0x7f000001", "[::1]",
		"bücher.example", "xn--.example", "exam\nple.com", strings.Repeat("a", 64) + ".example",
		strings.Repeat("aaaa.", 51) + "com",
	} {
		if _, err := Normalize(Config{Mode: Allowlist, AllowedDomains: []string{host}}); err == nil {
			t.Errorf("accepted invalid domain %q", host)
		}
	}
	if _, err := Normalize(Config{Mode: Allowlist, AllowedDomains: make([]string, MaxDomains+1)}); err == nil {
		t.Fatal("accepted oversized allowlist")
	}
}

func TestLoadOnReceipt(t *testing.T) {
	if Default().Mode != LoadOnReceipt {
		t.Fatal("new installations must load on receipt")
	}
	p, err := New(Default())
	if err != nil || !p.PermitsFetching() || p.PermitsConsent() {
		t.Fatalf("unexpected receipt policy: %v", err)
	}
	for _, source := range []string{"https://images.example/a", "https://other.example/b"} {
		if !p.AllowsURL(source) {
			t.Fatalf("public HTTPS image rejected: %s", source)
		}
	}
	for _, source := range []string{"http://images.example/a", "https://127.0.0.1/a", "https://[::1]/a", "https://user:pass@images.example/a", "data:image/png;base64,abc"} {
		if p.AllowsURL(source) {
			t.Fatalf("unsafe image accepted: %s", source)
		}
	}
	display := p.PrepareDisplayWithImages(`<img src="https://images.example/a"><script src="https://images.example/s"></script>`, nil, func(string) string { return "/api/image" })
	if display.BlockedImages != 0 || !strings.Contains(display.HTML, `src="/api/image"`) || strings.Contains(display.HTML, "https://") {
		t.Fatalf("unexpected display: %s", display.HTML)
	}
}
