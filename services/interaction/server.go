// Package interaction 实现互动服务的核心业务逻辑。
// 提供点赞、收藏、评论及互动汇总查询，操作成功后发布 RabbitMQ 事件。
package interaction

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	interactionpb "xys-clone/proto/interaction"
	"xys-clone/pkg/mq"
	userpb "xys-clone/proto/user"
)

// Server 实现了 InteractionServiceServer 接口。
type Server struct {
	interactionpb.UnimplementedInteractionServiceServer
	db         *gorm.DB
	mq         *mq.Publisher
	userClient userpb.UserServiceClient
}

// NewServer 创建互动服务实例。
func NewServer(db *gorm.DB, mq *mq.Publisher, userClient userpb.UserServiceClient) *Server {
	return &Server{db: db, mq: mq, userClient: userClient}
}

// AutoMigrate 自动建表。
func (s *Server) AutoMigrate() error {
	return s.db.AutoMigrate(&Like{}, &Collect{}, &Comment{})
}

// ──────────── 点赞 ────────────

func (s *Server) LikeNote(ctx context.Context, req *interactionpb.LikeNoteReq) (*interactionpb.LikeNoteResp, error) {
	like := &Like{UserID: req.UserId, NoteID: req.NoteId, CreatedAt: time.Now().Unix()}
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND note_id = ?", req.UserId, req.NoteId).
		FirstOrCreate(like).Error
	if err != nil {
		return nil, fmt.Errorf("点赞失败: %w", err)
	}
	s.publish("interaction.like", req.UserId, req.NoteId, 0, "")
	return &interactionpb.LikeNoteResp{Ok: true}, nil
}

