package main

import "testing"

func TestBurstThenReject(t *testing.T) {
	b := NewTokenBucket(3, 1, 0)
	for i := 0; i < 3; i++ {
		if !b.Take(0) {
			t.Fatalf("take %d should be allowed", i)
		}
	}
	if b.Take(0) {
		t.Fatal("fourth take should be rejected")
	}
}

func TestRefillOverTime(t *testing.T) {
	b := NewTokenBucket(2, 1, 0)
	b.Take(0)
	b.Take(0)
	if b.Take(0.5) {
		t.Fatal("0.5 tokens should not allow a take")
	}
	if !b.Take(1) {
		t.Fatal("one token should be available after 1s")
	}
}

func TestNeverExceedsCapacity(t *testing.T) {
	b := NewTokenBucket(2, 10, 0)
	b.Take(0)
	if got := b.Remaining(1000); got != 2 {
		t.Fatalf("Remaining = %d, want 2", got)
	}
}

func TestClockGoingBackwards(t *testing.T) {
	b := NewTokenBucket(2, 1, 10)
	b.Take(10)
	b.Take(10)
	if b.Take(5) {
		t.Fatal("backwards clock must not add tokens")
	}
	// last must not have moved back: only 1s has elapsed since t=10.
	if !b.Take(11) {
		t.Fatal("token expected 1s after the original last time")
	}
	if b.Take(11) {
		t.Fatal("backwards clock must not have granted extra tokens")
	}
}

func TestRetryAfterRoundsUp(t *testing.T) {
	b := NewTokenBucket(1, 0.4, 0)
	b.Take(0)
	if got := b.RetryAfter(0); got != 3 { // 1/0.4 = 2.5s
		t.Fatalf("RetryAfter = %d, want 3", got)
	}
	if got := b.RetryAfter(1); got != 2 { // 0.6 left / 0.4 = 1.5s
		t.Fatalf("RetryAfter = %d, want 2", got)
	}
}

func TestRetryAfterZeroWhenAvailable(t *testing.T) {
	b := NewTokenBucket(2, 1, 0)
	if got := b.RetryAfter(0); got != 0 {
		t.Fatalf("RetryAfter = %d, want 0", got)
	}
}
