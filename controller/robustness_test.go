package controller_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestRegisterRejectsTooLongPassword(t *testing.T) {
	r := setupRouter(t)

	// 73 字节密码：超过 bcrypt 72 字节上限，应在入口被 422 拦截而非 500。
	body := map[string]string{
		"name":     "gina",
		"phone":    "13800005555",
		"password": strings.Repeat("a", 73),
	}
	w := postJSON(t, r, "/api/v1/users/register", body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for overlong password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConcurrentRegistrationNo500(t *testing.T) {
	r := setupRouter(t)

	const n = 10
	var wg sync.WaitGroup
	results := make(chan int, n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // 同时发起，制造“先查不到、再插入”的竞态窗口
			w := postJSON(t, r, "/api/v1/users/register", map[string]string{
				"name": "race", "phone": "13800006666", "password": "secret123",
			})
			results <- w.Code
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	ok, conflict, other := 0, 0, 0
	for code := range results {
		switch code {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		default:
			other++
		}
	}
	if ok != 1 {
		t.Fatalf("expected exactly 1 successful registration, got %d", ok)
	}
	if conflict != n-1 {
		t.Fatalf("expected %d conflicts, got %d", n-1, conflict)
	}
	if other != 0 {
		t.Fatalf("no request should return 5xx/other codes, got %d", other)
	}
}

func TestReadyProbe(t *testing.T) {
	r := setupRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ready probe expected 200, got %d", w.Code)
	}
}
