# Makefile for Storybook
# CGO 已禁用，使用纯 Go 构建

.PHONY: help build test test-race clean fmt vet lint run docker-build docker-run deps

# 变量定义
BINARY_NAME=storybook
SERVER_BINARY=bin/server
EMBEDUI_BINARY=bin/embedui

# Go 参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# 构建参数
CGO_ENABLED=0
GOFLAGS=-v
LDFLAGS=-s -w

# 默认目标
.DEFAULT_GOAL := help

## help: 显示此帮助信息
help:
	@echo "可用的 make 目标:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## deps: 下载依赖
deps:
	$(info 下载依赖...)
	@$(GOMOD) download
	@$(GOMOD) tidy

## fmt: 格式化代码
fmt:
	$(info 格式化代码...)
	@$(GOFMT) -s -w .

## vet: 运行 go vet 静态分析
vet:
	$(info 运行 go vet...)
	@CGO_ENABLED=$(CGO_ENABLED) $(GOVET) ./...

## lint: 运行 golangci-lint 代码检查
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "运行 golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint 未安装，跳过"; \
		echo "安装: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## test: 在 CGO 禁用模式下运行测试
test:
	$(info 运行测试...)
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -v -coverprofile=coverage.out ./...

## test-race: 使用 race detector 运行测试（需要 CGO_ENABLED=1）
test-race:
	$(info 运行 race detector 测试...)
	@CGO_ENABLED=1 $(GOTEST) -v -race ./...

## test-coverage: 运行测试并显示覆盖率
test-coverage: test
	$(info 测试覆盖率:)
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

## build: 构建单体 server
build: build-server

## build-server: 构建 server
build-server:
	$(info 构建 server...)
	@mkdir -p bin
	@CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(SERVER_BINARY) ./cmd/server

## build-embedui: 构建 embedui
build-embedui:
	$(info 构建 embedui...)
	@mkdir -p bin
	@CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(EMBEDUI_BINARY) ./cmd/embedui

## run: 运行 server（需要先设置环境变量）
run: build-server
	$(info 运行 server...)
	@$(SERVER_BINARY)

## clean: 清理构建文件
clean:
	$(info 清理构建文件...)
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "清理完成"

## docker-build: 构建 Docker 镜像
docker-build:
	$(info 构建 Docker 镜像...)
	@docker build -t $(BINARY_NAME):latest .

## docker-run: 运行 Docker 容器
docker-run:
	$(info 运行 Docker 容器...)
	@docker run -p 8080:8080 \
		-e DB_DRIVER=postgres \
		-e DB_DSN="host=postgres user=storybook password=storybook dbname=storybook port=5432 sslmode=disable" \
		-e JWT_SECRET=your-secret-key \
		$(BINARY_NAME):latest

## install-tools: 安装开发工具
install-tools:
	$(info 安装开发工具...)
	@$(GOGET) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "开发工具安装完成"

## check: 运行所有检查（fmt + vet + test）
check: fmt vet test
	$(info 所有检查通过！✓)
