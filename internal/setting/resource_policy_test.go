package setting

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestResourcePolicyPersistence(t *testing.T) {
	db := testutil.NewDB(t, "resource_policy")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	assertBlocked := func(wantError bool) {
		t.Helper()
		cfg, err := m.GetResourcePolicy()
		if (err != nil) != wantError || cfg.Mode != resourcepolicy.BlockAll || len(cfg.AllowedDomains) != 0 {
			t.Fatalf("expected blocked policy, got %#v, %v", cfg, err)
		}
	}
	cfg, err := m.GetResourcePolicy()
	if err != nil || cfg.Mode != resourcepolicy.LoadOnReceipt || cfg.MaxCacheBytes != resourcepolicy.DefaultMaxCacheBytes {
		t.Fatalf("unexpected installation default: %#v, %v", cfg, err)
	}
	db.MustExec(`DELETE FROM settings WHERE key = 'security.resource_policy'`)
	cfg, err = m.GetResourcePolicy()
	if err != nil || cfg.Mode != resourcepolicy.LoadOnReceipt || cfg.MaxCacheBytes != resourcepolicy.DefaultMaxCacheBytes {
		t.Fatalf("unexpected missing-setting default: %#v, %v", cfg, err)
	}
	cfg = resourcepolicy.Config{MaxCacheBytes: 2 << 30, Mode: resourcepolicy.Allowlist, AllowedDomains: []string{" IMAGES.example.com "}}
	if err := m.SetResourcePolicy(cfg); err != nil {
		t.Fatal(err)
	}
	if err := m.SetResourcePolicy(resourcepolicy.Config{Mode: "invalid"}); err == nil {
		t.Fatal("accepted invalid policy")
	}
	got, err := m.GetResourcePolicy()
	if err != nil || got.MaxCacheBytes != 2<<30 || got.Mode != resourcepolicy.Allowlist || len(got.AllowedDomains) != 1 || got.AllowedDomains[0] != "images.example.com" {
		t.Fatalf("unexpected persisted policy: %#v, %v", got, err)
	}
	db.MustExec(`UPDATE settings SET value = '{"mode":"allowlist","allowed_domains":[]}'::jsonb WHERE key = 'security.resource_policy'`)
	legacy, err := m.GetResourcePolicy()
	if err != nil || legacy.Mode != resourcepolicy.Allowlist || legacy.MaxCacheBytes != 10<<30 {
		t.Fatalf("legacy policy did not receive default budget: %#v, %v", legacy, err)
	}
	if err := m.SetResourcePolicy(resourcepolicy.Blocked()); err != nil {
		t.Fatal(err)
	}
	assertBlocked(false)
	for _, value := range []string{`null`, `{"mode":"block_all","max_cache_bytes":0}`, `{"mode":"invalid"}`, `{"mode":"allowlist","allowed_domains":["*"]}`} {
		db.MustExec(`UPDATE settings SET value = $1::jsonb WHERE key = 'security.resource_policy'`, value)
		assertBlocked(true)
	}
	db.Close()
	assertBlocked(true)
}