func (s *Server) UnlikeNote(ctx context.Context, req *interactionpb.UnlikeNoteReq) (*interactionpb.UnlikeNoteResp, error) {
	res := s.db.WithContext(ctx).
		Where("user_id = ? AND note_id = ?", req.UserId, req.NoteId).
		Delete(&Like{})
	if res.Error != nil {
		return nil, fmt.Errorf("取消点赞失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		s.publish("interaction.unlike", req.UserId, req.NoteId, 0, "")
	}
	return &interactionpb.UnlikeNoteResp{Ok: true}, nil
}

// ──────────── 收藏 ────────────

func (s *Server) CollectNote(ctx context.Context, req *interactionpb.CollectNoteReq) (*interactionpb.CollectNoteResp, error) {
	c := &Collect{UserID: req.UserId, NoteID: req.NoteId, CreatedAt: time.Now().Unix()}
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND note_id = ?", req.UserId, req.NoteId).
		FirstOrCreate(c).Error
	if err != nil {
		return nil, fmt.Errorf("收藏失败: %w", err)
	}
	s.publish("interaction.collect", req.UserId, req.NoteId, 0, "")
	return &interactionpb.CollectNoteResp{Ok: true}, nil
}

func (s *Server) UncollectNote(ctx context.Context, req *interactionpb.UncollectNoteReq) (*interactionpb.UncollectNoteResp, error) {
	res := s.db.WithContext(ctx).
		Where("user_id = ? AND note_id = ?", req.UserId, req.NoteId).
		Delete(&Collect{})
	if res.Error != nil {
		return nil, fmt.Errorf("取消收藏失败: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		s.publish("interaction.uncollect", req.UserId, req.NoteId, 0, "")
	}
	return &interactionpb.UncollectNoteResp{Ok: true}, nil
}

// ──────────── 评论 ────────────

func (s *Server) CommentOnNote(ctx context.Context, req *interactionpb.CommentOnNoteReq) (*interactionpb.CommentOnNoteResp, error) {
	c := &Comment{
		UserID:    req.UserId,
		NoteID:    req.NoteId,
		Content:   req.Content,
		ParentID:  req.ParentId,
		CreatedAt: time.Now().Unix(),
	}
	if err := s.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, fmt.Errorf("评论失败: %w", err)
	}
	s.publish("interaction.comment", req.UserId, req.NoteId, c.ID, req.Content)
	return &interactionpb.CommentOnNoteResp{CommentId: c.ID}, nil
}

func (s *Server) DeleteComment(ctx context.Context, req *interactionpb.DeleteCommentReq) (*interactionpb.DeleteCommentResp, error) {
	res := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", req.CommentId, req.UserId).
		Delete(&Comment{})
	if res.Error != nil {
		return nil, fmt.Errorf("删除评论失败: %w", res.Error)
	}
	return &interactionpb.DeleteCommentResp{Ok: res.RowsAffected > 0}, nil
}

// ──────────── 互动汇总 ────────────

func (s *Server) GetNoteInteractions(ctx context.Context, req *interactionpb.GetNoteInteractionsReq) (*interactionpb.GetNoteInteractionsResp, error) {
	var likeCount, collectCount, commentCount int64
	s.db.WithContext(ctx).Model(&Like{}).Where("note_id = ?", req.NoteId).Count(&likeCount)
	s.db.WithContext(ctx).Model(&Collect{}).Where("note_id = ?", req.NoteId).Count(&collectCount)
	s.db.WithContext(ctx).Model(&Comment{}).Where("note_id = ?", req.NoteId).Count(&commentCount)

	var isLiked, isCollected bool
	if req.UserId > 0 {
		var lc, cc int64
		s.db.WithContext(ctx).Model(&Like{}).Where("user_id = ? AND note_id = ?", req.UserId, req.NoteId).Count(&lc)
		s.db.WithContext(ctx).Model(&Collect{}).Where("user_id = ? AND note_id = ?", req.UserId, req.NoteId).Count(&cc)
		isLiked = lc > 0
		isCollected = cc > 0
	}

	type commentRow struct {
		ID        int64  `gorm:"column:id"`
		UserID    int64  `gorm:"column:user_id"`
		Username  string `gorm:"column:username"`
		Avatar    string `gorm:"column:avatar"`
		Content   string `gorm:"column:content"`
		ParentID  int64  `gorm:"column:parent_id"`
		CreatedAt int64  `gorm:"column:created_at"`
	}
	var rows []commentRow
	s.db.WithContext(ctx).
		Table("comments").
		Select("comments.id, comments.user_id, users.username, users.avatar, comments.content, comments.parent_id, comments.created_at").
		Joins("LEFT JOIN users ON users.id = comments.user_id").
		Where("comments.note_id = ?", req.NoteId).
		Order("comments.created_at ASC").
		Limit(10).
		Scan(&rows)

	comments := make([]*interactionpb.CommentItem, len(rows))
	for i, r := range rows {
		comments[i] = &interactionpb.CommentItem{
			Id: r.ID, UserId: r.UserID, Username: r.Username, Avatar: r.Avatar,
			Content: r.Content, ParentId: r.ParentID, CreatedAt: r.CreatedAt,
		}
	}

	return &interactionpb.GetNoteInteractionsResp{
		LikeCount: likeCount, CollectCount: collectCount, CommentCount: commentCount,
		IsLiked: isLiked, IsCollected: isCollected, Comments: comments,
	}, nil
}

// ──────────── 内部 ────────────

func (s *Server) publish(routingKey string, userID, noteID, commentID int64, content string) {
	if s.mq == nil {
		return
	}
	go func() {
		body := fmt.Sprintf(`{"user_id":%d,"note_id":%d,"created_at":%d`, userID, noteID, time.Now().Unix())
		if commentID > 0 {
			body += fmt.Sprintf(`,"comment_id":%d,"content":"%s"`, commentID, content)
		}
		body += "}"
		if err := s.mq.Publish("interaction.events", routingKey, []byte(body)); err != nil {
			log.Printf("[mq-error] 发布 %s 失败: %v", routingKey, err)
		} else {
			log.Printf("[mq] 已发送 %s: %s", routingKey, body)
		}
	}()
}

// ──────────── gRPC 客户端 ────────────

// DialUser 连接用户服务 gRPC。
func DialUser() (userpb.UserServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("USER_SERVICE_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return userpb.NewUserServiceClient(conn), conn, nil
}
