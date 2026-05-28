package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyPrefix = "ai_cache:"
	cacheTTL       = 30 * time.Minute
)

func cacheKey(model string, messages []ChatMessage) string {
	h := sha256.New()
	h.Write([]byte(model))
	for _, msg := range messages {
		h.Write([]byte("|" + msg.Role + "|" + msg.Content))
	}
	return cacheKeyPrefix + hex.EncodeToString(h.Sum(nil))
}

func GetCachedReply(rdb *redis.Client, model string, messages []ChatMessage) (string, bool) {
	if rdb == nil || len(messages) == 0 {
		return "", false
	}
	ctx := context.Background()
	reply, err := rdb.Get(ctx, cacheKey(model, messages)).Result()
	if err != nil {
		return "", false
	}
	return reply, true
}

func SetCachedReply(rdb *redis.Client, model string, messages []ChatMessage, reply string) {
	if rdb == nil || len(messages) == 0 {
		return
	}
	ctx := context.Background()
	key := cacheKey(model, messages)
	rdb.Set(ctx, key, reply, cacheTTL)
}

func AIRateLimited(rdb *redis.Client, inboxID int) (bool, error) {
	if rdb == nil {
		return false, nil
	}
	ctx := context.Background()
	key := fmt.Sprintf("ai_rate_limit:inbox:%d", inboxID)

	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 60*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	count := incr.Val()
	if count > 20 {
		return true, nil
	}
	return false, nil
}
