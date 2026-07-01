// Package feed 实现 Feed 服务。
// 推拉结合：消费 note.created 事件 → 推入粉丝 Redis Timeline；
// 查询时优先从 Redis 读取，不足时回退 MySQL 拉模式。
package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	notepb "xys-clone/proto/note"
	feedpb "xys-clone/proto/feed"
	userpb "xys-clone/proto/user"
)

const (
	timelineKeyPrefix = "timeline:"   // Redis key: timeline:<user_id>
	timelineMaxLen    = 500           // 每人最多保留 500 条
	fanoutThreshold   = 1000          // 粉丝超过此数时跳过推模式
)

// Server 实现了 FeedServiceServer 接口。
type Server struct {
	feedpb.UnimplementedFeedServiceServer
	db         *gorm.DB
	rdb        *redis.Client
	userClient userpb.UserServiceClient
	noteClient notepb.NoteServiceClient
}

// NewServer 创建 Feed 服务实例。
func NewServer(db *gorm.DB, rdb *redis.Client, userClient userpb.UserServiceClient, noteClient notepb.NoteServiceClient) *Server {
	return &Server{db: db, rdb: rdb, userClient: userClient, noteClient: noteClient}
}

// noteCreatedEvent RabbitMQ 消息体。
type noteCreatedEvent struct {
	NoteID int64 `json:"note_id"`
	UserID int64 `json:"user_id"`
}

// HandleNoteCreated 被 RabbitMQ 消费者回调：查询作者粉丝 → 推入各粉丝 Redis Timeline。
func (s *Server) HandleNoteCreated(body []byte) {
	var evt noteCreatedEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("[feed] 解析 note.created 消息失败: %v", err)
		return
	}

	// 获取粉丝总数
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fResp, err := s.userClient.GetFollowers(ctx, &userpb.GetFollowersReq{
		UserId: evt.UserID, Page: 1, PageSize: 1,
	})
	if err != nil {
		log.Printf("[feed] 查询粉丝数失败 (user=%d): %v", evt.UserID, err)
		return
	}

	// 大 V 保护：粉丝数 > 1000 时跳过推模式
	if fResp.Total > fanoutThreshold {
		log.Printf("[feed] 用户 %d 粉丝数 %d > %d，跳过推模式，走拉模式",
			evt.UserID, fResp.Total, fanoutThreshold)
		return
	}

	// 分页获取全部粉丝 ID，逐页推入 Redis
	noteIDStr := strconv.FormatInt(evt.NoteID, 10)
	fanoutCount := 0

	for page := int32(1); ; page++ {
		resp, err := s.userClient.GetFollowers(ctx, &userpb.GetFollowersReq{
			UserId: evt.UserID, Page: page, PageSize: 200,
		})
		if err != nil {
			log.Printf("[feed] 查询粉丝列表失败 (user=%d, page=%d): %v", evt.UserID, page, err)
			return
		}
		if len(resp.Users) == 0 {
			break
		}

		for _, u := range resp.Users {
			key := timelineKeyPrefix + strconv.FormatInt(u.Id, 10)
			pipe := s.rdb.Pipeline()
			pipe.LPush(ctx, key, noteIDStr)
			pipe.LTrim(ctx, key, 0, timelineMaxLen-1)
			if _, err := pipe.Exec(ctx); err != nil {
				log.Printf("[feed] Redis 写入失败 (timeline=%s): %v", key, err)
				return
			}
			fanoutCount++
		}

		if int32(len(resp.Users)) < 200 {
			break
		}
	}

	log.Printf("[feed] note %d 作者 %d 粉丝 %d 人，已推入 %d 条 timeline",
		evt.NoteID, evt.UserID, fResp.Total, fanoutCount)
}

// ──────────── gRPC ────────────

// GetUserFeed 获取用户信息流。
// 优先从 Redis Timeline 读取；Redis 空或不足时，回退 MySQL 拉模式补全。
func (s *Server) GetUserFeed(ctx context.Context, req *feedpb.GetUserFeedReq) (*feedpb.GetUserFeedResp, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	// 1. 尝试 Redis 推模式
	if s.rdb != nil {
		key := timelineKeyPrefix + strconv.FormatInt(req.UserId, 10)
		noteIDs, err := s.rdb.LRange(ctx, key, start, stop).Result()
		if err == nil && len(noteIDs) > 0 {
			items, err := s.fetchNotes(ctx, noteIDs)
			if err != nil {
				log.Printf("[feed] 从 Redis 批量查笔记失败: %v，回退拉模式", err)
			} else {
				return &feedpb.GetUserFeedResp{Items: items, Total: -1}, nil
			}
		}
	}

	// 2. 回退 MySQL 拉模式
	log.Printf("[feed] 用户 %d 走拉模式 (page=%d)", req.UserId, page)
	return s.pullMode(ctx, req.UserId, page, pageSize)
}

