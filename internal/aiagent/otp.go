package aiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/redis/go-redis/v9"
)

const (
	otpPendingKeyPrefix  = "ai:otp:pending:"
	otpVerifiedKeyPrefix = "ai:otp:verified:"
	otpSendsKeyPrefix    = "ai:otp:sends:"

	otpPendingTTL  = 10 * time.Minute
	otpVerifiedTTL = 30 * time.Minute

	otpMaxAttempts = 3
	// otpMaxSends is the per-address nag limit; otpMaxConvSends bounds total outbound mail for the
	// conversation, since set_contact_email lets the customer switch to a fresh address at will.
	otpMaxSends     = 3
	otpMaxConvSends = 6
)

// checkOTPScript matches the pending code and, on match, stores the attested email as the verified
// value, all in one step. Returns 1 on match, 0 on miss/expiry, -1 on corrupt data (key cleared).
var checkOTPScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then
	return 0
end
local ok, p = pcall(cjson.decode, raw)
if not ok or type(p) ~= 'table' then
	redis.call('DEL', KEYS[1])
	return -1
end
if type(p.email) ~= 'string' or p.email == '' then
	redis.call('DEL', KEYS[1])
	return 0
end
if p.code == ARGV[1] then
	redis.call('DEL', KEYS[1])
	redis.call('SET', KEYS[2], p.email, 'EX', ARGV[3])
	return 1
end
p.attempts = (p.attempts or 0) + 1
if p.attempts >= tonumber(ARGV[2]) then
	redis.call('DEL', KEYS[1])
else
	redis.call('SET', KEYS[1], cjson.encode(p), 'KEEPTTL')
end
return 0
`)

// incrOTPSendsScript increments the send counter and sets its TTL on first increment.
var incrOTPSendsScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then
	redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return n
`)

// pendingOTP is the JSON stored at otpPendingKeyPrefix while a code awaits entry.
type pendingOTP struct {
	Code     string `json:"code"`
	Email    string `json:"email"`
	Attempts int    `json:"attempts"`
}

func otpPendingKey(convUUID string) string  { return otpPendingKeyPrefix + convUUID }
func otpVerifiedKey(convUUID string) string { return otpVerifiedKeyPrefix + convUUID }

func normalizeOTPEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// otpSendsKey scopes the send budget to one address so correcting a mistyped email gets a fresh one.
func otpSendsKey(convUUID, email string) string {
	return otpSendsKeyPrefix + convUUID + ":" + normalizeOTPEmail(email)
}

func otpConvSendsKey(convUUID string) string { return otpSendsKeyPrefix + convUUID }

// isConversationVerified holds the invariant "verified == the contact's current email is the one
// proven by OTP": the verified value stores the attested address, and a contact email rebound
// after verification (from this or any sibling conversation) no longer matches.
func (m *Manager) isConversationVerified(convUUID, contactEmail string) bool {
	email := normalizeOTPEmail(contactEmail)
	if email == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	v, err := m.redis.Get(ctx, otpVerifiedKey(convUUID)).Result()
	if err != nil {
		if err != redis.Nil {
			m.lo.Error("error reading otp verified key", "conversation_uuid", convUUID, "error", err)
		}
		return false
	}
	return v == email
}

// clearConversationVerified drops the verified flag and any pending code so a changed email must be
// verified afresh.
func (m *Manager) clearConversationVerified(convUUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return m.redis.Del(ctx, otpVerifiedKey(convUUID), otpPendingKey(convUUID)).Err()
}

// otpSendCapReached reports whether this address or the conversation as a whole is out of sends.
func (m *Manager) otpSendCapReached(convUUID, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	counts, err := m.redis.MGet(ctx, otpSendsKey(convUUID, email), otpConvSendsKey(convUUID)).Result()
	if err != nil {
		return false, err
	}
	return otpCount(counts[0]) >= otpMaxSends || otpCount(counts[1]) >= otpMaxConvSends, nil
}

// incrOTPSends records a code that was actually emailed against both the address and conversation budgets.
func (m *Manager) incrOTPSends(convUUID, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ttl := int(otpVerifiedTTL.Seconds())
	if err := incrOTPSendsScript.Run(ctx, m.redis, []string{otpSendsKey(convUUID, email)}, ttl).Err(); err != nil {
		return err
	}
	return incrOTPSendsScript.Run(ctx, m.redis, []string{otpConvSendsKey(convUUID)}, ttl).Err()
}

func (m *Manager) storePendingOTP(convUUID, code, email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	b, err := json.Marshal(pendingOTP{Code: code, Email: normalizeOTPEmail(email)})
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, otpPendingKey(convUUID), b, otpPendingTTL).Err()
}

// checkPendingOTP matches code against the pending code and, on match, marks the conversation
// verified atomically; the pending key is cleared on match, expiry, or the attempt cap.
func (m *Manager) checkPendingOTP(convUUID, code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := checkOTPScript.Run(ctx, m.redis,
		[]string{otpPendingKey(convUUID), otpVerifiedKey(convUUID)},
		code, otpMaxAttempts, int(otpVerifiedTTL.Seconds())).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case 1:
		return true, nil
	case -1:
		return false, fmt.Errorf("corrupt pending otp for conversation %s", convUUID)
	default:
		return false, nil
	}
}

// generateOTP returns a random 6-digit numeric code.
func generateOTP() (string, error) {
	return stringutil.RandomNumeric(6)
}

// otpCount reads one counter out of an MGet result, treating a missing or unparseable key as zero.
func otpCount(v any) int64 {
	s, ok := v.(string)
	if !ok {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
