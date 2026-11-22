package main

import (
	"os"

	"github.com/coder9527-lg/go-user-service/cmd"
)

// @title           go-user-service 用户服务 API
// @version         1.0
// @description     Gin + GORM 分层架构练习项目，提供注册、登录、令牌刷新与用户信息接口。
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer <access_token>
func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
