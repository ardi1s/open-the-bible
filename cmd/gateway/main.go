// API 网关（Gateway）入口。
// 基于 Gin 框架提供 HTTP RESTful 接口，内部通过 gRPC 调用下游微服务。
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

	feedpb "xys-clone/proto/feed"
	interactionpb "xys-clone/proto/interaction"
	notepb "xys-clone/proto/note"
	rankpb "xys-clone/proto/rank"
	userpb "xys-clone/proto/user"
)

func main() {
	userConn := mustDial("USER_SERVICE_ADDR", "localhost:50051")
	defer userConn.Close()
	userClient := userpb.NewUserServiceClient(userConn)

	noteConn := mustDial("NOTE_SERVICE_ADDR", "localhost:50052")
	defer noteConn.Close()
	noteClient := notepb.NewNoteServiceClient(noteConn)

	feedConn := mustDial("FEED_SERVICE_ADDR", "localhost:50053")
	defer feedConn.Close()
	feedClient := feedpb.NewFeedServiceClient(feedConn)

	rankConn := mustDial("RANK_SERVICE_ADDR", "localhost:50055")
	defer rankConn.Close()
	rankClient := rankpb.NewRankServiceClient(rankConn)

	interactionConn := mustDial("INTERACTION_SERVICE_ADDR", "localhost:50054")
	defer interactionConn.Close()
	interactionClient := interactionpb.NewInteractionServiceClient(interactionConn)

	r := gin.Default()

	// ──────────── 基础 ────────────

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ──────────── 用户 ────────────

	r.GET("/api/user/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 必须为整数"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.GetUser(ctx, &userpb.GetUserReq{UserId: userID})
		if err != nil {
			log.Printf("[error] GetUser(%d) 失败: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "查询用户失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"id": resp.Id, "username": resp.Username, "bio": resp.Bio,
				"avatar": resp.Avatar, "created_at": resp.CreatedAt,
			},
		})
	})

	// ── 关注 ──

	// POST /api/user/:id/follow  —  当前用户关注 :id
	r.POST("/api/user/:id/follow", func(c *gin.Context) {
		followeeID := mustParseInt(c.Param("id"))

		var body struct {
			UserID       int64 `json:"user_id"       binding:"required"`
			SourceNoteID int64 `json:"source_note_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.Follow(ctx, &userpb.FollowReq{
			UserId: body.UserID, FolloweeId: followeeID, SourceNoteId: body.SourceNoteID,
		})
		if err != nil {
			log.Printf("[error] Follow(%d→%d) 失败: %v", body.UserID, followeeID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}

		log.Printf("[info] Follow ok: %d → %d", body.UserID, followeeID)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// DELETE /api/user/:id/follow  —  当前用户取关 :id
	r.DELETE("/api/user/:id/follow", func(c *gin.Context) {
		followeeID := mustParseInt(c.Param("id"))

		var body struct {
			UserID int64 `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.Unfollow(ctx, &userpb.UnfollowReq{
			UserId: body.UserID, FolloweeId: followeeID,
		})
		if err != nil {
			log.Printf("[error] Unfollow(%d→%d) 失败: %v", body.UserID, followeeID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}

		log.Printf("[info] Unfollow ok: %d → %d", body.UserID, followeeID)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// GET /api/user/:id/followers?page=1&size=20  —  查看 :id 的粉丝
	r.GET("/api/user/:id/followers", func(c *gin.Context) {
		userID := mustParseInt(c.Param("id"))
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.GetFollowers(ctx, &userpb.GetFollowersReq{
			UserId: userID, Page: int32(page), PageSize: int32(size),
		})
		if err != nil {
			log.Printf("[error] GetFollowers(%d) 失败: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"users": resp.Users, "total": resp.Total,
		}})
	})

	// GET /api/user/:id/following?page=1&size=20  —  查看 :id 关注的人
	r.GET("/api/user/:id/following", func(c *gin.Context) {
		userID := mustParseInt(c.Param("id"))
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.GetFollowing(ctx, &userpb.GetFollowingReq{
			UserId: userID, Page: int32(page), PageSize: int32(size),
		})
		if err != nil {
			log.Printf("[error] GetFollowing(%d) 失败: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"users": resp.Users, "total": resp.Total,
		}})
	})

	// GET /api/user/:id/is-following?target_id=2  —  检查关注状态
	r.GET("/api/user/:id/is-following", func(c *gin.Context) {
		userID := mustParseInt(c.Param("id"))
		targetID := mustParseInt(c.Query("target_id"))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := userClient.IsFollowing(ctx, &userpb.IsFollowingReq{
			UserId: userID, TargetId: targetID,
		})
		if err != nil {
			log.Printf("[error] IsFollowing(%d→%d) 失败: %v", userID, targetID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"following": resp.Following}})
	})

	// ──────────── Feed ────────────

	// GET /api/feed?user_id=1&page=1&size=10  —  获取用户信息流
	r.GET("/api/feed", func(c *gin.Context) {
		userID := mustParseInt(c.Query("user_id"))
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		resp, err := feedClient.GetUserFeed(ctx, &feedpb.GetUserFeedReq{
			UserId: userID, Page: int32(page), PageSize: int32(size),
		})
		if err != nil {
			log.Printf("[error] GetUserFeed(%d) 失败: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "获取信息流失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{"items": resp.Items, "total": resp.Total},
		})
	})

	// ──────────── 排行榜 ────────────

	// GET /api/rank/hot?count=20  —  热门笔记 Top N
	r.GET("/api/rank/hot", func(c *gin.Context) {
		count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resp, err := rankClient.GetHotNotes(ctx, &rankpb.GetHotNotesReq{Count: int32(count)})
		if err != nil {
			log.Printf("[error] GetHotNotes 失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "获取排行榜失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": resp.Items}})
	})

	// ──────────── 互动 ────────────

	// POST /api/notes/:id/like  —  点赞
	r.POST("/api/notes/:id/like", func(c *gin.Context) {
		noteID := mustParseInt(c.Param("id"))
		var body struct {
			UserID int64 `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.LikeNote(ctx, &interactionpb.LikeNoteReq{UserId: body.UserID, NoteId: noteID})
		if err != nil {
			log.Printf("[error] LikeNote(%d) 失败: %v", noteID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// DELETE /api/notes/:id/like  —  取消点赞
	r.DELETE("/api/notes/:id/like", func(c *gin.Context) {
		noteID := mustParseInt(c.Param("id"))
		var body struct {
			UserID int64 `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.UnlikeNote(ctx, &interactionpb.UnlikeNoteReq{UserId: body.UserID, NoteId: noteID})
		if err != nil {
			log.Printf("[error] UnlikeNote(%d) 失败: %v", noteID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// POST /api/notes/:id/collect  —  收藏
	r.POST("/api/notes/:id/collect", func(c *gin.Context) {
		noteID := mustParseInt(c.Param("id"))
		var body struct {
			UserID int64 `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.CollectNote(ctx, &interactionpb.CollectNoteReq{UserId: body.UserID, NoteId: noteID})
		if err != nil {
			log.Printf("[error] CollectNote(%d) 失败: %v", noteID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// DELETE /api/notes/:id/collect  —  取消收藏
	r.DELETE("/api/notes/:id/collect", func(c *gin.Context) {
		noteID := mustParseInt(c.Param("id"))
		var body struct {
			UserID int64 `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.UncollectNote(ctx, &interactionpb.UncollectNoteReq{UserId: body.UserID, NoteId: noteID})
		if err != nil {
			log.Printf("[error] UncollectNote(%d) 失败: %v", noteID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// POST /api/notes/:id/comment  —  评论
	r.POST("/api/notes/:id/comment", func(c *gin.Context) {
		noteID := mustParseInt(c.Param("id"))
		var body struct {
			UserID   int64  `json:"user_id"   binding:"required"`
			Content  string `json:"content"   binding:"required"`
			ParentID int64  `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 和 content 为必填"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.CommentOnNote(ctx, &interactionpb.CommentOnNoteReq{
			UserId: body.UserID, NoteId: noteID, Content: body.Content, ParentId: body.ParentID,
		})
		if err != nil {
			log.Printf("[error] CommentOnNote(%d) 失败: %v", noteID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"comment_id": resp.CommentId}})
	})

	// DELETE /api/comments/:id  —  删除评论
	r.DELETE("/api/comments/:id", func(c *gin.Context) {
		commentID := mustParseInt(c.Param("id"))
		var body struct {
			UserID int64 `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "user_id 为必填"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.DeleteComment(ctx, &interactionpb.DeleteCommentReq{
			CommentId: commentID, UserId: body.UserID,
		})
		if err != nil {
			log.Printf("[error] DeleteComment(%d) 失败: %v", commentID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"ok": resp.Ok}})
	})

	// GET /api/notes/:id/interactions?user_id=1  —  互动汇总
	r.GET("/api/notes/:id/interactions", func(c *gin.Context) {
		noteID := mustParseInt(c.Param("id"))
		userID := mustParseInt(c.Query("user_id"))
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		resp, err := interactionClient.GetNoteInteractions(ctx, &interactionpb.GetNoteInteractionsReq{
			NoteId: noteID, UserId: userID,
		})
		if err != nil {
			log.Printf("[error] GetNoteInteractions(%d) 失败: %v", noteID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"like_count":    resp.LikeCount,
			"collect_count": resp.CollectCount,
			"comment_count": resp.CommentCount,
			"is_liked":      resp.IsLiked,
			"is_collected":  resp.IsCollected,
			"comments":      resp.Comments,
		}})
	})

	// ──────────── 笔记 ────────────

	r.POST("/api/notes", func(c *gin.Context) {
		var req struct {
			UserID    int64    `json:"user_id"    binding:"required"`
			Title     string   `json:"title"      binding:"required"`
			Content   string   `json:"content"    binding:"required"`
			ImageURLs []string `json:"image_urls"`
			Tags      []string `json:"tags"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "error": "参数不合法"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := noteClient.CreateNote(ctx, &notepb.CreateNoteReq{
			UserId: req.UserID, Title: req.Title, Content: req.Content,
			ImageUrls: req.ImageURLs, Tags: req.Tags,
		})
		if err != nil {
			log.Printf("[error] CreateNote 失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "创建笔记失败"})
			return
		}

		log.Printf("[info] 笔记创建成功 note_id=%d", resp.NoteId)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"note_id": resp.NoteId}})
	})

	log.Println("Gateway 已启动，监听 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gateway 启动失败: %v", err)
	}
}

// ──────────── helpers ────────────

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

func mustParseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
