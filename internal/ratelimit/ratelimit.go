package ratelimit

import (
	"fmt"
	"strconv"
	"time"

	realip "github.com/ferluci/fast-realip"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Config holds rate limiting configuration
type Config struct {
	Widget WidgetConfig `toml:"widget"`
}

// WidgetConfig holds widget-specific rate limiting configuration
type WidgetConfig struct {
	Enabled           bool `toml:"enabled"`
	RequestsPerMinute int  `toml:"requests_per_minute"`
}

// Limiter handles rate limiting using Redis
type Limiter struct {
	redis  *redis.Client
	config Config
}

// New creates a new rate limiter
func New(redisClient *redis.Client, config Config) *Limiter {
	return &Limiter{
		redis:  redisClient,
		config: config,
	}
}

// CheckWidgetLimit checks if the widget request should be rate limited
func (l *Limiter) CheckWidgetLimit(ctx *fasthttp.RequestCtx) error {
	if !l.config.Widget.Enabled {
		return nil
	}

	clientIP := realip.FromRequest(ctx)
	key := fmt.Sprintf("rate_limit:widget:%s", clientIP)

	// Use sliding window approach with Redis
	now := time.Now().Unix()
	windowStart := now - 60 // 60 seconds window

	// Get current count in the last minute
	count, err := l.redis.ZCount(ctx, key, strconv.FormatInt(windowStart, 10), "+inf").Result()
	if err != nil {
		// Redis is down, allow request
		return nil
	}

	if count >= int64(l.config.Widget.RequestsPerMinute) {
		// Set rate limit headers
		ctx.Response.Header.Set("X-RateLimit-Limit", strconv.Itoa(l.config.Widget.RequestsPerMinute))
		ctx.Response.Header.Set("X-RateLimit-Remaining", "0")
		ctx.Response.Header.Set("X-RateLimit-Reset", strconv.FormatInt(now+60, 10))
		ctx.Response.Header.Set("Retry-After", "60")

		ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
		ctx.SetBodyString(`{"status":"error","message":"Rate limit exceeded"}`)
		return fmt.Errorf("rate limit exceeded")
	}

	// Add current request to the sliding window
	// Use nanoseconds as member to ensure uniqueness for multiple requests in same second
	pipe := l.redis.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: time.Now().UnixNano()})
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))
	pipe.Expire(ctx, key, time.Minute*2) // Set expiry to cleanup old keys
	_, err = pipe.Exec(ctx)
	if err != nil {
		// Redis is down, allow request
		return nil
	}

	// Set rate limit headers for successful requests
	remaining := max(l.config.Widget.RequestsPerMinute-int(count)-1, 0)
	ctx.Response.Header.Set("X-RateLimit-Limit", strconv.Itoa(l.config.Widget.RequestsPerMinute))
	ctx.Response.Header.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	ctx.Response.Header.Set("X-RateLimit-Reset", strconv.FormatInt(now+60, 10))

	return nil
}

// WidgetMiddleware returns a fastglue middleware for widget rate limiting
func (l *Limiter) WidgetMiddleware() func(*fastglue.Request) error {
	return func(r *fastglue.Request) error {
		return l.CheckWidgetLimit(r.RequestCtx)
	}
}
