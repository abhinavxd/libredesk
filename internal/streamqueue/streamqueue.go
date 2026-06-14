// Package streamqueue is a durable, at-least-once work queue backed by Redis Streams; handlers MUST be idempotent.
package streamqueue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

const (
	defaultWorkers      = 4
	defaultMaxAttempts  = 50
	defaultMaxLen       = 100000
	defaultClaimMinIdle = 30 * time.Second
	defaultReclaimEvery = 15 * time.Second
	defaultBlock        = 5 * time.Second
	defaultBatch        = 16
	defaultAckTimeout   = 5 * time.Second

	payloadField = "payload"
	origIDField  = "orig_id"
)

// Handler processes one entry; a non-nil error leaves the entry pending for retry, nil acknowledges it.
type Handler func(ctx context.Context, payload []byte) error

// Opts configures a Queue. Redis, Stream, Group, Handler and Logger are required.
type Opts struct {
	Redis        *redis.Client
	Logger       *logf.Logger
	Handler      Handler
	Stream       string
	Group        string
	Consumer     string
	Workers      int
	MaxAttempts  int
	MaxLen       int64
	ClaimMinIdle time.Duration
	ReclaimEvery time.Duration
}

// Queue is a single consumer group over one Redis stream, with a dead-letter stream for poison entries.
type Queue struct {
	rd           *redis.Client
	lo           *logf.Logger
	handler      Handler
	stream       string
	deadStream   string
	group        string
	consumer     string
	workers      int
	maxAttempts  int
	maxLen       int64
	claimMinIdle time.Duration
	reclaimEvery time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// New creates the consumer group (and stream) if absent and returns a ready Queue. Call Run to start consuming.
func New(opts Opts) (*Queue, error) {
	if opts.Redis == nil {
		return nil, errors.New("streamqueue: redis client is required")
	}
	if opts.Stream == "" || opts.Group == "" {
		return nil, errors.New("streamqueue: stream and group are required")
	}
	if opts.Handler == nil {
		return nil, errors.New("streamqueue: handler is required")
	}
	if opts.Logger == nil {
		return nil, errors.New("streamqueue: logger is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		rd:           opts.Redis,
		lo:           opts.Logger,
		handler:      opts.Handler,
		stream:       opts.Stream,
		deadStream:   opts.Stream + ":dead",
		group:        opts.Group,
		consumer:     cmpOr(opts.Consumer, "consumer"),
		workers:      positiveOr(opts.Workers, defaultWorkers),
		maxAttempts:  positiveOr(opts.MaxAttempts, defaultMaxAttempts),
		maxLen:       positiveOr64(opts.MaxLen, defaultMaxLen),
		claimMinIdle: durationOr(opts.ClaimMinIdle, defaultClaimMinIdle),
		reclaimEvery: durationOr(opts.ReclaimEvery, defaultReclaimEvery),
		ctx:          ctx,
		cancel:       cancel,
	}

	if err := q.rd.XGroupCreateMkStream(ctx, q.stream, q.group, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		cancel()
		return nil, fmt.Errorf("streamqueue: creating consumer group: %w", err)
	}
	return q, nil
}

// Enqueue appends a payload to the stream. A nil error means the entry is durably stored.
func (q *Queue) Enqueue(ctx context.Context, payload []byte) error {
	return q.rd.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		MaxLen: q.maxLen,
		Approx: true,
		Values: map[string]any{payloadField: payload},
	}).Err()
}

// Run starts the workers and the reclaimer and blocks until Close is called.
func (q *Queue) Run() {
	for i := range q.workers {
		q.wg.Add(1)
		go q.worker(fmt.Sprintf("%s:%d", q.consumer, i))
	}
	q.wg.Add(1)
	go q.reclaimer()
	q.wg.Wait()
}

// Close stops consuming and waits for in-flight handlers to finish; un-acked entries stay durable for the next start.
func (q *Queue) Close() {
	q.cancel()
	q.wg.Wait()
}

