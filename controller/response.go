package controller

import (
	"errors"
	"net/http"

	"github.com/coder9527-lg/go-user-service/service"
	"github.com/gin-gonic/gin"
)

// 统一响应结构：code=0 表示成功，非 0 表示业务错误。
type response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response{Code: 0, Msg: "ok", Data: data})
}

func fail(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, response{Code: httpStatus, Msg: msg})
}

// writeServiceError 把 service 层的业务错误统一翻译成 HTTP 响应。
// controller 不自己判断业务分支，只维护这张“错误 → 状态码”映射表，
// 这样新增业务错误时只需改一处。
func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidPhone), errors.Is(err, service.ErrInvalidPassword):
		fail(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, service.ErrPhoneRegistered):
		fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInvalidToken):
		fail(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		fail(c, http.StatusUnauthorized, "user no longer exists")
	default:
		fail(c, http.StatusInternalServerError, "internal server error")
	}
}
