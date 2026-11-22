package common

import (
	"testing"
	"time"
)

func TestIPRateLimiterAllowThenBlock(t *testing.T) {
	now := time.Now()
	l := NewIPRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 1; i <= 3; i++ {
		if ok, _ := l.Allow("1.1.1.1"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	ok, retryAfter := l.Allow("1.1.1.1")
	if ok {
		t.Fatal("4th request within window should be blocked")
	}
	if retryAfter <= 0 {
		t.Fatal("blocked request should report positive retry-after")
	}
}

func TestIPRateLimiterWindowReset(t *testing.T) {
	base := time.Now()
	cur := base
	l := NewIPRateLimiter(2, time.Minute)
	l.now = func() time.Time { return cur }

	l.Allow("k")
	l.Allow("k")
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("expected block within the same window")
	}

	// 时间推进超过窗口后，计数应清零、重新放行。
	cur = base.Add(2 * time.Minute)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("expected allow after the window has reset")
	}
}

func TestIPRateLimiterKeyIsolation(t *testing.T) {
	l := NewIPRateLimiter(1, time.Minute)

	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("first request of key a should be allowed")
	}
	// 不同 key 各自独立计数，互不影响。
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("different key should have an independent counter")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("second request of key a should be blocked")
	}
}
