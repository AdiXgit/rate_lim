package main

import (
	"sync"
	"time"
)

// Clock returns the current time in seconds from an arbitrary fixed origin.
type Clock func() float64

// MonotonicClock returns a Clock measuring seconds since this call.
func MonotonicClock() Clock {
	start := time.Now()
	return func() float64 { return time.Since(start).Seconds() }
}

// Store keeps one TokenBucket per key and evicts idle ones.
type Store struct {
	mu        sync.Mutex
	buckets   map[string]*TokenBucket
	capacity  float64
	refill    float64
	ttl       float64
	clock     Clock
	lastSweep float64
}

// NewStore returns an empty Store.
func NewStore(capacity, refill, ttl float64, clock Clock) *Store {
	return &Store{
		buckets:   make(map[string]*TokenBucket),
		capacity:  capacity,
		refill:    refill,
		ttl:       ttl,
		clock:     clock,
		lastSweep: clock(),
	}
}

// Check consumes one token for key and reports whether the request is allowed,
// the tokens remaining, and the seconds to wait when it is not.
func (s *Store) Check(key string) (allowed bool, remaining int, retryAfter int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock()
	s.sweep(now)
	b, ok := s.buckets[key]
	if !ok {
		b = NewTokenBucket(s.capacity, s.refill, now)
		s.buckets[key] = b
	}
	allowed = b.Take(now)
	return allowed, b.Remaining(now), b.RetryAfter(now)
}

// Peek returns the tokens remaining for key without consuming one or creating
// a bucket for an unknown key.
func (s *Store) Peek(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[key]
	if !ok {
		return int(s.capacity)
	}
	return b.Remaining(s.clock())
}

// sweep evicts buckets idle longer than ttl, at most once per ttl/2.
// The caller must hold s.mu.
func (s *Store) sweep(now float64) {
	if now-s.lastSweep < s.ttl/2 {
		return
	}
	s.lastSweep = now
	for k, b := range s.buckets {
		if now-b.last > s.ttl {
			delete(s.buckets, k)
		}
	}
}
