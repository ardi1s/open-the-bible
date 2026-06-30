// Package user 实现用户服务的核心业务逻辑。
// 提供用户查询与关注 / 取关 / 粉丝列表 / 关注列表功能。
package user

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"xys-clone/pkg/mq"
	userpb "xys-clone/proto/user"
)

// Server 实现了 proto/user 中定义的 UserServiceServer 接口。
type Server struct {
	userpb.UnimplementedUserServiceServer
	db *gorm.DB
	mq *mq.Publisher
}

// NewServer 创建用户服务实例。db 或 mq 为 nil 时对应能力降级。
func NewServer(db *gorm.DB, mq *mq.Publisher) *Server {
	return &Server{db: db, mq: mq}
}

// AutoMigrate 自动建表。
func (s *Server) AutoMigrate() error {
	if s.db == nil {
		return nil
	}
	return s.db.AutoMigrate(&User{}, &Follow{})
}

// ──────────── 查询用户 ────────────

// GetUser 根据 user_id 查询用户。优先查 MySQL，未命中时返回 mock 数据。
func (s *Server) GetUser(ctx context.Context, req *userpb.GetUserReq) (*userpb.UserResponse, error) {
	if s.db != nil {
		var user User
		err := s.db.WithContext(ctx).First(&user, req.UserId).Error
		if err == nil {
			return &userpb.UserResponse{
				Id: user.ID, Username: user.Username, Bio: user.Bio,
				Avatar: user.Avatar, CreatedAt: user.CreatedAt,
			}, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	return &userpb.UserResponse{
		Id: req.UserId, Username: "mock_user",
		Bio: "这是一个仿小红书的社交平台", Avatar: "https://picsum.photos/200",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}, nil
}

// ──────────── 关注 ────────────

// Follow 创建关注关系。重复关注直接返回 ok。
func (s *Server) Follow(ctx context.Context, req *userpb.FollowReq) (*userpb.FollowResp, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未连接")
	}
	if req.UserId == req.FolloweeId {
		return nil, fmt.Errorf("不能关注自己")
	}

	f := &Follow{
		FollowerID:   req.UserId,
		FolloweeID:   req.FolloweeId,
		SourceNoteID: req.SourceNoteId,
		CreatedAt:    time.Now().Unix(),
	}

	// INSERT IGNORE：已存在则忽略
	err := s.db.WithContext(ctx).Where("follower_id = ? AND followee_id = ?",
		req.UserId, req.FolloweeId).FirstOrCreate(f).Error
	if err != nil {
		return nil, fmt.Errorf("关注失败: %w", err)
	}

	s.publishFollowEvent("follow.add", req.UserId, req.FolloweeId, req.SourceNoteId)

	return &userpb.FollowResp{Ok: true}, nil
}

// Unfollow 取消关注。
func (s *Server) Unfollow(ctx context.Context, req *userpb.UnfollowReq) (*userpb.UnfollowResp, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未连接")
	}

	result := s.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", req.UserId, req.FolloweeId).
		Delete(&Follow{})
	if result.Error != nil {
		return nil, fmt.Errorf("取关失败: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		s.publishFollowEvent("follow.remove", req.UserId, req.FolloweeId, 0)
	}

	return &userpb.UnfollowResp{Ok: true}, nil
}

// ──────────── 查询 ────────────

// GetFollowers 查询某用户的粉丝列表（分页）。
func (s *Server) GetFollowers(ctx context.Context, req *userpb.GetFollowersReq) (*userpb.GetFollowersResp, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未连接")
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)

	// 子查询 JOIN users 拿到粉丝的用户名和头像
	type row struct {
		ID         int64  `gorm:"column:id"`
		Username   string `gorm:"column:username"`
		Avatar     string `gorm:"column:avatar"`
		FollowedAt int64  `gorm:"column:followed_at"`
	}

	var total int64
	s.db.Table("follows").Where("followee_id = ?", req.UserId).Count(&total)

	var rows []row
	s.db.WithContext(ctx).
		Table("follows").
		Select("users.id, users.username, users.avatar, follows.created_at AS followed_at").
		Joins("JOIN users ON users.id = follows.follower_id").
		Where("follows.followee_id = ?", req.UserId).
		Order("follows.created_at DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Scan(&rows)

	users := make([]*userpb.FollowUserInfo, len(rows))
	for i, r := range rows {
		users[i] = &userpb.FollowUserInfo{
			Id: r.ID, Username: r.Username, Avatar: r.Avatar, FollowedAt: r.FollowedAt,
		}
	}

	return &userpb.GetFollowersResp{Users: users, Total: total}, nil
}

// GetFollowing 查询某用户正在关注的人（分页）。
func (s *Server) GetFollowing(ctx context.Context, req *userpb.GetFollowingReq) (*userpb.GetFollowingResp, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未连接")
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)

	type row struct {
		ID         int64  `gorm:"column:id"`
		Username   string `gorm:"column:username"`
		Avatar     string `gorm:"column:avatar"`
		FollowedAt int64  `gorm:"column:followed_at"`
	}

	var total int64
	s.db.Table("follows").Where("follower_id = ?", req.UserId).Count(&total)

	var rows []row
	s.db.WithContext(ctx).
		Table("follows").
		Select("users.id, users.username, users.avatar, follows.created_at AS followed_at").
		Joins("JOIN users ON users.id = follows.followee_id").
		Where("follows.follower_id = ?", req.UserId).
		Order("follows.created_at DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Scan(&rows)

	users := make([]*userpb.FollowUserInfo, len(rows))
	for i, r := range rows {
		users[i] = &userpb.FollowUserInfo{
			Id: r.ID, Username: r.Username, Avatar: r.Avatar, FollowedAt: r.FollowedAt,
		}
	}

	return &userpb.GetFollowingResp{Users: users, Total: total}, nil
}

// IsFollowing 查询 user_id 是否关注了 target_id。
func (s *Server) IsFollowing(ctx context.Context, req *userpb.IsFollowingReq) (*userpb.IsFollowingResp, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未连接")
	}

	var count int64
	s.db.WithContext(ctx).Model(&Follow{}).
		Where("follower_id = ? AND followee_id = ?", req.UserId, req.TargetId).
		Count(&count)

	return &userpb.IsFollowingResp{Following: count > 0}, nil
}

// ──────────── 内部方法 ────────────

func (s *Server) publishFollowEvent(routingKey string, followerID, followeeID, sourceNoteID int64) {
	if s.mq == nil {
		return
	}
	go func() {
		body := fmt.Sprintf(`{"follower_id":%d,"followee_id":%d,"source_note_id":%d}`,
			followerID, followeeID, sourceNoteID)
		if err := s.mq.Publish("follow.events", routingKey, []byte(body)); err != nil {
			log.Printf("[mq-error] 发布 %s 失败: %v", routingKey, err)
		} else {
			log.Printf("[mq] 已发送 %s: %s", routingKey, body)
		}
	}()
}

func normalizePage(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
