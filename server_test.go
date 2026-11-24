package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func newTestServer(capacity, refill float64) (http.Handler, *fakeClock) {
	fc := &fakeClock{}
	store := NewStore(capacity, refill, 300, fc.Now)
	return NewServer(Settings{Capacity: capacity, RefillPerSec: refill, IdleTTLSec: 300}, store), fc
}

func do(h http.Handler, path, remote string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = remote
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAllowThenRateLimited(t *testing.T) {
	h, _ := newTestServer(2, 1)
	for i := 0; i < 2; i++ {
		rec := do(h, "/resource", "1.2.3.4:5000", nil)
		if rec.Code != 200 || rec.Body.String() != "{\"ok\":true}\n" {
			t.Fatalf("request %d: %d %q", i, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("X-RateLimit-Limit"); got != "2" {
			t.Fatalf("Limit = %q", got)
		}
		if got := rec.Header().Get("X-RateLimit-Remaining"); got != strconv.Itoa(1-i) {
			t.Fatalf("Remaining = %q", got)
		}
		if got := rec.Header().Get("X-RateLimit-Reset"); got != strconv.Itoa(i+1) {
			t.Fatalf("Reset = %q", got)
		}
		if rec.Header().Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON content type")
		}
	}
	rec := do(h, "/resource", "1.2.3.4:5000", nil)
	if rec.Code != 429 || rec.Body.String() != "{\"error\":\"rate limited\"}\n" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "0" || rec.Header().Get("X-RateLimit-Limit") != "2" {
		t.Fatal("rate limit headers missing on 429")
	}
	if rec.Header().Get("X-RateLimit-Reset") != "2" {
		t.Fatalf("Reset = %q", rec.Header().Get("X-RateLimit-Reset"))
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatal("missing JSON content type on 429")
	}
	if ra, err := strconv.Atoi(rec.Header().Get("Retry-After")); err != nil || ra < 1 {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
}

func TestRetryAfterAtLeastOneOnFractionalWait(t *testing.T) {
	h, fc := newTestServer(1, 1)
	do(h, "/resource", "1.2.3.4:1", nil)
	fc.now = 0.999
	rec := do(h, "/resource", "1.2.3.4:1", nil)
	if rec.Code != 429 || rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("got %d Retry-After=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
}

func TestStatusDoesNotConsume(t *testing.T) {
	h, _ := newTestServer(1, 1)
	for i := 0; i < 3; i++ {
		rec := do(h, "/status", "1.2.3.4:1", nil)
		if rec.Code != 200 || rec.Body.String() != "{\"limit\":1,\"remaining\":1}\n" {
			t.Fatalf("status: %d %q", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON content type")
		}
	}
	if rec := do(h, "/resource", "1.2.3.4:1", nil); rec.Code != 200 {
		t.Fatalf("token was consumed by /status: %d", rec.Code)
	}
}

func TestSeparateLimitsPerRemoteAddr(t *testing.T) {
	h, _ := newTestServer(1, 1)
	if do(h, "/resource", "1.1.1.1:1", nil).Code != 200 {
		t.Fatal("first client should be allowed")
	}
	if do(h, "/resource", "1.1.1.1:2", nil).Code != 429 {
		t.Fatal("same host, different port must share a limit")
	}
	if do(h, "/resource", "2.2.2.2:1", nil).Code != 200 {
		t.Fatal("second client should have its own limit")
	}
}

func TestForwardedForIgnored(t *testing.T) {
	h, _ := newTestServer(1, 1)
	do(h, "/resource", "1.1.1.1:1", map[string]string{"X-Forwarded-For": "9.9.9.1"})
	rec := do(h, "/resource", "1.1.1.1:1", map[string]string{"X-Forwarded-For": "9.9.9.2"})
	if rec.Code != 429 {
		t.Fatalf("spoofed X-Forwarded-For bypassed the limit: %d", rec.Code)
	}
}

func TestClientKeyFallsBackToRawAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-a-hostport"
	if got := clientKey(req); got != "not-a-hostport" {
		t.Fatalf("clientKey = %q", got)
	}
}
