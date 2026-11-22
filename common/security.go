package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// CORS 处理跨域与预检请求。
// 允许来源取配置 cors.allowed_origins（逗号分隔）；为 "*" 时允许任意来源，
// 此时按规范不能同时开启 Allow-Credentials。
func CORS() gin.HandlerFunc {
	allowed := viper.GetStringSlice("cors.allowed_origins")
	if len(allowed) == 0 {
		allowed = []string{"*"}
	}
	allowAll := len(allowed) == 1 && allowed[0] == "*"
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allowedSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		switch {
		case allowAll:
			c.Header("Access-Control-Allow-Origin", "*")
		case origin != "":
			if _, ok := allowedSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, "+RequestIDHeader)
		c.Header("Access-Control-Max-Age", "86400")

		// 浏览器预检请求无需进入业务逻辑，直接 204 返回。
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SecureHeaders 注入一组标准安全响应头，降低 XSS、点击劫持、MIME 嗅探等风险。
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// HSTS 仅在 HTTPS 下有意义；本地 http 开发时浏览器会忽略，无副作用。
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
