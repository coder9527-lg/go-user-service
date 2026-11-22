# 可通过环境变量覆盖：make build BIN_DIR=/tmp/out
APP        := go-user-service
BIN_DIR    := bin
BIN        := $(BIN_DIR)/$(APP)
PKG        := ./...
CONFIG     := config/dev.yaml
IMAGE      := go-user-service:latest

# .DEFAULT_GOAL 指定只敲 make 时执行的目标
.DEFAULT_GOAL := help

.PHONY: help build run test cover vet fmt lint tidy clean docker

help: ## 显示所有可用目标
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

build: ## 编译二进制到 bin/
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN) .

run: ## 本地直接运行
	go run . --config $(CONFIG)

test: ## 运行全部单元测试
	go test $(PKG) -count=1

cover: ## 运行测试并输出覆盖率报告
	go test $(PKG) -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

vet: ## 静态检查 go vet
	go vet $(PKG)

# 只检查项目自身的 Go 目录，排除本地模块缓存等非源码目录
GO_DIRS := cmd common config controller dao db model routers

fmt: ## 格式化代码（含 import 顺序，需要 goimports）
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w $(GO_DIRS) *.go ; \
	else \
		gofmt -s -w $(GO_DIRS) *.go ; \
	fi

fmt-check: ## CI 用：检查格式，不合规则失败
	@out=$$(gofmt -l $(GO_DIRS) *.go 2>/dev/null); \
	if [ -n "$$out" ]; then echo "未通过 gofmt："; echo "$$out"; exit 1; fi

lint: ## 运行 golangci-lint（需要本地已安装）
	golangci-lint run ./...

tidy: ## 整理依赖
	go mod tidy

swag: ## 重新生成 Swagger 文档（需要 swag 工具）
	swag init --parseDependency --parseInternal

clean: ## 清理构建产物
	rm -rf $(BIN_DIR) coverage.out
	rm -f *.db *.sqlite

docker: ## 构建 Docker 镜像
	docker build -t $(IMAGE) .
