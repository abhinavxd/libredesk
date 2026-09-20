package main

import (
	"strings"
	"testing"
)

func TestDecodeResourcePolicy(t *testing.T) {
	for _, body := range []string{
		`{"mode":"block_all","allowed_domains":[]}`,
		`{"max_cache_bytes":1024}`,
		`{"mode":"load_on_receipt","allowed_domains":[]}`,
		`{"mode":"allowlist","allowed_domains":["IMAGES.example.com"]}`,
	} {
		if _, err := decodeResourcePolicy([]byte(body)); err != nil {
			t.Errorf("valid request rejected: %v", err)
		}
	}
	for _, body := range []string{
		``, `null`, `{}`, `[]`, `{"mode":true}`,
		`{"mode":"allow_all"}`, `{"mode":"allowlist","allowed_domains":["*"]}`,
		`{"mode":"allowlist","allowed_domains":[null]}`,
		`{"mode":"allowlist","allowed_domains":"example.com"}`,
		`{"mode":"allowlist","allowed_domain":["example.com"]}`,
		`{"mode":"block_all"} {"mode":"allowlist"}`, `{"mode":"block_all"} trailing`,
		strings.Repeat(" ", 32769),
	} {
		if _, err := decodeResourcePolicy([]byte(body)); err == nil {
			t.Errorf("invalid request accepted: %q", body)
		}
	}
}

func TestDecodeResourceCacheBudget(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int64
	}{
		{`{"mode":"load_on_receipt"}`, 0},
		{`{"mode":"allowlist","max_cache_bytes":10737418240}`, 10 << 30},
		{`{"mode":"block_all","max_cache_bytes":536870912}`, 1 << 29},
	} {
		cfg, err := decodeResourcePolicy([]byte(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		if tc.want == 0 {
			if cfg.MaxCacheBytes != nil {
				t.Fatal("omitted cache size must stay omitted")
			}
		} else if cfg.MaxCacheBytes == nil || *cfg.MaxCacheBytes != tc.want {
			t.Fatalf("unexpected budget: %+v", cfg)
		}
	}
	for _, value := range []string{"0", "-1", "1.5", "9007199254740992", `"10"`} {
		if _, err := decodeResourcePolicy([]byte(`{"mode":"load_on_receipt","max_cache_bytes":` + value + `}`)); err == nil {
			t.Fatalf("accepted invalid budget %s", value)
		}
	}
}
