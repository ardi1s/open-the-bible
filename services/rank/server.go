// Package rank 实现排行榜服务。
// 消费 note / interaction 事件，维护 Redis Sorted Set "hot_notes"。
// 热度分 = (likes*3 + collects*5 + comments*2) / (hours_since_post + 2)
// 每小时通过 robfig/cron 将 Top N 快照写入 MySQL。
package rank

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	notepb "xys-clone/proto/note"
	rankpb "xys-clone/proto/rank"
	userpb "xys-clone/proto/user"
)

const (
	hotNotesKey   = "hot_notes"          // Redis Sorted Set
	noteScoreHash = "note:score:"        // Redis Hash: note:score:<id>
	snapshotTopN  = 100                  // 每次快照保存 Top 100
	// 权重
	weightLike    = 3.0
	weightCollect = 5.0
	weightComment = 2.0
)

// Server 实现了 RankServiceServer 接口。
type Server struct {
	rankpb.UnimplementedRankServiceServer
	rdb        *redis.Client
	db         *gorm.DB
	noteClient notepb.NoteServiceClient
	userClient userpb.UserServiceClient
	cron       *cron.Cron
}

// NewServer 创建排行榜服务实例。
func NewServer(rdb *redis.Client, db *gorm.DB, noteClient notepb.NoteServiceClient, userClient userpb.UserServiceClient) *Server {
	s := &Server{
		rdb:        rdb,
		db:         db,
		noteClient: noteClient,
		userClient: userClient,
		cron:       cron.New(),
	}
	// 每小时整点跑一次快照
	s.cron.AddFunc("0 * * * *", s.snapshot)
	s.cron.Start()
	// 启动时也跑一次（如果有数据）
	go s.snapshot()
	return s
}

// AutoMigrate 自动建表。
func (s *Server) AutoMigrate() error {
	return s.db.AutoMigrate(&RankSnapshot{})
}

// ──────────── 事件处理 ────────────

// HandleEvent 根据 routingKey 更新 Redis 热度分。
func (s *Server) HandleEvent(routingKey string, body []byte) {
	// 解析 note_id
	noteID := parseNoteID(body)
	if noteID == 0 {
		return
	}

	hashKey := noteScoreHash + strconv.FormatInt(noteID, 10)
	log.Printf("[rank] 收到 %s note=%d", routingKey, noteID)

	switch routingKey {
	case "note.created":
		// 记录笔记创建时间
		ts := parseCreatedAt(body)
		if ts > 0 {
			s.rdb.HSet(context.Background(), hashKey, "created_at", ts)
		}
		s.recalcScore(noteID)
	case "interaction.like":
		s.rdb.HIncrBy(context.Background(), hashKey, "likes", 1)
		s.recalcScore(noteID)
	case "interaction.collect":
		s.rdb.HIncrBy(context.Background(), hashKey, "collects", 1)
		s.recalcScore(noteID)
	case "interaction.comment":
		s.rdb.HIncrBy(context.Background(), hashKey, "comments", 1)
		s.recalcScore(noteID)
	}
}

// recalcScore 读取 hash 中的计数 + 时间，计算热度分后 ZADD 到 hot_notes。
func (s *Server) recalcScore(noteID int64) {
	ctx := context.Background()
	hashKey := noteScoreHash + strconv.FormatInt(noteID, 10)

	vals, err := s.rdb.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return
	}

	likes := parseFloat(vals["likes"])
	collects := parseFloat(vals["collects"])
	comments := parseFloat(vals["comments"])
	createdAt := parseFloat(vals["created_at"])

	// 若无创建时间（已从 Redis 清除），尝试从 note 服务查
	if createdAt == 0 {
		note, err := s.noteClient.GetNoteDetail(ctx, &notepb.GetNoteDetailReq{NoteId: noteID})
		if err != nil {
			return
		}
		createdAt = float64(note.CreatedAt)
		s.rdb.HSet(ctx, hashKey, "created_at", int64(createdAt))
	}

	baseScore := likes*weightLike + collects*weightCollect + comments*weightComment
	hours := math.Max(float64(time.Now().Unix())/3600.0-createdAt/3600.0, 0)
	heat := baseScore / (hours + 2.0)

	// 防零分——新笔记即使无互动也有微小的基础分
	if heat < 0.01 && createdAt > 0 {
		heat = 0.01 / (hours + 2.0)
	}

	s.rdb.ZAdd(ctx, hotNotesKey, redis.Z{Score: heat, Member: noteID})
	log.Printf("[rank] note=%d 热度更新: base=%.1f hours=%.1f heat=%.4f", noteID, baseScore, hours, heat)
}

