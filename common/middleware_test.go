package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.Use(RequestID())
	r.Use(Logger())

	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(ContextRequestID))
	})
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})
	return r
}

func TestRequestIDGeneratedAndEchoed(t *testing.T) {
	r := newTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	rid := w.Header().Get(RequestIDHeader)
	if rid == "" {
		t.Fatal("response should contain X-Request-ID header")
	}
	// 响应体回显的 requestID 应与响应头一致，证明已注入 context。
	if body := w.Body.String(); body != rid {
		t.Fatalf("context requestID %q != header %q", body, rid)
	}
	if len(rid) != 36 || rid[14] != '4' {
		t.Fatalf("not a UUID v4 style id: %q", rid)
	}
}

func TestRequestIDPropagatedFromHeader(t *testing.T) {
	r := newTestRouter()
	const upstream = "upstream-trace-id-12345"

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set(RequestIDHeader, upstream)
	r.ServeHTTP(w, req)

	if got := w.Header().Get(RequestIDHeader); got != upstream {
		t.Fatalf("expected upstream id propagated, got %q", got)
	}
}

func TestRecoveryCapturesPanic(t *testing.T) {
	r := newTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	// panic 被 Recovery 捕获，进程不退出且返回 500，而不是连接中断。
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", w.Code)
	}
	if w.Header().Get(RequestIDHeader) == "" {
		t.Fatal("error response should still carry request id for troubleshooting")
	}
}
