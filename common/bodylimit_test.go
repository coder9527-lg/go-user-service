package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxBodyBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytes(10))
	r.POST("/x", func(c *gin.Context) {
		buf := make([]byte, 64)
		n, err := c.Request.Body.Read(buf)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(http.StatusOK, strings.Repeat("a", n))
	})

	// 小 body 正常
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("small"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("small body should pass, got %d", w.Code)
	}

	// 超过 10 字节的 body 读取应失败
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("this-body-is-way-too-long"))
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body should be rejected, got %d", w2.Code)
	}
}
