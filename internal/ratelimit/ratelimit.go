package ratelimit

import (
	"fmt"
	"strconv"
	"time"

	realip "github.com/ferluci/fast-realip"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

// Rule defines a rate limiting rule for a named group of endpoints.
type Rule struct {
	Name              string
	Enabled           bool
	RequestsPerMinute int
}

// Limiter handles rate limiting using Redis.
type Limiter struct {
	redis *redis.Client
	rules map[string]Rule
}

// New creates a new rate limiter.
func New(redisClient *redis.Client) *Limiter {
	return &Limiter{
		redis: redisClient,
		rules: make(map[string]Rule),
	}
}

// AddRule registers a named rate limiting rule.
func (l *Limiter) AddRule(rule Rule) {
	l.rules[rule.Name] = rule
}

// Check checks if the request should be rate limited for the given rule.
func (l *Limiter) Check(ctx *fasthttp.RequestCtx, ruleName string) error {
	rule, ok := l.rules[ruleName]
	if !ok || !rule.Enabled {
		return nil
	}

	clientIP := realip.FromRequest(ctx)
	key := fmt.Sprintf("rate_limit:%s:%s", ruleName, clientIP)

	// Use sliding window approach with Redis.
	now := time.Now().Unix()
	windowStart := now - 60 // 60 seconds window

	count, err := l.redis.ZCount(ctx, key, strconv.FormatInt(windowStart, 10), "+inf").Result()
	if err != nil {
		return nil
	}

	if count >= int64(rule.RequestsPerMinute) {
		ctx.Response.Header.Set("X-RateLimit-Limit", strconv.Itoa(rule.RequestsPerMinute))
		ctx.Response.Header.Set("X-RateLimit-Remaining", "0")
		ctx.Response.Header.Set("X-RateLimit-Reset", strconv.FormatInt(now+60, 10))
		ctx.Response.Header.Set("Retry-After", "60")

		ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
		ctx.SetBodyString(`{"status":"error","message":"Rate limit exceeded"}`)
		return fmt.Errorf("rate limit exceeded")
	}

	// Add current request to the sliding window.
	pipe := l.redis.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: time.Now().UnixNano()})
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))
	pipe.Expire(ctx, key, time.Minute*2)
	if _, err = pipe.Exec(ctx); err != nil {
		return nil
	}

	remaining := max(rule.RequestsPerMinute-int(count)-1, 0)
	ctx.Response.Header.Set("X-RateLimit-Limit", strconv.Itoa(rule.RequestsPerMinute))
	ctx.Response.Header.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	ctx.Response.Header.Set("X-RateLimit-Reset", strconv.FormatInt(now+60, 10))

	return nil
}
