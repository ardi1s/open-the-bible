.PHONY: proto proto-user proto-clean run-user run-gateway build-user build-gateway \
        test test-cover lint ci-check \
        docker-build docker-build-all docker-push docker-push-all \
        docker-up docker-down docker-build-user docker-build-gateway

# 自动探测 GOPATH，确保 protoc 插件在 PATH 中
GOPATH    := $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)

PROTO_DIR := proto
OUT_DIR   := proto

# ============================================================
# ★ 服务清单 —— 新增服务只需在此加一行
#   CI/CD 工作流中的 SERVICES 对应更新为 '["user","gateway","note",...]'
# ============================================================
SERVICES  := user gateway

REGISTRY  ?= docker.io
DOCKER_USERNAME ?=
IMAGE_NS  ?= $(DOCKER_USERNAME)
GIT_SHA   := $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo "local")

# ============================================================
# 工具安装
# ============================================================
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# ============================================================
# Proto 桩代码
# ============================================================
proto: proto-user

proto-user:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(OUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(OUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/user/user.proto

proto-clean:
	rm -f $(PROTO_DIR)/user/*.pb.go

# ============================================================
# 运行
# ============================================================
run-user:
	go run ./cmd/user/main.go

run-gateway:
	go run ./cmd/gateway/main.go

# ============================================================
# 编译
# ============================================================
build-user:
	go build -o bin/user ./cmd/user

build-gateway:
	go build -o bin/gateway ./cmd/gateway

build-all:
	@for svc in $(SERVICES); do \
		echo "→ build $$svc"; \
		go build -o bin/$$svc ./cmd/$$svc; \
	done

# ============================================================
# 测试 & 代码检查（CI 核心步骤）
# ============================================================
test:
	go test ./... -v -count=1

test-cover:
	go test ./... -v -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint:
	go vet ./...

ci-check: lint test
	@echo "✅ CI check passed"

# ============================================================
# Docker 构建（按服务）
# 用法: make docker-build-user DOCKER_USERNAME=xxx
# ============================================================
define docker_build
docker-build-$(1):
	docker build \
		-t $(REGISTRY)/$(IMAGE_NS)/xys-$(1):$(GIT_SHA) \
		-t $(REGISTRY)/$(IMAGE_NS)/xys-$(1):latest \
		-f cmd/$(1)/Dockerfile .
endef
$(foreach svc,$(SERVICES),$(eval $(call docker_build,$(svc))))

docker-build: $(addprefix docker-build-,$(SERVICES))

# ============================================================
# Docker 推送（按服务）
# ============================================================
define docker_push
docker-push-$(1):
	docker push $(REGISTRY)/$(IMAGE_NS)/xys-$(1):$(GIT_SHA)
	docker push $(REGISTRY)/$(IMAGE_NS)/xys-$(1):latest
endef
$(foreach svc,$(SERVICES),$(eval $(call docker_push,$(svc))))

docker-push: $(addprefix docker-push-,$(SERVICES))

# ============================================================
# Docker Compose
# ============================================================
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
