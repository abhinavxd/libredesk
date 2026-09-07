package twofactor

import (
	"testing"
	"time"
)

func TestMatchStep(t *testing.T) {
	now := time.Unix(59, 0)
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	step, ok := matchStep(secret, "287082", now, -1)
	if !ok || step != 1 {
		t.Fatalf("RFC code rejected: %d %v", step, ok)
	}
	if _, ok := matchStep(secret, "287082", now, 1); ok {
		t.Fatal("accepted reused code")
	}
	for _, code := range []string{"", "0", "000000", "-287082", "2870820", "abcdef", "28708 "} {
		if _, ok := matchStep(secret, code, now, -1); ok {
			t.Errorf("accepted invalid code %q", code)
		}
	}
	if _, ok := matchStep(secret, "287082", now.Add(2*time.Minute), -1); ok {
		t.Fatal("accepted expired code")
	}
}

func TestRecoveryCodes(t *testing.T) {
	codes, hashes, err := newRecoveryCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 10 || len(hashes) != 10 {
		t.Fatal("expected ten codes")
	}
	seen := map[string]bool{}
	for i, code := range codes {
		if len(code) != 35 || seen[code] {
			t.Fatal("invalid or duplicate recovery code")
		}
		seen[code] = true
		if recoveryHash(code) != hashes[i] || code == hashes[i] {
			t.Fatal("recovery hash mismatch")
		}
	}
}
