// Package user 实现用户服务的核心业务逻辑。
// 当前为开发初期，GetUser 返回 mock 数据，后续将接入 MySQL/Redis。
package user

import (
	"context"
	"time"

	userpb "xys-clone/proto/user"
)

// Server 实现了 proto/user 中定义的 UserServiceServer 接口。
// 嵌入 UnimplementedUserServiceServer 以保证向前兼容（proto 新增方法时编译不会报错）。
type Server struct {
	userpb.UnimplementedUserServiceServer
}

// NewServer 创建用户服务实例。
func NewServer() *Server {
	return &Server{}
}

// GetUser 根据请求中的 user_id 返回用户信息。
// 当前为 mock 实现，始终返回固定数据，后续改为从数据库查询。
func (s *Server) GetUser(ctx context.Context, req *userpb.GetUserReq) (*userpb.UserResponse, error) {
	// TODO: 接入 MySQL 后改为真实查询
	return &userpb.UserResponse{
		Id:        req.UserId,
		Username:  "mock_user",
		Bio:       "这是一个仿小红书的社交平台",
		Avatar:    "https://picsum.photos/200",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}, nil
}
