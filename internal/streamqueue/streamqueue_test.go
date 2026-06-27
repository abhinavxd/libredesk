package streamqueue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

func testQueue(t *testing.T, mr *miniredis.Miniredis, opts Opts) *Queue {
	t.Helper()
	opts.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	opts.Logger = ptrLogger()
	if opts.Stream == "" {
		opts.Stream = "test:stream"
	}
	if opts.Group == "" {
		opts.Group = "test:group"
	}
	q, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return q
}

func ptrLogger() *logf.Logger {
	l := logf.New(logf.Opts{Level: logf.FatalLevel})
	return &l
}

func readOne(t *testing.T, q *Queue, consumer string) redis.XMessage {
	t.Helper()
	res, err := q.rd.XReadGroup(context.Background(), &redis.XReadGroupArgs{
		Group: q.group, Consumer: consumer, Streams: []string{q.stream, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("XReadGroup: %v", err)
	}
	if len(res) == 0 || len(res[0].Messages) == 0 {
		t.Fatalf("expected one message, got none")
	}
	return res[0].Messages[0]
}

func pendingCount(t *testing.T, q *Queue) int64 {
	t.Helper()
	p, err := q.rd.XPending(context.Background(), q.stream, q.group).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	return p.Count
}

func TestQueueProcessesAndAcks(t *testing.T) {
	mr := miniredis.RunT(t)
	var got int64
	done := make(chan struct{}, 3)
	q := testQueue(t, mr, Opts{
		Workers: 2,
		Handler: func(_ context.Context, _ []byte) error {
			atomic.AddInt64(&got, 1)
			done <- struct{}{}
			return nil
		},
	})

	for range 3 {
		if err := q.Enqueue(context.Background(), []byte("x")); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}

	go q.Run()
	defer q.Close()

	for range 3 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out, processed %d/3", atomic.LoadInt64(&got))
		}
	}
	if n := atomic.LoadInt64(&got); n != 3 {
		t.Fatalf("processed %d, want 3", n)
	}
	waitFor(t, func() bool { return pendingCount(t, q) == 0 }, "pending should drain to 0")
}

func TestQueueRetriesPendingUntilSuccess(t *testing.T) {
	mr := miniredis.RunT(t)
	var attempts int64
	q := testQueue(t, mr, Opts{
		Workers:      1,
		ClaimMinIdle: 10 * time.Millisecond,
		Handler: func(_ context.Context, _ []byte) error {
			if atomic.AddInt64(&attempts, 1) < 3 {
				return errors.New("transient failure")
			}
			return nil
		},
	})

	if err := q.Enqueue(context.Background(), []byte("x")); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	msg := readOne(t, q, "w0")
	q.process("w0", msg)
	if pendingCount(t, q) != 1 {
		t.Fatalf("entry should stay pending after a failed handler")
	}

	for range 2 {
		time.Sleep(25 * time.Millisecond)
		q.reclaimOnce()
	}

	if n := atomic.LoadInt64(&attempts); n != 3 {
		t.Fatalf("attempts = %d, want 3 (1 initial + 2 reclaim)", n)
	}
	if c := pendingCount(t, q); c != 0 {
		t.Fatalf("pending = %d, want 0 after success", c)
	}
}

func TestQueueDeadLettersAfterMaxAttempts(t *testing.T) {
	mr := miniredis.RunT(t)
	var attempts int64
	q := testQueue(t, mr, Opts{
		Workers:      1,
		MaxAttempts:  1,
		ClaimMinIdle: 10 * time.Millisecond,
		Handler: func(_ context.Context, _ []byte) error {
			atomic.AddInt64(&attempts, 1)
			return errors.New("always fails")
		},
	})

	if err := q.Enqueue(context.Background(), []byte("poison")); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	msg := readOne(t, q, "w0")
	q.process("w0", msg)

	for range 4 {
		time.Sleep(25 * time.Millisecond)
		q.reclaimOnce()
		if pendingCount(t, q) == 0 {
			break
		}
	}

	if c := pendingCount(t, q); c != 0 {
		t.Fatalf("pending = %d, want 0 after dead-letter", c)
	}
	if n := atomic.LoadInt64(&attempts); n != 1 {
		t.Fatalf("handler ran %d times, want 1 (MaxAttempts=1 must be honored exactly)", n)
	}
	deadLen, err := q.rd.XLen(context.Background(), q.deadStream).Result()
	if err != nil {
		t.Fatalf("XLen dead: %v", err)
	}
	if deadLen != 1 {
		t.Fatalf("dead-letter stream length = %d, want 1", deadLen)
	}
}

func TestQueueSurvivesRestart(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	stream, group := "test:restart", "g"

	producer, err := New(Opts{
		Redis: redis.NewClient(&redis.Options{Addr: addr}), Logger: ptrLogger(),
		Stream: stream, Group: group, Handler: func(context.Context, []byte) error { return nil },
	})
	if err != nil {
		t.Fatalf("New producer: %v", err)
	}
	if err := producer.Enqueue(context.Background(), []byte("queued-before-start")); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var got []byte
	var mu sync.Mutex
	processed := make(chan struct{}, 1)
	consumer := testQueue(t, mr, Opts{
		Stream: stream, Group: group, Workers: 1,
		Handler: func(_ context.Context, payload []byte) error {
			mu.Lock()
			got = append([]byte(nil), payload...)
			mu.Unlock()
			processed <- struct{}{}
			return nil
		},
	})
	go consumer.Run()
	defer consumer.Close()

	select {
	case <-processed:
	case <-time.After(5 * time.Second):
		t.Fatal("entry enqueued before the consumer started was never processed")
	}
	mu.Lock()
	defer mu.Unlock()
	if string(got) != "queued-before-start" {
		t.Fatalf("got %q, want %q", got, "queued-before-start")
	}
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met: %s", msg)
}
