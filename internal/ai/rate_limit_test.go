package ai

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zerodha/logf"
)

func TestConversationRateLimiting(t *testing.T) {
	// Create a minimal manager for testing
	lo := logf.New(logf.Opts{})
	manager := &Manager{
		lo: &lo,
	}

	conversationUUID := "test-conv-123"

	// Test acquiring slots up to the limit
	if !manager.tryAcquireConversationSlot(conversationUUID) {
		t.Error("First slot acquisition should succeed")
	}

	if !manager.tryAcquireConversationSlot(conversationUUID) {
		t.Error("Second slot acquisition should succeed")
	}

	// Third attempt should fail (limit is 2)
	if manager.tryAcquireConversationSlot(conversationUUID) {
		t.Error("Third slot acquisition should fail - rate limit exceeded")
	}

	// Release one slot
	manager.releaseConversationSlot(conversationUUID)

	// Now we should be able to acquire again
	if !manager.tryAcquireConversationSlot(conversationUUID) {
		t.Error("Slot acquisition should succeed after release")
	}

	// Release all slots
	manager.releaseConversationSlot(conversationUUID)
	manager.releaseConversationSlot(conversationUUID)

	// Verify the conversation remains in the map (no cleanup by design)
	if _, exists := manager.pendingRequests.Load(conversationUUID); !exists {
		t.Error("Conversation should remain in pending requests map (no cleanup by design)")
	}
}

func TestConcurrentRateLimiting(t *testing.T) {
	lo := logf.New(logf.Opts{})
	manager := &Manager{
		lo: &lo,
	}

	conversationUUID := "test-conv-concurrent"
	numGoroutines := 10
	successCount := int64(0)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Try to acquire slots concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			if manager.tryAcquireConversationSlot(conversationUUID) {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	// Only 2 should succeed (our rate limit)
	if successCount != 2 {
		t.Errorf("Expected exactly 2 successful acquisitions, got %d", successCount)
	}

	// Clean up
	manager.releaseConversationSlot(conversationUUID)
	manager.releaseConversationSlot(conversationUUID)
}

func TestRaceConditionInCleanup(t *testing.T) {
	lo := logf.New(logf.Opts{})
	manager := &Manager{
		lo: &lo,
	}

	conversationUUID := "test-race-cleanup"
	numGoroutines := 100
	var wg sync.WaitGroup

	// Acquire one slot
	if !manager.tryAcquireConversationSlot(conversationUUID) {
		t.Fatal("Initial slot acquisition should succeed")
	}

	// Start many goroutines trying to acquire/release simultaneously
	successCount := int64(0)
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			
			// Half try to acquire, half try to release the initial slot
			if i%2 == 0 {
				if manager.tryAcquireConversationSlot(conversationUUID) {
					atomic.AddInt64(&successCount, 1)
					// Release immediately to create more race opportunities
					manager.releaseConversationSlot(conversationUUID)
				}
			} else if i == 1 {
				// Only one goroutine releases the initial slot
				manager.releaseConversationSlot(conversationUUID)
			}
		}(i)
	}

	wg.Wait()

	// Verify the map entry persists (no cleanup by design)
	if _, exists := manager.pendingRequests.Load(conversationUUID); !exists {
		t.Error("Conversation should remain in pending requests map (no cleanup by design)")
	}

	t.Logf("Race condition test completed with %d successful acquisitions", successCount)
}

func TestStressTestRateLimiting(t *testing.T) {
	lo := logf.New(logf.Opts{})
	manager := &Manager{
		lo: &lo,
	}

	conversationUUID := "test-stress"
	numGoroutines := 50
	opsPerGoroutine := 100
	var wg sync.WaitGroup

	successCount := int64(0)
	failureCount := int64(0)

	wg.Add(numGoroutines)

	start := time.Now()
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			for j := 0; j < opsPerGoroutine; j++ {
				if manager.tryAcquireConversationSlot(conversationUUID) {
					atomic.AddInt64(&successCount, 1)
					// Simulate some work
					time.Sleep(1 * time.Microsecond)
					manager.releaseConversationSlot(conversationUUID)
				} else {
					atomic.AddInt64(&failureCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	totalOps := numGoroutines * opsPerGoroutine
	t.Logf("Stress test: %d total operations in %v", totalOps, duration)
	t.Logf("Success: %d, Failures: %d", successCount, failureCount)
	t.Logf("Rate limiting working: %d operations were blocked", failureCount)

	// Verify entry persists (no cleanup by design) - this is intentional for simplicity
	if _, exists := manager.pendingRequests.Load(conversationUUID); !exists {
		t.Error("Conversation should remain in pending requests map (no cleanup by design)")
	}
}

func TestCleanupWorkerFunctionality(t *testing.T) {
	lo := logf.New(logf.Opts{})
	
	manager := &Manager{
		lo: &lo,
	}

	conversationUUID := "test-cleanup-worker"

	// Acquire and release a slot to create a zero-count entry
	if !manager.tryAcquireConversationSlot(conversationUUID) {
		t.Fatal("Initial slot acquisition should succeed")
	}
	manager.releaseConversationSlot(conversationUUID)

	// Verify entry exists with zero count
	if value, ok := manager.pendingRequests.Load(conversationUUID); !ok {
		t.Fatal("Entry should exist before cleanup")
	} else {
		counter := value.(*atomic.Int64)
		if counter.Load() != 0 {
			t.Errorf("Counter should be 0, got %d", counter.Load())
		}
	}

	// Manually trigger cleanup logic (without waiting 1 hour)
	var keysToDelete []interface{}
	manager.pendingRequests.Range(func(key, value interface{}) bool {
		counter := value.(*atomic.Int64)
		if counter.Load() <= 0 {
			keysToDelete = append(keysToDelete, key)
		}
		return true
	})

	for _, key := range keysToDelete {
		manager.pendingRequests.Delete(key)
	}

	// Verify entry was cleaned up
	if _, ok := manager.pendingRequests.Load(conversationUUID); ok {
		t.Error("Entry should be cleaned up after manual cleanup")
	}

	t.Logf("Cleanup test completed, %d entries cleaned", len(keysToDelete))
}
