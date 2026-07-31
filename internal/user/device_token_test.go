package user

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"
	"testing"
)

func TestParseDeviceToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantErr  bool
		selector string
		verifier string
	}{
		{name: "valid", token: "ld_abc_def", selector: "abc", verifier: "def"},
		{name: "empty", token: "", wantErr: true},
		{name: "garbage", token: "garbage", wantErr: true},
		{name: "wrong prefix", token: "xx_abc_def", wantErr: true},
		{name: "two parts", token: "ld_abc", wantErr: true},
		{name: "four parts", token: "ld_abc_def_ghi", wantErr: true},
		{name: "empty selector", token: "ld__def", wantErr: true},
		{name: "empty verifier", token: "ld_abc_", wantErr: true},
		{name: "bearer prefix left on", token: "Bearer ld_abc_def", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			selector, verifier, err := parseDeviceToken(tc.token)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got selector=%q verifier=%q", tc.token, selector, verifier)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.token, err)
			}
			if selector != tc.selector || verifier != tc.verifier {
				t.Fatalf("got %q/%q, want %q/%q", selector, verifier, tc.selector, tc.verifier)
			}
		})
	}
}

func TestRandomHexIsDistinctAndSized(t *testing.T) {
	a, err := randomHex(deviceTokenSelectorN)
	if err != nil {
		t.Fatalf("randomHex: %v", err)
	}
	b, err := randomHex(deviceTokenSelectorN)
	if err != nil {
		t.Fatalf("randomHex: %v", err)
	}
	if a == b {
		t.Fatal("two selectors collided")
	}
	if len(a) != deviceTokenSelectorN*2 {
		t.Fatalf("got length %d, want %d", len(a), deviceTokenSelectorN*2)
	}
}

func TestMintedTokenShapeRoundTrips(t *testing.T) {
	selector, err := randomHex(deviceTokenSelectorN)
	if err != nil {
		t.Fatalf("randomHex: %v", err)
	}
	verifier, err := randomHex(deviceTokenVerifierN)
	if err != nil {
		t.Fatalf("randomHex: %v", err)
	}
	token := deviceTokenPrefix + "_" + selector + "_" + verifier

	gotSelector, gotVerifier, err := parseDeviceToken(token)
	if err != nil {
		t.Fatalf("parse of a minted token failed: %v", err)
	}
	if gotSelector != selector || gotVerifier != verifier {
		t.Fatal("round trip changed the parts")
	}

	stored := sha256.Sum256([]byte(verifier))
	presented := sha256.Sum256([]byte(gotVerifier))
	if subtle.ConstantTimeCompare(stored[:], presented[:]) != 1 {
		t.Fatal("verifier hash did not match")
	}

	wrong := sha256.Sum256([]byte(verifier + "x"))
	if subtle.ConstantTimeCompare(stored[:], wrong[:]) == 1 {
		t.Fatal("a wrong verifier matched")
	}
}

func TestTokenDoesNotLeakVerifierInSelector(t *testing.T) {
	selector, _ := randomHex(deviceTokenSelectorN)
	verifier, _ := randomHex(deviceTokenVerifierN)
	token := deviceTokenPrefix + "_" + selector + "_" + verifier
	if strings.Contains(selector, verifier) {
		t.Fatal("selector contains the verifier")
	}
	if strings.Count(token, "_") != deviceTokenParts-1 {
		t.Fatalf("token has %d separators, want %d", strings.Count(token, "_"), deviceTokenParts-1)
	}
}
