package common

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit 基于客户端 IP 的限流中间件。
// limit 为每个窗口允许的最大请求数，window 为窗口时长。
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewIPRateLimiter(limit, window)
	return func(c *gin.Context) {
		allow, retryAfter := limiter.Allow(c.ClientIP())
		if !allow {
			seconds := int(retryAfter.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": http.StatusTooManyRequests,
				"msg":  "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}
