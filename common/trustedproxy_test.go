package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// H2 回归：不信任任何代理时，伪造的 X-Forwarded-For 不应影响 ClientIP。
func TestClientIPIgnoresSpoofedForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 与 routers.configureTrustedProxies(nil) 等价：不信任任何代理。
	_ = r.SetTrustedProxies(nil)
	var gotIP string
	r.GET("/ip", func(c *gin.Context) {
		gotIP = c.ClientIP()
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "203.0.113.7:5000"
	// 攻击者伪造 XFF，试图每次换一个 IP 绕过限流。
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", w.Code)
	}
	if gotIP != "203.0.113.7" {
		t.Fatalf("ClientIP should fall back to real remote addr, got %q", gotIP)
	}
}
