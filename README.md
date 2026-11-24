# go-user-service

一个用于学习 Go Web 分层架构的练习项目：**Gin + GORM v2 + SQLite/MySQL + JWT + Cobra + Viper**。

## 目录结构

```
cmd/         程序入口与命令行（cobra），负责配置加载、DB 初始化、优雅启停
config/      viper 配置加载与热更新，dev.yaml 为默认配置
common/      通用能力：JWT、中间件（鉴权/限流/requestID/日志/Recovery）
routers/     Gin 路由与中间件链注册
controller/  薄控制器：只做请求绑定、调用 service、错误→HTTP 状态码映射
service/     业务逻辑层：校验规则、bcrypt、JWT 编排，定义与传输无关的业务错误
dao/         数据访问层：只含 GORM 查询，返回 dao 层错误
db/          数据库连接（SQLite / MySQL）
model/       GORM 模型
docs/        设计资料
```

### 分层架构与依赖方向

依赖严格单向，禁止反向或跨层引用：

```
router → controller → service → dao → db/model
                     ↘ 业务错误（service 定义）
```

- **controller** 不直接 import dao，也不写业务规则；它只维护一张
  `service 错误 → HTTP 状态码` 的映射表（见 `writeServiceError`）
- **service** 返回与 HTTP 解耦的业务错误（如 `ErrInvalidPhone`），
  同一套业务逻辑可被 HTTP、gRPC、定时任务等不同入口复用
- **dao** 只负责数据存取，把 GORM 错误翻译成 `ErrUserNotFound` 等数据层错误
- 跨层数据用 DTO（如 `RegisterInput`）传递，不把带 HTTP 标签的结构体泄漏到 service

### 依赖注入与可测试性

依赖在 `cmd` 中统一组装，逐层通过构造函数注入，全程无全局单例：

```
gorm.DB → dao.UserRepo → service.UserService → controller.UserHandler → router
```

- `service` 定义 `UserRepository` 接口（接口在消费方定义是 Go 的惯例），
  只依赖抽象而不依赖具体的 GORM 实现
- 单元测试用手写的 `fakeUserRepo` 替身注入，**完全不依赖数据库即可测业务规则**
  （见 `service/user_mock_test.go`），还能注入故障验证错误传播
- 切换存储实现（如改用其他 ORM）只需实现同一接口，业务代码零改动

## 快速开始

项目默认使用纯 Go 实现的 SQLite 驱动，**无需安装任何数据库**，开箱即跑：

```bash
go run . --config config/dev.yaml
# 服务监听 :8080；加 --debug=false 以 release 模式运行（不打印业务 SQL）
```

### 健康检查

- `GET /ping`：存活探针（liveness），只看进程是否在
- `GET /ready`：就绪探针（readiness），额外检查数据库连通性，供容器编排决定是否导流

切换到 MySQL：把 `config/dev.yaml` 里的 `db.driver` 改为 `mysql` 并填好连接信息即可。

### 配置与密钥安全

- 仓库只提交不含凭证的 `config/dev.yaml` 和 `config/config.example.yaml` 模板
- 本地私有配置请复制为 `config/dev.local.yaml`（已被 gitignore），通过 `--config` 指定
- 敏感项支持环境变量覆盖，**不要把真实密码写进任何被 git 跟踪的文件**：

```bash
export DB_USERNAME=your_user
export DB_PASSWORD=your_password
export JWT_SECRET=your_long_random_secret
go run . --config config/dev.yaml
```

- `jwt.secret` 留空时会生成一次性随机密钥（仅适合本地开发，重启后旧 token 失效）

## 接口示例

```bash
# 健康检查
curl http://127.0.0.1:8080/ping

# 注册
curl -X POST http://127.0.0.1:8080/api/v1/users/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"alice","phone":"13800000001","password":"secret123"}'

# 登录（返回 JWT）
curl -X POST http://127.0.0.1:8080/api/v1/users/login \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800000001","password":"secret123"}'

# 访问受保护接口（携带 Bearer Token，缺失/伪造/过期一律返回 401）
curl http://127.0.0.1:8080/api/v1/users/me \
  -H "Authorization: Bearer <上一步返回的 token>"
```

## 鉴权机制

- 登录成功后服务端签发 HS256 JWT，有效期 7 天
- `common.JWTAuth()` 中间件解析 `Authorization: Bearer <token>`，
  校验通过后把 `userID` 注入 gin.Context，失败统一返回 401
- 受保护接口通过 `common.ContextUserID` 读取当前用户，返回信息不含密码哈希
- 公开接口（注册/登录）与受保护接口在路由中分组管理，见 `routers/router.go`

## 运行测试

测试使用内存 SQLite，不依赖任何外部服务：

```bash
make test      # 全部单元测试
make cover     # 带覆盖率报告
make lint      # golangci-lint 静态检查
```

## 常用命令（Makefile）

```bash
make help     # 查看所有目标
make build    # 编译到 bin/
make fmt      # 格式化
make vet      # go vet
make tidy     # 整理依赖
make docker   # 构建镜像
```

## 容器化部署

多阶段构建，最终镜像基于 distroless、非 root 运行、纯静态二进制：

```bash
docker build -t go-user-service .
docker run -p 8080:8080 -e JWT_SECRET=$(openssl rand -hex 32) go-user-service
```

需要 MySQL 联调时，用 docker-compose 一条命令拉起应用和数据库：

