package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginEndpointRateLimited(t *testing.T) {
	r := setupRouter(t)

	// 登录接口限流阈值为每分钟 10 次/IP（见 routers/router.go）。
	var lastStatus int
	for i := 0; i < 10; i++ {
		w := postJSON(t, r, "/api/v1/users/login", map[string]string{
			"phone": "13800009001", "password": "whatever",
		})
		lastStatus = w.Code
	}
	// 前 10 次即使凭据错误，也只是 401，不会被限流。
	if lastStatus != http.StatusUnauthorized {
		t.Fatalf("first 10 requests should reach handler (401), got %d", lastStatus)
	}

	// 第 11 次应在中间件被拦截，直接 429，且带 Retry-After 头。
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit exceeded, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("429 response should include Retry-After header")
	}
}

func TestNonCredentialEndpointNotRateLimited(t *testing.T) {
	r := setupRouter(t)

	// /ping 未挂限流，连续 15 次都应正常响应。
	for i := 0; i < 15; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("ping should never be rate limited, got %d at attempt %d", w.Code, i)
		}
	}
}
