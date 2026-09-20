package main

import (
	"errors"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/resourceusage"
)

func TestLiveResourceLimitsConcurrentReadAndSave(t *testing.T) {
	var state resourceLimitState
	initial := resourceusage.Limits{MaxIncomingMessageSize: 100, MaxStorageBytes: 100}
	state.current.Store(&initial)
	var persisted, applied resourceusage.Limits
	var wg sync.WaitGroup
	for i := 1; i <= 30; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			limits := resourceusage.Limits{MaxIncomingMessageSize: int64(n), MaxStorageBytes: int64(n)}
			if err := state.update(limits, func(v resourceusage.Limits) error { persisted = v; return nil }, func(v resourceusage.Limits) { applied = v }); err != nil {
				t.Error(err)
			}
		}(i)
		go func() {
			defer wg.Done()
			for range 100 {
				v := state.snapshot()
				if v.MaxIncomingMessageSize != v.MaxStorageBytes {
					t.Errorf("torn snapshot: %+v", v)
				}
			}
		}()
	}
	wg.Wait()
	if persisted != applied || applied != state.snapshot() {
		t.Fatalf("persisted=%+v applied=%+v live=%+v", persisted, applied, state.snapshot())
	}
	before := state.snapshot()
	if err := state.update(initial, func(resourceusage.Limits) error { return errors.New("database offline") }, func(resourceusage.Limits) { t.Error("applied failed save") }); err == nil {
		t.Fatal("expected error")
	}
	if state.snapshot() != before {
		t.Fatal("failed save changed live limits")
	}
}