// fetchNotes 根据笔记 ID 列表批量调用 note 服务查详情 + user 服务查作者。
func (s *Server) fetchNotes(ctx context.Context, noteIDs []string) ([]*feedpb.FeedItem, error) {
	items := make([]*feedpb.FeedItem, 0, len(noteIDs))
	for _, idStr := range noteIDs {
		nid, _ := strconv.ParseInt(idStr, 10, 64)
		if nid <= 0 {
			continue
		}

		note, err := s.noteClient.GetNoteDetail(ctx, &notepb.GetNoteDetailReq{NoteId: nid})
		if err != nil {
			log.Printf("[feed] GetNoteDetail(%d) 失败: %v", nid, err)
			continue
		}

		user, err := s.userClient.GetUser(ctx, &userpb.GetUserReq{UserId: note.UserId})
		if err != nil {
			log.Printf("[feed] GetUser(%d) 失败: %v", note.UserId, err)
		}

		item := &feedpb.FeedItem{
			NoteId:   note.Id,
			AuthorId: note.UserId,
			Title:    note.Title,
			Content:  note.Content,
			ImageUrls: note.ImageUrls,
			Tags:     note.Tags,
			CreatedAt: note.CreatedAt,
		}
		if user != nil {
			item.AuthorName = user.Username
			item.AuthorAvatar = user.Avatar
		}
		items = append(items, item)
	}
	return items, nil
}

// pullMode 回退拉模式：查 follows 表获取关注列表 → 查 notes 表按时间倒序分页。
func (s *Server) pullMode(ctx context.Context, userID int64, page, pageSize int32) (*feedpb.GetUserFeedResp, error) {
	// 获取关注的人 ID 列表
	var followeeIDs []int64
	s.db.WithContext(ctx).Model(&Follow{}).
		Where("follower_id = ?", userID).
		Pluck("followee_id", &followeeIDs)

	if len(followeeIDs) == 0 {
		return &feedpb.GetUserFeedResp{Items: nil, Total: 0}, nil
	}

	// 查这些人的笔记，按 created_at 倒序分页
	var notes []Note
	var total int64

	s.db.WithContext(ctx).Model(&Note{}).
		Where("user_id IN ?", followeeIDs).Count(&total)

	s.db.WithContext(ctx).
		Where("user_id IN ?", followeeIDs).
		Order("created_at DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&notes)

	// 转换为 FeedItem，批量查作者
	items := make([]*feedpb.FeedItem, len(notes))
	for i, n := range notes {
		user, _ := s.userClient.GetUser(ctx, &userpb.GetUserReq{UserId: n.UserID})
		item := &feedpb.FeedItem{
			NoteId:   n.ID,
			AuthorId: n.UserID,
			Title:    n.Title,
			Content:  n.Content,
			ImageUrls: splitNonEmpty(n.ImageURLs),
			Tags:     splitNonEmpty(n.Tags),
			CreatedAt: n.CreatedAt,
		}
		if user != nil {
			item.AuthorName = user.Username
			item.AuthorAvatar = user.Avatar
		}
		items[i] = item
	}

	return &feedpb.GetUserFeedResp{Items: items, Total: total}, nil
}

// ──────────── 连接与初始化 ────────────

// OpenRedis 根据环境变量 REDIS_ADDR 创建 Redis 客户端。
func OpenRedis() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}
	log.Println("Redis 连接成功")
	return rdb, nil
}

// DialUser 连接用户服务 gRPC。
func DialUser() (userpb.UserServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("USER_SERVICE_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("连接 UserService 失败: %w", err)
	}
	return userpb.NewUserServiceClient(conn), conn, nil
}

// DialNote 连接笔记服务 gRPC。
func DialNote() (notepb.NoteServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("NOTE_SERVICE_ADDR")
	if addr == "" {
		addr = "localhost:50052"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("连接 NoteService 失败: %w", err)
	}
	return notepb.NewNoteServiceClient(conn), conn, nil
}

// ──────────── GORM 模型（复用，仅用于拉模式查询）──

// Follow 与 user 包中的定义一致，仅用于查询。
type Follow struct {
	ID         int64 `gorm:"primaryKey"`
	FollowerID int64 `gorm:"column:follower_id"`
	FolloweeID int64 `gorm:"column:followee_id"`
}

func (Follow) TableName() string { return "follows" }

// Note 与 note 包中的定义一致，仅用于拉模式查询。
type Note struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    int64  `gorm:"column:user_id"`
	Title     string `gorm:"column:title"`
	Content   string `gorm:"column:content"`
	ImageURLs string `gorm:"column:image_urls"`
	Tags      string `gorm:"column:tags"`
	CreatedAt int64  `gorm:"column:created_at"`
}

func (Note) TableName() string { return "notes" }

// ──────────── helpers ────────────

func normalizePage(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var r []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			r = append(r, p)
		}
	}
	return r
}
