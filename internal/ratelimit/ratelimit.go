package ratelimit

import (
	"fmt"
	"strconv"
	"time"

	realip "github.com/ferluci/fast-realip"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/logf"
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
	lo    *logf.Logger
}

// New creates a new rate limiter.
func New(redisClient *redis.Client, lo *logf.Logger) *Limiter {
	return &Limiter{
		redis: redisClient,
		lo:    lo,
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

	now := time.Now()
	nowUnix := now.Unix()
	nowNano := now.UnixNano()
	windowStart := strconv.FormatInt(nowUnix-60, 10)

	// Single pipeline: cleanup, add, count, set expiry.
	pipe := l.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", windowStart)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(nowUnix), Member: nowNano})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, time.Minute*2)
	if _, err := pipe.Exec(ctx); err != nil {
		l.lo.Warn("rate limit check unavailable, allowing request", "rule", ruleName, "error", err)
		return nil
	}

	count := countCmd.Val()
	limit := int64(rule.RequestsPerMinute)

	ctx.Response.Header.Set("X-RateLimit-Limit", strconv.Itoa(rule.RequestsPerMinute))
	ctx.Response.Header.Set("X-RateLimit-Reset", strconv.FormatInt(nowUnix+60, 10))

	if count > limit {
		ctx.Response.Header.Set("X-RateLimit-Remaining", "0")
		ctx.Response.Header.Set("Retry-After", "60")
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
		ctx.SetBodyString(`{"status":"error","message":"Rate limit exceeded"}`)
		return fmt.Errorf("rate limit exceeded")
	}

	remaining := max(int(limit-count), 0)
	ctx.Response.Header.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	return nil
}
