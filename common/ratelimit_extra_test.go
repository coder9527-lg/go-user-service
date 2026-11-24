package common

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// M1 回归：过期 key 应被惰性删除，map 不会无限增长。
func TestIPRateLimiterExpiredKeyIsDeleted(t *testing.T) {
	base := time.Now()
	cur := base
	l := NewIPRateLimiter(1, time.Minute)
	l.now = func() time.Time { return cur }

	l.Allow("stale-ip")
	if l.Len() != 1 {
		t.Fatalf("expected 1 key, got %d", l.Len())
	}

	// 窗口过期后，同一 key 再次访问应复用而非累积新条目。
	cur = base.Add(2 * time.Minute)
	l.Allow("stale-ip")
	if l.Len() != 1 {
		t.Fatalf("expired entry should be replaced in place, len=%d", l.Len())
	}
}

// H3 回归：同一个 limiter 被多个路由复用时，配额是共享的。
func TestSharedLimiterQuotaIsShared(t *testing.T) {
	l := NewIPRateLimiter(2, time.Minute)
	mw := RateLimit(l)

	// 模拟多个路由共用同一 limiter，对同一 IP 计数应累加。
	gin.SetMode(gin.TestMode)
	run := func() bool {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.RemoteAddr = "10.0.0.9:54321"
		c.Request = req
		mw(c)
		return !c.IsAborted()
	}
	first, second := run(), run()
	if !first || !second {
		t.Fatal("first two requests should pass")
	}
	if run() {
		t.Fatal("third request across shared limiter should be blocked")
	}
}
