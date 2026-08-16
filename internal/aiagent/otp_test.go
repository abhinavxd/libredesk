package aiagent

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

func newOTPTestManager(t *testing.T) (*Manager, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	lo := logf.New(logf.Opts{})
	return &Manager{
		lo:    &lo,
		redis: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}, mr
}

func TestIsConversationVerified(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv = "conv-a"

	if m.isConversationVerified(conv, "john@example.com") {
		t.Error("no verified key must mean unverified")
	}

	mr.Set(otpVerifiedKey(conv), "john@example.com")
	if !m.isConversationVerified(conv, "john@example.com") {
		t.Error("matching email must be verified")
	}
	if !m.isConversationVerified(conv, "  John@Example.COM  ") {
		t.Error("email comparison must be case- and whitespace-insensitive")
	}

	if m.isConversationVerified(conv, "victim@example.com") {
		t.Error("a rebound contact email must not stay verified")
	}
	if m.isConversationVerified(conv, "") {
		t.Error("empty contact email must be unverified")
	}
	if m.isConversationVerified("conv-b", "john@example.com") {
		t.Error("another conversation must not inherit the verified flag")
	}

	// Value written by a build that stored a bare flag instead of the email.
	mr.Set(otpVerifiedKey("conv-legacy"), "1")
	if m.isConversationVerified("conv-legacy", "john@example.com") {
		t.Error("legacy flag value must be unverified")
	}
}