func (q *Queue) worker(consumer string) {
	defer q.wg.Done()
	for {
		if q.ctx.Err() != nil {
			return
		}
		res, err := q.rd.XReadGroup(q.ctx, &redis.XReadGroupArgs{
			Group:    q.group,
			Consumer: consumer,
			Streams:  []string{q.stream, ">"},
			Count:    defaultBatch,
			Block:    defaultBlock,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || q.ctx.Err() != nil {
				continue
			}
			q.lo.Error("error reading from stream", "stream", q.stream, "error", err)
			q.sleep(time.Second)
			continue
		}
		for _, s := range res {
			for _, msg := range s.Messages {
				if q.ctx.Err() != nil {
					return
				}
				q.process(consumer, msg)
			}
		}
	}
}

func (q *Queue) reclaimer() {
	defer q.wg.Done()
	t := time.NewTicker(q.reclaimEvery)
	defer t.Stop()
	for {
		select {
		case <-q.ctx.Done():
			return
		case <-t.C:
			q.reclaimOnce()
		}
	}
}

func (q *Queue) reclaimOnce() {
	pending, err := q.rd.XPendingExt(q.ctx, &redis.XPendingExtArgs{
		Stream: q.stream,
		Group:  q.group,
		Idle:   q.claimMinIdle,
		Start:  "-",
		End:    "+",
		Count:  defaultBatch,
	}).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) && q.ctx.Err() == nil {
			q.lo.Error("error scanning pending stream entries", "stream", q.stream, "error", err)
		}
		return
	}

	var retry, dead []string
	for _, p := range pending {
		if int(p.RetryCount) >= q.maxAttempts {
			dead = append(dead, p.ID)
		} else {
			retry = append(retry, p.ID)
		}
	}

	if len(dead) > 0 {
		q.deadLetter(dead)
	}
	if len(retry) == 0 {
		return
	}

	msgs, err := q.rd.XClaim(q.ctx, &redis.XClaimArgs{
		Stream:   q.stream,
		Group:    q.group,
		Consumer: q.consumer + ":reclaim",
		MinIdle:  q.claimMinIdle,
		Messages: retry,
	}).Result()
	if err != nil {
		if q.ctx.Err() == nil {
			q.lo.Error("error claiming pending stream entries", "stream", q.stream, "error", err)
		}
		return
	}
	for _, msg := range msgs {
		if q.ctx.Err() != nil {
			return
		}
		q.process(q.consumer+":reclaim", msg)
	}
}

func (q *Queue) deadLetter(ids []string) {
	msgs, err := q.rd.XClaim(q.ctx, &redis.XClaimArgs{
		Stream:   q.stream,
		Group:    q.group,
		Consumer: q.consumer + ":dead",
		MinIdle:  q.claimMinIdle,
		Messages: ids,
	}).Result()
	if err != nil {
		if q.ctx.Err() == nil {
			q.lo.Error("error claiming dead stream entries", "stream", q.stream, "error", err)
		}
		return
	}
	for _, msg := range msgs {
		if err := q.rd.XAdd(q.ctx, &redis.XAddArgs{
			Stream: q.deadStream,
			Values: map[string]any{payloadField: msg.Values[payloadField], origIDField: msg.ID},
		}).Err(); err != nil {
			q.lo.Error("error moving entry to dead-letter stream", "stream", q.stream, "dead_stream", q.deadStream, "id", msg.ID, "error", err)
			continue
		}
		if err := q.rd.XAck(q.ctx, q.stream, q.group, msg.ID).Err(); err == nil {
			q.rd.XDel(q.ctx, q.stream, msg.ID)
		}
		q.lo.Error("stream entry dead-lettered after exceeding max attempts", "stream", q.stream, "dead_stream", q.deadStream, "id", msg.ID, "max_attempts", q.maxAttempts)
	}
}

func (q *Queue) process(consumer string, msg redis.XMessage) {
	payload, _ := msg.Values[payloadField].(string)
	if err := q.invoke([]byte(payload)); err != nil {
		q.lo.Warn("stream handler failed, entry left pending for retry", "stream", q.stream, "consumer", consumer, "id", msg.ID, "error", err)
		return
	}
	// Ack on a detached context so a handler that finished as shutdown cancelled q.ctx still releases the entry.
	ackCtx, cancel := context.WithTimeout(context.Background(), defaultAckTimeout)
	defer cancel()
	if err := q.rd.XAck(ackCtx, q.stream, q.group, msg.ID).Err(); err != nil {
		q.lo.Error("error acking stream entry", "stream", q.stream, "id", msg.ID, "error", err)
		return
	}
	q.rd.XDel(ackCtx, q.stream, msg.ID)
}

func (q *Queue) invoke(payload []byte) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("streamqueue: handler panic: %v", rec)
		}
	}()
	return q.handler(q.ctx, payload)
}

func (q *Queue) sleep(d time.Duration) {
	select {
	case <-q.ctx.Done():
	case <-time.After(d):
	}
}

func cmpOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func positiveOr(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func positiveOr64(v, fallback int64) int64 {
	if v <= 0 {
		return fallback
	}
	return v
}

func durationOr(v, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	return v
}
