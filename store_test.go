package main

import (
	"sync"
	"testing"
)

type fakeClock struct{ now float64 }

func (f *fakeClock) Now() float64 { return f.now }

func newTestStore(capacity, refill, ttl float64) (*Store, *fakeClock) {
	fc := &fakeClock{}
	return NewStore(capacity, refill, ttl, fc.Now), fc
}

func TestPerKeyIsolation(t *testing.T) {
	s, _ := newTestStore(1, 1, 100)
	if ok, _, _ := s.Check("a"); !ok {
		t.Fatal("a first check should be allowed")
	}
	if ok, _, _ := s.Check("a"); ok {
		t.Fatal("a second check should be rejected")
	}
	if ok, _, _ := s.Check("b"); !ok {
		t.Fatal("b should be unaffected by a")
	}
}

func TestEvictsIdleKeys(t *testing.T) {
	s, fc := newTestStore(2, 0.001, 10)
	s.Check("idle")
	fc.now = 11
	s.Check("other") // triggers the sweep (11 >= ttl/2)
	if _, ok := s.buckets["idle"]; ok {
		t.Fatal("idle bucket should be evicted after ttl")
	}
}

func TestDoesNotEvictActiveKeys(t *testing.T) {
	s, fc := newTestStore(100, 0.001, 10)
	for i := 1; i <= 30; i++ {
		fc.now = float64(i)
		s.Check("active")
	}
	b, ok := s.buckets["active"]
	if !ok {
		t.Fatal("active bucket was evicted")
	}
	if got := b.Remaining(fc.now); got != 70 {
		t.Fatalf("remaining = %d, want 70 (state must survive sweeps)", got)
	}
}

func TestSweepIsThrottled(t *testing.T) {
	s, fc := newTestStore(2, 0.001, 10)
	s.Check("a")
	fc.now = 4 // below ttl/2: no sweep yet
	s.Check("b")
	fc.now = 12
	s.Check("c") // sweep: a idle 12 > 10 is evicted, b idle 8 stays
	if _, ok := s.buckets["a"]; ok {
		t.Fatal("a should be evicted")
	}
	if _, ok := s.buckets["b"]; !ok {
		t.Fatal("b should remain")
	}
}

func TestPeekDoesNotConsumeOrCreate(t *testing.T) {
	s, _ := newTestStore(3, 1, 100)
	if got := s.Peek("new"); got != 3 {
		t.Fatalf("Peek unknown = %d, want 3", got)
	}
	if len(s.buckets) != 0 {
		t.Fatal("Peek created a bucket")
	}
	s.Check("k")
	for i := 0; i < 5; i++ {
		if got := s.Peek("k"); got != 2 {
			t.Fatalf("Peek = %d, want 2", got)
		}
	}
}

func TestConcurrentCheckNeverExceedsCapacity(t *testing.T) {
	const capacity = 50
	s, _ := newTestStore(capacity, 1, 100)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _, _ := s.Check("k"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != capacity {
		t.Fatalf("allowed = %d, want %d", allowed, capacity)
	}
}