func TestCheckPendingOTP(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv, email, code = "conv-a", "john@example.com", "123456"

	ok, err := m.checkPendingOTP(conv, code)
	if err != nil || ok {
		t.Errorf("check without a pending code = (%v, %v), want (false, nil)", ok, err)
	}

	if err := m.storePendingOTP(conv, code, "  John@Example.COM "); err != nil {
		t.Fatal(err)
	}
	ok, err = m.checkPendingOTP(conv, "999999")
	if err != nil || ok {
		t.Errorf("wrong code = (%v, %v), want (false, nil)", ok, err)
	}
	ok, err = m.checkPendingOTP(conv, code)
	if err != nil || !ok {
		t.Fatalf("correct code = (%v, %v), want (true, nil)", ok, err)
	}
	if got, _ := mr.Get(otpVerifiedKey(conv)); got != email {
		t.Errorf("verified value = %q, want the normalized attested email %q", got, email)
	}
	if mr.Exists(otpPendingKey(conv)) {
		t.Error("pending key must be deleted on match")
	}
	if !m.isConversationVerified(conv, email) {
		t.Error("conversation must be verified after a correct code")
	}

	ok, err = m.checkPendingOTP(conv, code)
	if err != nil || ok {
		t.Errorf("code replay = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestCheckPendingOTPAttemptCap(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv, code = "conv-a", "123456"

	if err := m.storePendingOTP(conv, code, "john@example.com"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < otpMaxAttempts; i++ {
		if ok, _ := m.checkPendingOTP(conv, "000000"); ok {
			t.Fatal("wrong code must not verify")
		}
	}
	if mr.Exists(otpPendingKey(conv)) {
		t.Error("pending key must be deleted at the attempt cap")
	}
	if ok, _ := m.checkPendingOTP(conv, code); ok {
		t.Error("correct code must not verify after the attempt cap")
	}
}

func TestCheckPendingOTPLegacyAndCorruptPayloads(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv = "conv-a"

	// Pending record from a build that stored no attested email.
	mr.Set(otpPendingKey(conv), `{"code":"123456"}`)
	ok, err := m.checkPendingOTP(conv, "123456")
	if err != nil || ok {
		t.Errorf("legacy pending without email = (%v, %v), want (false, nil)", ok, err)
	}
	if mr.Exists(otpPendingKey(conv)) {
		t.Error("legacy pending record must be deleted")
	}
	if mr.Exists(otpVerifiedKey(conv)) {
		t.Error("legacy pending record must never set the verified key")
	}

	mr.Set(otpPendingKey(conv), "not-json")
	if ok, err := m.checkPendingOTP(conv, "123456"); err == nil || ok {
		t.Errorf("corrupt pending = (%v, %v), want (false, error)", ok, err)
	}
	if mr.Exists(otpPendingKey(conv)) {
		t.Error("corrupt pending record must be deleted")
	}
}

func TestClearConversationVerified(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv, email = "conv-a", "john@example.com"

	mr.Set(otpVerifiedKey(conv), email)
	if err := m.storePendingOTP(conv, "123456", email); err != nil {
		t.Fatal(err)
	}
	if err := m.clearConversationVerified(conv); err != nil {
		t.Fatal(err)
	}
	if m.isConversationVerified(conv, email) {
		t.Error("conversation must be unverified after clear")
	}
	if mr.Exists(otpPendingKey(conv)) {
		t.Error("pending key must be deleted on clear")
	}
}

// The cross-conversation rebind: verified on conversation A must not survive the shared
// contact's email being rewritten from a sibling conversation.
func TestVerifiedDoesNotSurviveEmailRebind(t *testing.T) {
	m, _ := newOTPTestManager(t)
	const convA, attacker, victim = "conv-a", "attacker@evil.com", "victim@example.com"

	if err := m.storePendingOTP(convA, "123456", attacker); err != nil {
		t.Fatal(err)
	}
	if ok, _ := m.checkPendingOTP(convA, "123456"); !ok {
		t.Fatal("verification must succeed for the attested email")
	}
	if !m.isConversationVerified(convA, attacker) {
		t.Fatal("conversation A must be verified for the attested email")
	}

	// Sibling conversation B rewrites the shared contact row; A's Redis keys are untouched.
	if m.isConversationVerified(convA, victim) {
		t.Error("conversation A must not stay verified once the contact email is rebound")
	}
}

func TestVerifiedTTL(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv, email = "conv-a", "john@example.com"

	if err := m.storePendingOTP(conv, "123456", email); err != nil {
		t.Fatal(err)
	}
	if got := mr.TTL(otpPendingKey(conv)); got != otpPendingTTL {
		t.Errorf("pending TTL = %v, want %v", got, otpPendingTTL)
	}
	if ok, _ := m.checkPendingOTP(conv, "123456"); !ok {
		t.Fatal("verification must succeed")
	}
	if got := mr.TTL(otpVerifiedKey(conv)); got != otpVerifiedTTL {
		t.Errorf("verified TTL = %v, want %v", got, otpVerifiedTTL)
	}
	mr.FastForward(otpVerifiedTTL + time.Second)
	if m.isConversationVerified(conv, email) {
		t.Error("conversation must be unverified after the verified TTL")
	}
}

func TestWrongAttemptKeepsPendingTTL(t *testing.T) {
	m, mr := newOTPTestManager(t)
	const conv = "conv-a"

	if err := m.storePendingOTP(conv, "123456", "john@example.com"); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(otpPendingTTL / 2)
	if ok, _ := m.checkPendingOTP(conv, "000000"); ok {
		t.Fatal("wrong code must not verify")
	}
	if got := mr.TTL(otpPendingKey(conv)); got != otpPendingTTL/2 {
		t.Errorf("pending TTL after a wrong attempt = %v, want %v", got, otpPendingTTL/2)
	}
}

// Guards against the redis client failing open on connection errors.
func TestVerifiedFailsClosedOnRedisError(t *testing.T) {
	m, mr := newOTPTestManager(t)
	mr.Close()
	if m.isConversationVerified("conv-a", "john@example.com") {
		t.Error("a redis error must mean unverified")
	}
}

func TestNormalizeOTPEmail(t *testing.T) {
	cases := map[string]string{
		"  John@Example.COM ": "john@example.com",
		"":                    "",
		"a@b.c":               "a@b.c",
	}
	for in, want := range cases {
		if got := normalizeOTPEmail(in); got != want {
			t.Errorf("normalizeOTPEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