// ──────────── gRPC ────────────

// GetHotNotes 返回热门笔记 Top N。
func (s *Server) GetHotNotes(ctx context.Context, req *rankpb.GetHotNotesReq) (*rankpb.GetHotNotesResp, error) {
	count := req.Count
	if count <= 0 || count > 100 {
		count = 20
	}

	// ZREVRANGE WITHSCORES
	results, err := s.rdb.ZRevRangeWithScores(ctx, hotNotesKey, 0, int64(count)-1).Result()
	if err != nil {
		return nil, fmt.Errorf("查询排行榜失败: %w", err)
	}

	items := make([]*rankpb.RankItem, 0, len(results))
	for _, z := range results {
		nid, _ := strconv.ParseInt(fmt.Sprint(z.Member), 10, 64)
		note, err := s.noteClient.GetNoteDetail(ctx, &notepb.GetNoteDetailReq{NoteId: nid})
		if err != nil {
			continue
		}
		user, _ := s.userClient.GetUser(ctx, &userpb.GetUserReq{UserId: note.UserId})

		item := &rankpb.RankItem{
			NoteId:   note.Id,
			AuthorId: note.UserId,
			Title:    note.Title,
			Content:  note.Content,
			ImageUrls: note.ImageUrls,
			Tags:     note.Tags,
			CreatedAt: note.CreatedAt,
			HeatScore: z.Score,
		}
		if user != nil {
			item.AuthorName = user.Username
			item.AuthorAvatar = user.Avatar
		}

		// 截断长内容
		runes := []rune(note.Content)
		if len(runes) > 200 {
			item.Content = string(runes[:200]) + "..."
		}

		items = append(items, item)
	}

	return &rankpb.GetHotNotesResp{Items: items}, nil
}

// ──────────── 快照 ────────────

func (s *Server) snapshot() {
	ctx := context.Background()
	results, err := s.rdb.ZRevRangeWithScores(ctx, hotNotesKey, 0, snapshotTopN-1).Result()
	if err != nil || len(results) == 0 {
		return
	}

	hour := time.Now().Format("2006-01-02T15")
	for i, z := range results {
		nid, _ := strconv.ParseInt(fmt.Sprint(z.Member), 10, 64)
		s.db.Create(&RankSnapshot{
			NoteID:    nid,
			Hour:      hour,
			HeatScore: z.Score,
			RankPos:   int32(i + 1),
		})
	}
	log.Printf("[rank] 快照完成 hour=%s count=%d", hour, len(results))
}

// ──────────── gRPC 客户端 ────────────

// DialNote 连接笔记服务。
func DialNote() (notepb.NoteServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("NOTE_SERVICE_ADDR")
	if addr == "" {
		addr = "localhost:50052"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return notepb.NewNoteServiceClient(conn), conn, nil
}

// DialUser 连接用户服务。
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

// ──────────── Redis ────────────

// OpenRedis 创建 Redis 客户端。
func OpenRedis() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 1}) // DB 1，避免和 feed 冲突
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	log.Println("Redis（rank）连接成功")
	return rdb, nil
}

// ──────────── helpers ────────────

func parseNoteID(body []byte) int64 {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return 0
	}
	if v, ok := m["note_id"]; ok {
		return toInt64(v)
	}
	return 0
}

func parseCreatedAt(body []byte) int64 {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return 0
	}
	if v, ok := m["created_at"]; ok {
		return toInt64(v)
	}
	return 0
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case int64:
		return n
	}
	return 0
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
