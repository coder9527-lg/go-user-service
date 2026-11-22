package common

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger 结构化访问日志：每个请求输出一条带 requestID 的 JSON 日志。
// 替代 gin.Default() 自带的文本格式 Logger，便于日志系统检索与聚合。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		fields := logrus.Fields{
			"request_id": c.GetString(ContextRequestID),
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"latency_ms": time.Since(start).Milliseconds(),
			"client_ip":  c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		entry := logrus.WithFields(fields)
		switch {
		case c.Writer.Status() >= 500:
			entry.Error("http request")
		case c.Writer.Status() >= 400:
			entry.Warn("http request")
		default:
			entry.Info("http request")
		}
	}
}

// Recovery 捕获 handler 中的 panic，记录带 requestID 的错误日志并返回 500，
// 避免单个请求的崩溃导致整个进程退出。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logrus.WithFields(logrus.Fields{
					"request_id": c.GetString(ContextRequestID),
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"panic":      rec,
				}).Error("panic recovered")
				c.AbortWithStatusJSON(500, gin.H{
					"code": 500,
					"msg":  "internal server error",
				})
			}
		}()
		c.Next()
	}
}
