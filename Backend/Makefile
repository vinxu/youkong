.PHONY: build run dev clean test lint docker-build docker-run migrate

# 变量
APP_NAME := youkong-server
BUILD_DIR := ./build
MAIN_FILE := ./cmd/server/main.go

# Go 相关
GOCMD := go
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Docker 相关
DOCKER := docker
IMAGE_NAME := youkong-backend
IMAGE_TAG := latest

# 构建
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# 运行
run: build
	@echo "Running..."
	$(BUILD_DIR)/$(APP_NAME)

# 开发模式运行
dev:
	@echo "Running in development mode..."
	$(GORUN) $(MAIN_FILE)

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# 测试
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# 代码格式化
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

# 代码检查
lint:
	@echo "Linting..."
	@golangci-lint run ./...

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Docker 构建
docker-build:
	@echo "Building Docker image..."
	$(DOCKER) build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Docker 运行
docker-run:
	@echo "Running Docker container..."
	$(DOCKER) run -p 8080:8080 --env-file .env $(IMAGE_NAME):$(IMAGE_TAG)

# 数据库迁移
migrate:
	@echo "Running migrations..."
	@mysql -u$(DB_USER) -p$(DB_PASSWORD) -h$(DB_HOST) $(DB_NAME) < migrations/001_init.sql
	@echo "Migrations complete"

# 帮助
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Build and run the application"
	@echo "  dev          - Run in development mode (no build)"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  deps         - Download dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  migrate      - Run database migrations"
