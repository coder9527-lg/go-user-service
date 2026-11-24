package router

import (
	"time"

	"github.com/coder9527-lg/go-user-service/common"
	"github.com/coder9527-lg/go-user-service/controller"
	_ "github.com/coder9527-lg/go-user-service/docs" // swag 生成的 OpenAPI 文档
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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

	configureTrustedProxies(r)

	// 中间件顺序：Recovery 最外层兜底 → RequestID → 安全头/CORS → Logger → 业务。
	r.Use(common.Recovery())
	r.Use(common.RequestID())
	r.Use(common.SecureHeaders())
	r.Use(common.CORS())
	r.Use(common.Logger())
	r.Use(common.MaxBodyBytes(1 << 20)) // 限制请求体 1MB，防超大 body 耗尽内存

	r.GET("/ping", controller.Ping)
	r.GET("/ready", controller.Ready)

	// 三个凭据接口共享同一个限流器，使同一 IP 的总配额为 10 次/分钟，
	// 而不是每个接口各 10 次、合计 30 次。
	authLimiter := common.NewIPRateLimiter(authRateLimit, authRateWindow)

	api := r.Group("/api/v1")
	{
		// 公开接口：无需登录，但凭据接口要按 IP 限流防爆破。
		api.POST("/users/register", common.RateLimit(authLimiter), controller.Register)
		api.POST("/users/login", common.RateLimit(authLimiter), controller.Login)
		api.POST("/users/refresh", common.RateLimit(authLimiter), controller.Refresh)

		// 受保护接口：必须携带有效 access token
		auth := api.Group("/", common.JWTAuth())
		{
			auth.GET("/users/me", controller.Me)
		}
	}

	// Swagger 仅在非 release 模式暴露，避免生产环境公开接口结构。
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	return r
}

// configureTrustedProxies 配置可信代理，决定 ClientIP 是否信任 X-Forwarded-For。
// 默认不信任任何代理（直连部署最安全）；部署在已知反向代理/网关后时，
// 通过配置项 trusted_proxies 传入代理网段（如 ["10.0.0.0/8"]）。
func configureTrustedProxies(r *gin.Engine) {
	proxies := viper.GetStringSlice("trusted_proxies")
	if len(proxies) == 0 {
		// nil 表示不信任任何代理，ClientIP 直接取 TCP 远程地址，伪造的头不生效。
		_ = r.SetTrustedProxies(nil)
		return
	}
	_ = r.SetTrustedProxies(proxies)
}

// Run 在指定端口启动 HTTP 服务。
func Run(addr string) error {
	return New().Run(addr)
}
