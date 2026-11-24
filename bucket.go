package main

import "math"

// TokenBucket is a token-bucket limiter. It is not safe for concurrent use;
// Store serializes access.
type TokenBucket struct {
	capacity   float64
	refillRate float64
	tokens     float64
	last       float64
}

// NewTokenBucket returns a full bucket whose clock starts at now.
func NewTokenBucket(capacity, refillRate, now float64) *TokenBucket {
	return &TokenBucket{capacity: capacity, refillRate: refillRate, tokens: capacity, last: now}
}

// refill adds tokens for the time elapsed since last. A backwards clock is ignored.
func (b *TokenBucket) refill(now float64) {
	if now > b.last {
		b.tokens = math.Min(b.capacity, b.tokens+(now-b.last)*b.refillRate)
		b.last = now
	}
}

// Take consumes one token if available and reports whether it did.
func (b *TokenBucket) Take(now float64) bool {
	b.refill(now)
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Remaining returns the whole tokens currently available.
func (b *TokenBucket) Remaining(now float64) int {
	b.refill(now)
	return int(b.tokens)
}

// RetryAfter returns the whole seconds, rounded up, until one token is available.
func (b *TokenBucket) RetryAfter(now float64) int {
	b.refill(now)
	if b.tokens >= 1 {
		return 0
	}
	return int(math.Ceil((1 - b.tokens) / b.refillRate))
}
