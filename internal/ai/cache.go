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

func cacheKey(model, faqData, lastUserMessage string) string {
	h := sha256.New()
	h.Write([]byte(model + "|" + faqData + "|" + lastUserMessage))
	return cacheKeyPrefix + hex.EncodeToString(h.Sum(nil))
}

func GetCachedReply(rdb *redis.Client, model, faqData, lastUserMessage string) (string, bool) {
	if rdb == nil {
		return "", false
	}
	ctx := context.Background()
	reply, err := rdb.Get(ctx, cacheKey(model, faqData, lastUserMessage)).Result()
	if err != nil {
		return "", false
	}
	return reply, true
}

func SetCachedReply(rdb *redis.Client, model, faqData, lastUserMessage, reply string) {
	if rdb == nil {
		return
	}
	ctx := context.Background()
	key := cacheKey(model, faqData, lastUserMessage)
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
