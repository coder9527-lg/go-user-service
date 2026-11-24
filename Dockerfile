# ---------- 构建阶段 ----------
FROM golang:1.26-alpine AS builder

WORKDIR /build

# 先复制依赖清单，利用 Docker 层缓存：依赖不变时无需重新下载
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 纯静态编译，可放入无 libc 的 distroless 镜像
# 纯 Go 的 SQLite 驱动（glebarez/sqlite）不依赖 CGO
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/go-user-service .

# ---------- 运行阶段 ----------
# distroless 静态镜像：不含 shell 和包管理器，攻击面最小
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# 复制二进制与默认配置
COPY --from=builder /out/go-user-service /app/go-user-service
COPY --from=builder /build/config /app/config

# SQLite 数据目录（使用文件型 SQLite 时挂载卷到此）
ENV DB_DRIVER=sqlite \
    DB_SQLITE_DSN=/data/go-user-service.db \
    GIN_MODE=release
VOLUME ["/data"]

EXPOSE 8080

USER nonroot:nonroot

# 容器内置健康检查：调用内置 health 子命令访问 /ping（distroless 无 shell/curl）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/go-user-service", "health"]

ENTRYPOINT ["/app/go-user-service", "--config", "/app/config/dev.yaml"]
