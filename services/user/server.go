// Package user 实现用户服务的核心业务逻辑。
// GetUser 优先从 MySQL（GORM）查询，未命中时返回 mock 数据。
package user

import (
	"context"
	"time"

	"gorm.io/gorm"

	userpb "xys-clone/proto/user"
)

// Server 实现了 proto/user 中定义的 UserServiceServer 接口。
type Server struct {
	userpb.UnimplementedUserServiceServer
	db *gorm.DB
}

// NewServer 创建用户服务实例。db 为 nil 时纯 mock 运行。
func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

// AutoMigrate 自动建表。
func (s *Server) AutoMigrate() error {
	if s.db == nil {
		return nil
	}
	return s.db.AutoMigrate(&User{})
}

// GetUser 根据 user_id 查询用户。
// 优先查 MySQL，未命中时返回 mock 数据（保证开发阶段始终可用）。
func (s *Server) GetUser(ctx context.Context, req *userpb.GetUserReq) (*userpb.UserResponse, error) {
	// 有 DB 连接时，先查真实数据
	if s.db != nil {
		var user User
		err := s.db.WithContext(ctx).First(&user, req.UserId).Error
		if err == nil {
			return &userpb.UserResponse{
				Id:        user.ID,
				Username:  user.Username,
				Bio:       user.Bio,
				Avatar:    user.Avatar,
				CreatedAt: user.CreatedAt,
			}, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		// 未命中 → 走 mock
	}

	// mock 兜底
	return &userpb.UserResponse{
		Id:        req.UserId,
		Username:  "mock_user",
		Bio:       "这是一个仿小红书的社交平台",
		Avatar:    "https://picsum.photos/200",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}, nil
}
