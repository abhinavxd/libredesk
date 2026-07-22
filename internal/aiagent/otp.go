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

// checkOTPScript matches the pending code and sets the verified flag on match, all in one step.
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
	redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
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

// clearConversationVerified drops the verified flag and any pending code so a changed email must be
// verified afresh.
func (m *Manager) clearConversationVerified(convUUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return m.redis.Del(ctx, otpVerifiedKey(convUUID), otpPendingKey(convUUID)).Err()
}

// otpSendCapReached reports whether otpMaxSends codes have already been emailed for the conversation.
func (m *Manager) otpSendCapReached(convUUID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	n, err := m.redis.Get(ctx, otpSendsKey(convUUID)).Int64()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return n >= otpMaxSends, nil
}

// incrOTPSends records a code that was actually emailed, bumping the send counter and setting its TTL
// on the first send.
func (m *Manager) incrOTPSends(convUUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return incrOTPSendsScript.Run(ctx, m.redis, []string{otpSendsKey(convUUID)}, int(otpVerifiedTTL.Seconds())).Err()
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
