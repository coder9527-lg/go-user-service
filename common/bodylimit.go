package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes 限制请求体大小，防止超大 body 耗尽内存（OOM 攻击面）。
// 超过限制时，读取过程会返回错误，统一返回 413 Request Entity Too Large。
func MaxBodyBytes(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
