package controller

import (
	"net/http"

	"github.com/coder9527-lg/go-user-service/service"
	"github.com/gin-gonic/gin"
)

// UserHandler 承载用户相关的 HTTP 处理器，通过构造函数注入业务服务。
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type registerRequest struct {
	Name     string `json:"name" form:"name"`
	Phone    string `json:"phone" form:"phone"`
	Password string `json:"password" form:"password"`
}

type loginRequest struct {
	Phone    string `json:"phone" form:"phone"`
	Password string `json:"password" form:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

// Register 用户注册
//
// @Summary      用户注册
// @Description  手机号+密码注册，密码以 bcrypt 哈希存储
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest true "注册信息"
// @Success      200   {object}  response{data=registerResult}
// @Failure      422   {object}  response
// @Failure      409   {object}  response
// @Router       /users/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBind(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request params")
		return
	}

	user, err := h.userService.Register(service.RegisterInput{
		Name:     req.Name,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	success(c, gin.H{"id": user.ID, "name": user.Name, "phone": user.Phone})
}

// Login 用户登录
//
// @Summary      用户登录
// @Description  校验密码后下发 access/refresh 双令牌
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest true "登录信息"
// @Success      200   {object}  response{data=common.TokenPair}
// @Failure      401   {object}  response
// @Router       /users/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBind(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request params")
		return
	}

	pair, err := h.userService.Login(service.LoginInput{
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	success(c, pair)
}

// Refresh 刷新访问令牌
//
// @Summary      刷新 access token
// @Description  用 refresh token 换取新的 access token
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest true "刷新令牌"
// @Success      200   {object}  response{data=refreshResult}
// @Failure      400   {object}  response
// @Failure      401   {object}  response
// @Router       /users/refresh [post]
func (h *UserHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBind(&req); err != nil || req.RefreshToken == "" {
		fail(c, http.StatusBadRequest, "refresh_token is required")
		return
	}

	access, err := h.userService.Refresh(req.RefreshToken)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	success(c, gin.H{"access_token": access})
}
