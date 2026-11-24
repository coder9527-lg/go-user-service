package common

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RateLimit 基于客户端 IP 的限流中间件，复用调用方传入的 limiter。
// 多个路由传入同一个 limiter 即可共享配额，避免每个接口各自计数导致总配额翻倍。
func RateLimit(limiter *IPRateLimiter) gin.HandlerFunc {
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
