package router

import (
	"time"

	"github.com/coder9527-lg/go-user-service/common"
	"github.com/coder9527-lg/go-user-service/controller"
	_ "github.com/coder9527-lg/go-user-service/docs" // swag 生成的 OpenAPI 文档
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// 凭据接口的限流参数：同一 IP 每分钟最多 10 次，防止暴力破解与批量注册。
const (
	authRateLimit  = 10
	authRateWindow = time.Minute
)

// New 构建并返回 Gin 路由引擎，便于在测试中直接启动。
func New() *gin.Engine {
	// 用 gin.New() 而非 gin.Default()，以便用自定义的结构化日志和 Recovery 替换默认中间件。
	r := gin.New()

	// 中间件顺序：Recovery 最外层兜底 → RequestID → 安全头/CORS → Logger → 业务。
	r.Use(common.Recovery())
	r.Use(common.RequestID())
	r.Use(common.SecureHeaders())
	r.Use(common.CORS())
	r.Use(common.Logger())

	r.GET("/ping", controller.Ping)
	r.GET("/ready", controller.Ready)
	// Swagger UI：访问 /swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		// 公开接口：无需登录，但凭据接口要按 IP 限流防爆破。
		api.POST("/users/register", common.RateLimit(authRateLimit, authRateWindow), controller.Register)
		api.POST("/users/login", common.RateLimit(authRateLimit, authRateWindow), controller.Login)
		api.POST("/users/refresh", common.RateLimit(authRateLimit, authRateWindow), controller.Refresh)

		// 受保护接口：必须携带有效 access token
		auth := api.Group("/", common.JWTAuth())
		{
			auth.GET("/users/me", controller.Me)
		}
	}
	return r
}

// Run 在指定端口启动 HTTP 服务。
func Run(addr string) error {
	return New().Run(addr)
}
