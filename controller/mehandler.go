package controller

import (
	"net/http"

	"github.com/coder9527-lg/go-user-service/common"
	"github.com/gin-gonic/gin"
)

// Me 当前用户信息
//
// @Summary      获取当前登录用户
// @Description  凭 access token 返回当前用户信息
// @Tags         用户
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response{data=meResult}
// @Failure      401  {object}  response
// @Router       /users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	// 中间件已保证该值存在且类型正确；做一次防御性断言。
	raw, ok := c.Get(common.ContextUserID)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, ok := raw.(uint)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	// 只返回安全字段；User.Password 已通过 json:"-" 保证不会外泄，这里再显式挑选字段。
	success(c, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"phone": user.Phone,
	})
}
