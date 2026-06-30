// Package note 实现笔记服务的核心业务逻辑。
// 负责笔记的创建与查询（通过 GORM 操作 MySQL），同时将创建事件发布到 RabbitMQ。
package note

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	notepb "xys-clone/proto/note"
)

// Server 实现了 proto/note 中定义的 NoteServiceServer 接口。
type Server struct {
	notepb.UnimplementedNoteServiceServer
	db *gorm.DB
	mq *Publisher
}

// NewServer 创建笔记服务实例。
// db: GORM 数据库连接；mq: RabbitMQ 发布器（可为 nil，跳过消息发布）。
func NewServer(db *gorm.DB, mq *Publisher) *Server {
	return &Server{db: db, mq: mq}
}

// AutoMigrate 自动建表（缺表补建，不删已有列）。
func (s *Server) AutoMigrate() error {
	return s.db.AutoMigrate(&Note{})
}

// CreateNote 将笔记写入 MySQL，成功后异步发布 RabbitMQ 事件。
func (s *Server) CreateNote(ctx context.Context, req *notepb.CreateNoteReq) (*notepb.CreateNoteResp, error) {
	note := &Note{
		UserID:    req.UserId,
		Title:     req.Title,
		Content:   req.Content,
		ImageURLs: strings.Join(req.ImageUrls, ","),
		Tags:      strings.Join(req.Tags, ","),
		CreatedAt: time.Now().Unix(),
	}

	if err := s.db.WithContext(ctx).Create(note).Error; err != nil {
		return nil, fmt.Errorf("创建笔记失败: %w", err)
	}

	// 异步发布 RabbitMQ 事件（不影响主流程）
	s.publishEvent(note.ID, note.UserID)

	return &notepb.CreateNoteResp{NoteId: note.ID}, nil
}

// GetNoteDetail 根据 note_id 查询笔记详情。
func (s *Server) GetNoteDetail(ctx context.Context, req *notepb.GetNoteDetailReq) (*notepb.NoteResponse, error) {
	var note Note
	err := s.db.WithContext(ctx).First(&note, req.NoteId).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("笔记 %d 不存在", req.NoteId)
	}
	if err != nil {
		return nil, fmt.Errorf("查询笔记失败: %w", err)
	}

	return &notepb.NoteResponse{
		Id:        note.ID,
		UserId:    note.UserID,
		Title:     note.Title,
		Content:   note.Content,
		ImageUrls: splitNonEmpty(note.ImageURLs),
		Tags:      splitNonEmpty(note.Tags),
		CreatedAt: note.CreatedAt,
	}, nil
}

// publishEvent 异步向 RabbitMQ 发布笔记创建事件。
func (s *Server) publishEvent(noteID, userID int64) {
	if s.mq == nil {
		return
	}
	go func() {
		body := fmt.Sprintf(`{"note_id":%d,"user_id":%d}`, noteID, userID)
		if err := s.mq.Publish("note.events", "note.created", []byte(body)); err != nil {
			log.Printf("[mq-error] 发布 note.created 失败: %v", err)
		} else {
			log.Printf("[mq] 已发送 note.created: %s", body)
		}
	}()
}

// splitNonEmpty 将逗号分隔字符串拆分，滤掉空串。
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