```bash
docker compose up -d --build
```

配置通过环境变量覆盖（viper 自动映射，点号变下划线）：
`DB_DRIVER`、`DB_HOSTNAME`、`DB_PORT`、`DB_USERNAME`、`DB_PASSWORD`、`DB_DBNAME`、`JWT_SECRET`。

## 持续集成

`.github/workflows/ci.yml` 在每次 push / PR 时自动执行：
依赖下载 → gofmt 检查 → go vet → golangci-lint → `go test -race -cover` → 构建，
任何一步失败都会阻止合并。

## 可观测性：requestID 与结构化日志

每个请求都经过统一的中间件链（顺序见 `routers/router.go`）：

```
Recovery → RequestID → Logger → 业务 handler（受保护接口再接 JWTAuth）
```

- **RequestID**：优先透传入站 `X-Request-ID`（跨服务串联链路），缺失时用
  `crypto/rand` 生成 UUID v4，并回写到响应头，同时注入 gin.Context
- **结构化日志**：release 模式输出 JSON（含 request_id、status、method、path、
  latency_ms、client_ip），可直接接入 ELK/Loki 按 request_id 检索整条调用链
- **日志分级**：5xx 记 error、4xx 记 warn、其余 info
- **Recovery**：handler 发生 panic 时捕获并返回 500，记录带 requestID 的堆栈上下文，
  单请求崩溃不影响进程

排障时，拿到响应头里的 `X-Request-ID`，就能在日志中精确找到这一次请求的所有记录。

## 认证与令牌

采用 access + refresh 双令牌：

- access token：2 小时，用于访问受保护接口，Claims 中 `token_type=access`
- refresh token：7 天，仅用于调用 `/users/refresh` 换新 access token
- 中间件和刷新接口都会校验 `token_type`，两类令牌不能混用
- 每个 JWT 带唯一 `jti`，保证同一秒签发的令牌也不同，并为后续黑名单能力预留

## 跨域与安全头

- `CORS()`：来源白名单可配，预检请求直接返回 204；`*` 与凭证不共存
- `SecureHeaders()`：注入 X-Frame-Options、X-Content-Type-Options、CSP、HSTS 等

## 接口文档（Swagger）

注解写在各 handler 上，修改接口后重新生成文档：

```bash
make swag          # 生成到 docs/
go run .           # 访问 http://localhost:8080/swagger/index.html
```

## 接口限流

注册、登录等凭据接口按客户端 IP 限流，防止暴力破解密码和批量注册：

- 算法：进程内**固定窗口**（默认每 IP 每分钟 10 次，参数见 `routers/router.go`）
- 超限返回 `429 Too Many Requests`，并带 `Retry-After` 响应头提示等待秒数
- 限流器使用 mutex 保证并发安全，惰性过期不启动后台 goroutine
- 单元测试通过注入假时钟验证窗口重置，避免真实等待

> 适用边界：进程内限流只在**单实例**下准确；水平扩容到多个实例后，
> 各进程计数独立，应替换为 Redis 实现（`INCR` + `EXPIRE` 或令牌桶 Lua 脚本），
> `IPRateLimiter` 的方法签名可保持不变，业务代码无需改动。

## 本次重构修复了什么

- 修复无法编译的导入路径错误（`github/coder9527-lg/...` 缺少 `.com`）
- 移除对外部教程仓库 `shuwenhe/shuwen-shop` 的依赖，路由与处理器全部落在本项目内
- 路由从标准库 `net/http` 统一为真正的 Gin 引擎
- JWT：废弃的 `dgrijalva/jwt-go` 换成维护中的 `golang-jwt/jwt/v5`；
  修正 ES256 误用（对称密钥必须配 HS256）和错误的方法名；密钥改为配置注入
- 安全：密码使用 bcrypt 哈希存储（不再明文落库/打印日志）；
  随机昵称改用 `crypto/rand`；模型密码字段不序列化给前端
- 修复 `Register` 先返回成功后校验、校验失败不 return 的逻辑错误
- 修复 `main.go` 无条件 `os.Exit(-1)`，删除死代码
- ORM 统一到 GORM v2，删除停止维护的 v1 及硬编码账密
- 修复 model struct tag 语法错误，手机号加唯一索引
- 补齐统一响应结构、参数校验、单元测试与接口测试、`.gitignore`

## 仍待改进（后续学习方向）

- 配置通过环境变量覆盖（viper AutomaticEnv，已支持 DB/JWT 等关键项，可继续补全）
- CORS/安全响应头、统一错误码体系
- token 刷新机制与服务端失效（黑名单/版本号）
- 限流升级为 Redis 实现以支持多实例，service 依赖改用接口注入便于 mock
- 引入 OpenTelemetry，把 requestID 升级为分布式 trace
- Swagger 接口文档自动生成（swag）

## 健壮性设计要点

- HTTP Server 配置了 ReadHeader/Read/Write/Idle 超时，防御 Slowloris 慢速攻击
- 注册查重 + 数据库唯一索引双重保障，并发注册冲突正确返回 409（有并发测试覆盖）
- 密码长度限制 6-72（bcrypt 上限），密码以 bcrypt 哈希存储
- GORM 开启 TranslateError 与 IgnoreRecordNotFoundError，日志级别随环境切换
- 启动失败经错误通道回传，避免在 goroutine 中 os.Exit 跳过资源清理
