package aiagent

import (
	"context"
	"encoding/json"
	"fmt"
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
	otpMaxSends    = 3
)

// checkOTPScript atomically reads the pending code, compares it, and either clears it (match or cap
// reached) or increments the attempt count, so concurrent guesses cannot race past otpMaxAttempts.
// Returns 1 on match, 0 on miss/expiry, -1 on corrupt data (key cleared).
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
if p.code == ARGV[1] then
	redis.call('DEL', KEYS[1])
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

// incrOTPSendsScript increments the send counter and sets its TTL on first increment in one atomic
// step, so the key can never persist without an expiry and lock a conversation out of resends.
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
	Attempts int    `json:"attempts"`
}

func otpPendingKey(convUUID string) string  { return otpPendingKeyPrefix + convUUID }
func otpVerifiedKey(convUUID string) string { return otpVerifiedKeyPrefix + convUUID }
func otpSendsKey(convUUID string) string    { return otpSendsKeyPrefix + convUUID }

func (m *Manager) isConversationVerified(convUUID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	v, err := m.redis.Get(ctx, otpVerifiedKey(convUUID)).Result()
	if err != nil {
		if err != redis.Nil {
			m.lo.Error("error reading otp verified key", "conversation_uuid", convUUID, "error", err)
		}
		return false
	}
	return v == "1"
}

func (m *Manager) setConversationVerified(convUUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return m.redis.Set(ctx, otpVerifiedKey(convUUID), "1", otpVerifiedTTL).Err()
}

// incrOTPSends bumps the per-conversation send counter and reports whether the cap is now exceeded.
func (m *Manager) incrOTPSends(convUUID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	n, err := incrOTPSendsScript.Run(ctx, m.redis, []string{otpSendsKey(convUUID)}, int(otpVerifiedTTL.Seconds())).Int64()
	if err != nil {
		return false, err
	}
	return n > otpMaxSends, nil
}

func (m *Manager) storePendingOTP(convUUID, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	b, err := json.Marshal(pendingOTP{Code: code})
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, otpPendingKey(convUUID), b, otpPendingTTL).Err()
}

// checkPendingOTP compares code against the pending code, counting attempts. It returns whether the
// code matched. On a match, expiry, too many attempts, or no pending code, the pending key is cleared
// so a code is single-use and attempts cannot be retried past the cap.
func (m *Manager) checkPendingOTP(convUUID, code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := checkOTPScript.Run(ctx, m.redis, []string{otpPendingKey(convUUID)}, code, otpMaxAttempts).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case 1:
		if err := m.setConversationVerified(convUUID); err != nil {
			return false, err
		}
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
