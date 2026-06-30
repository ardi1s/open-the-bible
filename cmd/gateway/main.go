// API 网关（Gateway）入口。
// 基于 Gin 框架提供 HTTP RESTful 接口，内部通过 gRPC 调用下游微服务。
// 启动方式：
//   - 本地开发：make run-gateway（需先启动 user + note 服务）
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

	notepb "xys-clone/proto/note"
	userpb "xys-clone/proto/user"
)

func main() {
	// ── gRPC 客户端 ──

	userConn := mustDial("USER_SERVICE_ADDR", "localhost:50051")
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	noteConn := mustDial("NOTE_SERVICE_ADDR", "localhost:50052")
	defer noteConn.Close()
	noteClient := notepb.NewNoteServiceClient(noteConn)

	// 初始化 Gin 引擎
	r := gin.Default()

	// ──────────── 路由注册 ────────────

	// GET /health —— 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// GET /api/user/:id —— 查询用户信息
	r.GET("/api/user/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("[warn] 非法 user_id: %q", idStr)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 必须为整数"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.GetUser(ctx, &userpb.GetUserReq{UserId: userID})
		if err != nil {
			log.Printf("[error] GetUser(%d) 调用失败: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "查询用户失败，请稍后重试"})
			return
		}

		log.Printf("[info] GetUser(%d) 成功 => username=%s", userID, resp.Username)

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

	// POST /api/notes —— 创建笔记
	// 请求体 JSON：{"user_id":1, "title":"...", "content":"...", "image_urls":[...], "tags":[...]}
	r.POST("/api/notes", func(c *gin.Context) {
		var req struct {
			UserID    int64    `json:"user_id"    binding:"required"`
			Title     string   `json:"title"      binding:"required"`
			Content   string   `json:"content"    binding:"required"`
			ImageURLs []string `json:"image_urls"`
			Tags      []string `json:"tags"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[warn] 创建笔记参数不合法: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "参数不合法，user_id/title/content 为必填"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := noteClient.CreateNote(ctx, &notepb.CreateNoteReq{
			UserId:    req.UserID,
			Title:     req.Title,
			Content:   req.Content,
			ImageUrls: req.ImageURLs,
			Tags:      req.Tags,
		})
		if err != nil {
			log.Printf("[error] CreateNote 调用失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "创建笔记失败，请稍后重试"})
			return
		}

		log.Printf("[info] 笔记创建成功 note_id=%d user_id=%d", resp.NoteId, req.UserID)

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{"note_id": resp.NoteId},
		})
	})

	log.Println("Gateway 已启动，监听 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gateway 启动失败: %v", err)
	}
}

// mustDial 从环境变量读取地址，创建 gRPC 连接，失败直接 fatal。
func mustDial(envKey, defaultAddr string) *grpc.ClientConn {
	addr := os.Getenv(envKey)
	if addr == "" {
		addr = defaultAddr
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接 %s 失败 (%s): %v", envKey, addr, err)
	}
	return conn
}
