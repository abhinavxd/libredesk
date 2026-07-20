package aiagent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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
	key := otpSendsKey(convUUID)
	n, err := m.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		m.redis.Expire(ctx, key, otpVerifiedTTL)
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
	key := otpPendingKey(convUUID)
	raw, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	var p pendingOTP
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		m.redis.Del(ctx, key)
		return false, err
	}
	if p.Code == code {
		m.redis.Del(ctx, key)
		if err := m.setConversationVerified(convUUID); err != nil {
			return false, err
		}
		return true, nil
	}
	p.Attempts++
	if p.Attempts >= otpMaxAttempts {
		m.redis.Del(ctx, key)
		return false, nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return false, err
	}
	// Preserve the original TTL so a wrong guess doesn't extend the code's life.
	m.redis.Set(ctx, key, b, redis.KeepTTL)
	return false, nil
}

// generateOTP returns a random 6-digit numeric code.
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
