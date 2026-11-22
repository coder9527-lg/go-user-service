package common

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextUserID 是 gin.Context 中存放当前登录用户 ID 的键。
const ContextUserID = "userID"

// JWTAuth 校验 Authorization 头中的 Bearer Token，
// 通过后把用户 ID 注入 context；失败一律返回 401 并中止请求。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			abortUnauthorized(c, "missing authorization header")
			return
		}

		// 期望格式：Bearer <token>
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			abortUnauthorized(c, "authorization header must be 'Bearer <token>'")
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			abortUnauthorized(c, "invalid or expired token")
			return
		}
		// 只有 access token 能访问受保护接口；refresh token 仅用于换新。
		if claims.TokenType != TokenTypeAccess {
			abortUnauthorized(c, "invalid token type")
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code": http.StatusUnauthorized,
		"msg":  msg,
	})
}
