package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/knadh/go-i18n"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

func newTestAuth(t *testing.T) (*Auth, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	i18nManager, err := i18n.New([]byte(`{"_.code": "en", "_.name": "English"}`))
	if err != nil {
		t.Fatalf("initialising i18n: %v", err)
	}
	logger := logf.New(logf.Opts{})

	return &Auth{
		i18n:   i18nManager,
		logger: &logger,
		rd:     redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}, mr
}

func TestPKCEChallenge(t *testing.T) {
	// Vector from RFC 7636 appendix B.
	const (
		verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		want     = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	)
	if got := PKCEChallenge(verifier); got != want {
		t.Fatalf("PKCEChallenge = %q, want %q", got, want)
	}
}

func TestConsumeLoginCode(t *testing.T) {
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	t.Run("round trip", func(t *testing.T) {
		a, _ := newTestAuth(t)
		code, err := a.MintLoginCode(context.Background(), 42, PKCEChallenge(verifier))
		if err != nil {
			t.Fatalf("MintLoginCode: %v", err)
		}
		userID, err := a.ConsumeLoginCode(context.Background(), code, verifier)
		if err != nil {
			t.Fatalf("ConsumeLoginCode: %v", err)
		}
		if userID != 42 {
			t.Fatalf("user id = %d, want 42", userID)
		}
	})

	t.Run("wrong verifier is rejected and does not burn the code", func(t *testing.T) {
		a, _ := newTestAuth(t)
		code, err := a.MintLoginCode(context.Background(), 42, PKCEChallenge(verifier))
		if err != nil {
			t.Fatalf("MintLoginCode: %v", err)
		}
		if _, err := a.ConsumeLoginCode(context.Background(), code, "not-the-verifier"); err == nil {
			t.Fatal("expected an error for a mismatched verifier")
		}
		// GetDel already removed it, so even the right verifier cannot recover it.
		if _, err := a.ConsumeLoginCode(context.Background(), code, verifier); err == nil {
			t.Fatal("expected the code to be gone after a failed attempt")
		}
	})

	t.Run("a code is single use", func(t *testing.T) {
		a, _ := newTestAuth(t)
		code, err := a.MintLoginCode(context.Background(), 7, PKCEChallenge(verifier))
		if err != nil {
			t.Fatalf("MintLoginCode: %v", err)
		}
		if _, err := a.ConsumeLoginCode(context.Background(), code, verifier); err != nil {
			t.Fatalf("first ConsumeLoginCode: %v", err)
		}
		if _, err := a.ConsumeLoginCode(context.Background(), code, verifier); err == nil {
			t.Fatal("expected the second redemption to fail")
		}
	})

	t.Run("a code expires", func(t *testing.T) {
		a, mr := newTestAuth(t)
		code, err := a.MintLoginCode(context.Background(), 7, PKCEChallenge(verifier))
		if err != nil {
			t.Fatalf("MintLoginCode: %v", err)
		}
		mr.FastForward(loginCodeTTL + time.Second)
		if _, err := a.ConsumeLoginCode(context.Background(), code, verifier); err == nil {
			t.Fatal("expected an expired code to fail")
		}
	})

	t.Run("unknown and empty codes are rejected", func(t *testing.T) {
		a, _ := newTestAuth(t)
		if _, err := a.ConsumeLoginCode(context.Background(), "nope", verifier); err == nil {
			t.Fatal("expected an error for an unknown code")
		}
		if _, err := a.ConsumeLoginCode(context.Background(), "", verifier); err == nil {
			t.Fatal("expected an error for an empty code")
		}
		if _, err := a.ConsumeLoginCode(context.Background(), "nope", ""); err == nil {
			t.Fatal("expected an error for an empty verifier")
		}
	})
}
