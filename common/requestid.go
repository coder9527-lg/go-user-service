package common

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader 是承载请求 ID 的 HTTP 头，入站透传和出站回写都用它。
const RequestIDHeader = "X-Request-ID"

// ContextRequestID 是 gin.Context 中存放请求 ID 的键。
const ContextRequestID = "requestID"

// RequestID 为每个请求分配唯一 ID：
// 优先复用调用方传入的 X-Request-ID（便于跨服务串联链路），否则新生成一个。
// 同时写回响应头，方便客户端和日志对照。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}
		c.Set(ContextRequestID, rid)
		c.Writer.Header().Set(RequestIDHeader, rid)
		c.Next()
	}
}

// newRequestID 生成 16 字节随机数并格式化为 UUID v4 风格字符串，
// 全程使用 crypto/rand，不依赖第三方 uuid 库。
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败极罕见；兜底用一个固定前缀+时间，保证仍有标识可用。
		return "rid-fallback"
	}
	// 按 RFC 4122 设置 version(4) 与 variant(10xx) 位。
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}
