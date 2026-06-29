// API 网关（Gateway）入口。
// 基于 Gin 框架提供 HTTP RESTful 接口，内部通过 gRPC 调用下游微服务。
// 启动方式：
//   - 本地开发：make run-gateway（需先启动 user 服务）
//   - Docker：docker-compose up -d
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userpb "xys-clone/proto/user"
)

func main() {
	// 从环境变量 USER_SERVICE_ADDR 读取 user 服务地址。
	// 本地默认为 localhost:50051，Docker 内为 user:50051（走 docker-compose 网络）。
	userAddr := os.Getenv("USER_SERVICE_ADDR")
	if userAddr == "" {
		userAddr = "localhost:50051"
	}

	// 创建 gRPC 客户端，连接 user 服务（开发阶段使用明文传输）
	conn, err := grpc.NewClient(
		userAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("连接 UserService 失败 (%s): %v", userAddr, err)
	}
	defer conn.Close()

	userClient := userpb.NewUserServiceClient(conn)

	// 初始化 Gin 引擎（默认包含 Logger 和 Recovery 中间件）
	r := gin.Default()

	// ──────────── 路由注册 ────────────

	// GET /health —— 健康检查接口，用于负载均衡 & CI 冒烟测试
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// GET /api/user/:id —— 通过 gRPC 调用 user 服务查询用户信息
	r.GET("/api/user/:id", func(c *gin.Context) {
		// 1. 解析并校验路径参数
		idStr := c.Param("id")
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("[warn] 非法 user_id: %q", idStr)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  400,
				"error": "user_id 必须为整数",
			})
			return
		}

		// 2. 调用 gRPC（3 秒超时，避免下游故障拖死网关）
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.GetUser(ctx, &userpb.GetUserReq{UserId: userID})
		if err != nil {
			log.Printf("[error] GetUser(%d) 调用失败: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":  500,
				"error": "查询用户失败，请稍后重试",
			})
			return
		}

		log.Printf("[info] GetUser(%d) 成功 => username=%s", userID, resp.Username)

		// 3. 返回统一 JSON 结构
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"id":         resp.Id,
				"username":   resp.Username,
				"bio":        resp.Bio,
				"avatar":     resp.Avatar,
				"created_at": resp.CreatedAt,
			},
		})
	})

	// 启动 HTTP 服务器
	log.Println("Gateway 已启动，监听 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gateway 启动失败: %v", err)
	}
}
