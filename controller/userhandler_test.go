// 使用外部测试包 controller_test，避免 controller 与 router 之间的测试期导入循环。
package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder9527-lg/go-user-service/db"
	"github.com/coder9527-lg/go-user-service/model"
	router "github.com/coder9527-lg/go-user-service/routers"
	"github.com/gin-gonic/gin"
)

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gormDB, err := db.InitSQLite(sharedMemoryDSN(t))
	if err != nil {
		t.Fatalf("init sqlite: %v", err)
	}
	if err := gormDB.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return router.New()
}

func postJSON(t *testing.T, r *gin.Engine, path string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestRegisterValidation(t *testing.T) {
	r := setupRouter(t)

	cases := []map[string]string{
		{"name": "a", "phone": "123", "password": "secret123"},   // 手机号格式错
		{"name": "a", "phone": "13800000002", "password": "123"}, // 密码太短
		{"name": "a", "phone": "13800000003", "password": ""},    // 缺密码
	}
	for _, body := range cases {
		w := postJSON(t, r, "/api/v1/users/register", body)
		if w.Code == http.StatusOK {
			t.Fatalf("expected 4xx for body %v, got 200: %s", body, w.Body.String())
		}
	}
}

func TestRegisterThenLogin(t *testing.T) {
	r := setupRouter(t)

	// 注册
	w := postJSON(t, r, "/api/v1/users/register", map[string]string{
		"name": "carol", "phone": "13800000004", "password": "secret123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", w.Code, w.Body.String())
	}
	var regResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if regResp.Code != 0 {
		t.Fatalf("unexpected register code: %+v", regResp)
	}

	// 重复注册应冲突
	wDup := postJSON(t, r, "/api/v1/users/register", map[string]string{
		"name": "carol2", "phone": "13800000004", "password": "secret123",
	})
	if wDup.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", wDup.Code)
	}

	// 密码错误
	wBad := postJSON(t, r, "/api/v1/users/login", map[string]string{
		"phone": "13800000004", "password": "wrong-pwd",
	})
	if wBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", wBad.Code)
	}

	// 正确密码登录，应拿到 access + refresh 双令牌
	wLogin := postJSON(t, r, "/api/v1/users/login", map[string]string{
		"phone": "13800000004", "password": "secret123",
	})
	if wLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", wLogin.Code, wLogin.Body.String())
	}
	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wLogin.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResp.Data.AccessToken == "" || loginResp.Data.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens")
	}
}

func TestPing(t *testing.T) {
	r := setupRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
