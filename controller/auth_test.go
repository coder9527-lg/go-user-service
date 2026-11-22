package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// tokenPair 登录返回的双令牌。
type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// registerAndLogin 注册并登录，返回有效令牌对。
func registerAndLogin(t *testing.T, r *gin.Engine, phone string) tokenPair {
	t.Helper()
	w := postJSON(t, r, "/api/v1/users/register", map[string]string{
		"name": "dave", "phone": phone, "password": "secret123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", w.Code, w.Body.String())
	}
	wLogin := postJSON(t, r, "/api/v1/users/login", map[string]string{
		"phone": phone, "password": "secret123",
	})
	if wLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", wLogin.Code, wLogin.Body.String())
	}
	var resp struct {
		Data tokenPair `json:"data"`
	}
	if err := json.Unmarshal(wLogin.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode token pair: %v", err)
	}
	if resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens")
	}
	return resp.Data
}

func getMe(t *testing.T, r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMeRequiresAuth(t *testing.T) {
	r := setupRouter(t)

	// 1) 不带 Authorization 头 → 401
	if w := getMe(t, r, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no header: expected 401, got %d", w.Code)
	}

	// 2) 格式错误（缺少 Bearer 前缀）→ 401
	if w := getMe(t, r, "abc.def.ghi"); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad scheme: expected 401, got %d", w.Code)
	}

	// 3) 伪造 token → 401
	if w := getMe(t, r, "Bearer not-a-real-token"); w.Code != http.StatusUnauthorized {
		t.Fatalf("forged token: expected 401, got %d", w.Code)
	}
}

func TestMeWithValidToken(t *testing.T) {
	r := setupRouter(t)
	pair := registerAndLogin(t, r, "13800003333")

	// 有效 access token → 200，且返回的用户信息正确、不含密码
	w := getMe(t, r, "Bearer "+pair.AccessToken)
	if w.Code != http.StatusOK {
		t.Fatalf("valid token: expected 200, got %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID       uint   `json:"id"`
			Name     string `json:"name"`
			Phone    string `json:"phone"`
			Password string `json:"password"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if resp.Code != 0 || resp.Data.Phone != "13800003333" || resp.Data.Name != "dave" || resp.Data.ID == 0 {
		t.Fatalf("unexpected me payload: %+v", resp.Data)
	}
	if resp.Data.Password != "" {
		t.Fatal("password hash must never be serialized")
	}
}

func TestRefreshTokenFlow(t *testing.T) {
	r := setupRouter(t)
	pair := registerAndLogin(t, r, "13800003334")

	// refresh token 不能直接访问受保护接口
	if w := getMe(t, r, "Bearer "+pair.RefreshToken); w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token must not access /me, got %d", w.Code)
	}

	// 用 refresh token 换新的 access token
	w := postJSON(t, r, "/api/v1/users/refresh", map[string]string{
		"refresh_token": pair.RefreshToken,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("refresh failed: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if resp.Data.AccessToken == "" || resp.Data.AccessToken == pair.AccessToken {
		t.Fatal("expected a newly issued access token")
	}

	// 新 access token 能正常访问 /me
	if w := getMe(t, r, "Bearer "+resp.Data.AccessToken); w.Code != http.StatusOK {
		t.Fatalf("new access token should work, got %d", w.Code)
	}

	// access token 不能用于刷新
	bad := postJSON(t, r, "/api/v1/users/refresh", map[string]string{
		"refresh_token": pair.AccessToken,
	})
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("using access token to refresh should fail, got %d", bad.Code)
	}

	// 缺失 refresh_token 返回 400
	missing := postJSON(t, r, "/api/v1/users/refresh", map[string]string{})
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing refresh_token should be 400, got %d", missing.Code)
	}
}
