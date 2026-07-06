// Package agent 实现分析服务。
// 包含粉丝增长追踪、标签分析、规则驱动运营建议。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	agentpb "xys-clone/proto/agent"
	"xys-clone/pkg/llm"
	notepb "xys-clone/proto/note"
	userpb "xys-clone/proto/user"
)

// Server 实现了 AgentServiceServer 接口。
type Server struct {
	agentpb.UnimplementedAgentServiceServer
	rdb        *redis.Client
	db         *gorm.DB
	noteClient notepb.NoteServiceClient
	userClient userpb.UserServiceClient
	llm        *llm.Client
	cron       *cron.Cron
}

// NewServer 创建分析服务实例。
// llmClient 为 nil 时 GetSuggestions 回退规则引擎。
func NewServer(rdb *redis.Client, db *gorm.DB, noteClient notepb.NoteServiceClient, userClient userpb.UserServiceClient, llmClient *llm.Client) *Server {
	s := &Server{
		rdb:        rdb,
		db:         db,
		noteClient: noteClient,
		userClient: userClient,
		llm:        llmClient,
		cron:       cron.New(),
	}
	// 每天凌晨 0:10 固化 Redis 标签数据到 MySQL
	s.cron.AddFunc("10 0 * * *", s.persistTagAnalytics)
	// 每小时固化粉丝增长数据
	s.cron.AddFunc("0 * * * *", s.persistFansGrowth)
	s.cron.Start()
	return s
}

// AutoMigrate 自动建表。
func (s *Server) AutoMigrate() error {
	return s.db.AutoMigrate(&NoteFansGrowth{}, &TagAnalytics{})
}

// ──────────── 事件处理 ────────────

func (s *Server) HandleEvent(routingKey string, body []byte) {
	ctx := context.Background()
	switch routingKey {
	case "follow.add":
		s.handleFollowAdd(ctx, body)
	case "note.created":
		s.handleNoteCreated(ctx, body)
	case "interaction.like", "interaction.collect", "interaction.comment":
		s.handleInteraction(ctx, routingKey, body)
	}
}

// ── 粉丝增长 ──

func (s *Server) handleFollowAdd(ctx context.Context, body []byte) {
	var m struct {
		FollowerID   int64 `json:"follower_id"`
		FolloweeID   int64 `json:"followee_id"`
		SourceNoteID int64 `json:"source_note_id"`
	}
	if err := json.Unmarshal(body, &m); err != nil || m.SourceNoteID == 0 {
		return
	}

	hour := time.Now().Format("2006-01-02T15")
	key := fmt.Sprintf("fans_growth:%d:%s", m.SourceNoteID, hour)
	count, _ := s.rdb.Incr(ctx, key).Result()

	// 第一次出现时设 2 小时 TTL（防止冷数据堆积，固化后会清理）
	if count == 1 {
		s.rdb.Expire(ctx, key, 48*time.Hour)
	}

	log.Printf("[agent] fans_growth: note=%d hour=%s count=%d", m.SourceNoteID, hour, count)
}

// ── 标签统计 ──

func (s *Server) handleNoteCreated(ctx context.Context, body []byte) {
	var m struct {
		NoteID int64  `json:"note_id"`
		UserID int64  `json:"user_id"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return
	}

	note, err := s.noteClient.GetNoteDetail(ctx, &notepb.GetNoteDetailReq{NoteId: m.NoteID})
	if err != nil {
		return
	}

	for _, tag := range note.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		s.rdb.HIncrBy(ctx, "tag:note_count", tag, 1)
	}
}

func (s *Server) handleInteraction(ctx context.Context, routingKey string, body []byte) {
	var m struct {
		NoteID int64 `json:"note_id"`
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return
	}

	note, err := s.noteClient.GetNoteDetail(ctx, &notepb.GetNoteDetailReq{NoteId: m.NoteID})
	if err != nil {
		return
	}

	for _, tag := range note.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		switch routingKey {
		case "interaction.like":
			s.rdb.HIncrBy(ctx, "tag:likes", tag, 1)
		case "interaction.collect":
			s.rdb.HIncrBy(ctx, "tag:collects", tag, 1)
		case "interaction.comment":
			s.rdb.HIncrBy(ctx, "tag:comments", tag, 1)
		}
	}
}

// ──────────── 定时持久化 ────────────

func (s *Server) persistFansGrowth() {
	ctx := context.Background()
	// 读所有 fans_growth:* 的 key，写入 MySQL 后删除
	keys, err := s.rdb.Keys(ctx, "fans_growth:*").Result()
	if err != nil || len(keys) == 0 {
		return
	}

	for _, key := range keys {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) != 3 {
			continue
		}
		noteID, _ := strconv.ParseInt(parts[1], 10, 64)
		hour := parts[2]
		count, _ := s.rdb.Get(ctx, key).Int64()
		if count == 0 {
			continue
		}

		s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&NoteFansGrowth{
			NoteID: noteID, Hour: hour, Count: count,
		})

		s.rdb.Del(ctx, key)
	}
	log.Printf("[agent] 粉丝增长持久化完成 keys=%d", len(keys))
}

func (s *Server) persistTagAnalytics() {
	ctx := context.Background()
	date := time.Now().Add(-24 * time.Hour).Format("2006-01-02")

	// 获取所有有统计的 tag
	tags, _ := s.rdb.HKeys(ctx, "tag:note_count").Result()
	seen := map[string]bool{}
	for _, t := range tags {
		seen[t] = true
	}
	allTags := tags
	for _, h := range []string{"tag:likes", "tag:collects", "tag:comments"} {
		extra, _ := s.rdb.HKeys(ctx, h).Result()
		for _, t := range extra {
			if !seen[t] {
				allTags = append(allTags, t)
				seen[t] = true
			}
		}
	}

	for _, tag := range allTags {
		nc, _ := s.rdb.HGet(ctx, "tag:note_count", tag).Int64()
		l, _ := s.rdb.HGet(ctx, "tag:likes", tag).Int64()
		c, _ := s.rdb.HGet(ctx, "tag:collects", tag).Int64()
		cm, _ := s.rdb.HGet(ctx, "tag:comments", tag).Int64()

		totalInteractions := l + c + cm
		var rate float64
		if nc > 0 {
			rate = float64(totalInteractions) / float64(nc)
		}

		s.db.Create(&TagAnalytics{
			Tag: tag, NoteCount: nc,
			TotalLikes: l, TotalCollects: c, TotalComments: cm,
			EngagementRate: rate, Date: date,
		})
	}

	// 清理 Redis 中的当天计数（软重置，保留汇总）
	for _, h := range []string{"tag:note_count", "tag:likes", "tag:collects", "tag:comments"} {
		s.rdb.Del(ctx, h)
	}

	log.Printf("[agent] 标签分析持久化完成 tags=%d date=%s", len(allTags), date)
}

// ──────────── gRPC：粉丝增长 ────────────

func (s *Server) GetNoteFansGrowth(ctx context.Context, req *agentpb.GetNoteFansGrowthReq) (*agentpb.GetNoteFansGrowthResp, error) {
	now := time.Now()

	// 1. 查 MySQL 历史
	var records []NoteFansGrowth
	s.db.WithContext(ctx).
		Where("note_id = ? AND hour >= ?", req.NoteId, now.Add(-168*time.Hour).Format("2006-01-02T15")).
		Order("hour ASC").
		Find(&records)

	points := map[string]int64{}
	for _, r := range records {
		points[r.Hour] += r.Count
	}

	// 2. 补充 Redis 中尚未持久化的实时数据
	keys, _ := s.rdb.Keys(ctx, fmt.Sprintf("fans_growth:%d:*", req.NoteId)).Result()
	for _, k := range keys {
		hour := k[strings.LastIndex(k, ":")+1:]
		v, _ := s.rdb.Get(ctx, k).Int64()
		points[hour] += v
	}

	// 3. 排序
	var sorted []string
	for h := range points {
		sorted = append(sorted, h)
	}
	sort.Strings(sorted)

	result := make([]*agentpb.FansGrowthPoint, 0, len(sorted))
	for _, h := range sorted {
		result = append(result, &agentpb.FansGrowthPoint{Hour: h, Count: points[h]})
	}

	return &agentpb.GetNoteFansGrowthResp{Points: result}, nil
}

// ──────────── gRPC：标签分析 ────────────

func (s *Server) GetTagAnalytics(ctx context.Context, req *agentpb.GetTagAnalyticsReq) (*agentpb.GetTagAnalyticsResp, error) {
	// 实时查询：Redis 最新 + MySQL 历史合并
	nc, _ := s.rdb.HGet(ctx, "tag:note_count", req.Tag).Int64()
	l, _ := s.rdb.HGet(ctx, "tag:likes", req.Tag).Int64()
	c, _ := s.rdb.HGet(ctx, "tag:collects", req.Tag).Int64()
	cm, _ := s.rdb.HGet(ctx, "tag:comments", req.Tag).Int64()

	// 补充 MySQL 历史
	var hist TagAnalytics
	s.db.WithContext(ctx).Where("tag = ?", req.Tag).Order("date DESC").First(&hist)
	nc += hist.NoteCount
	l += hist.TotalLikes
	c += hist.TotalCollects
	cm += hist.TotalComments

	totalInteractions := l + c + cm
	var rate float64
	if nc > 0 {
		rate = float64(totalInteractions) / float64(nc)
	}

	return &agentpb.GetTagAnalyticsResp{
		Tag: req.Tag, NoteCount: nc,
		TotalLikes: l, TotalCollects: c, TotalComments: cm,
		EngagementRate: rate,
	}, nil
}

func (s *Server) GetTopTags(ctx context.Context, req *agentpb.GetTopTagsReq) (*agentpb.GetTopTagsResp, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// 聚合 Redis 当前 + MySQL 历史
	tags, _ := s.rdb.HGetAll(ctx, "tag:note_count").Result()

	type tagStat struct {
		tag       string
		noteCount int64
		likes     int64
		collects  int64
		comments  int64
		rate      float64
	}

	stats := map[string]*tagStat{}
	for t, v := range tags {
		nc, _ := strconv.ParseInt(v, 10, 64)
		stats[t] = &tagStat{tag: t, noteCount: nc}
	}
	for _, h := range []string{"tag:likes", "tag:collects", "tag:comments"} {
		m, _ := s.rdb.HGetAll(ctx, h).Result()
		for t, v := range m {
			n, _ := strconv.ParseInt(v, 10, 64)
			if stats[t] == nil {
				stats[t] = &tagStat{tag: t}
			}
			switch h {
			case "tag:likes":
				stats[t].likes = n
			case "tag:collects":
				stats[t].collects = n
			case "tag:comments":
				stats[t].comments = n
			}
		}
	}

	var list []*tagStat
	for _, st := range stats {
		// 合并历史
		var hist TagAnalytics
		if err := s.db.First(&hist, "tag = ?", st.tag); err == nil {
			st.noteCount += hist.NoteCount
			st.likes += hist.TotalLikes
			st.collects += hist.TotalCollects
			st.comments += hist.TotalComments
		}
		total := st.likes + st.collects + st.comments
		if st.noteCount > 0 {
			st.rate = float64(total) / float64(st.noteCount)
		}
		list = append(list, st)
	}

	// 按互动率排序
	sort.Slice(list, func(i, j int) bool { return list[i].rate > list[j].rate })

	if int32(len(list)) > limit {
		list = list[:limit]
	}

	tagList := make([]*agentpb.TagSummary, len(list))
	for i, st := range list {
		tagList[i] = &agentpb.TagSummary{
			Tag: st.tag, NoteCount: st.noteCount, EngagementRate: st.rate,
		}
	}

	return &agentpb.GetTopTagsResp{Tags: tagList}, nil
}

// ──────────── gRPC：运营建议 ────────────

func (s *Server) GetSuggestions(ctx context.Context, req *agentpb.GetSuggestionsReq) (*agentpb.GetSuggestionsResp, error) {
	// 1. 收集用户数据
	stats := s.collectUserStats(ctx, req.UserId)

	// 2. 尝试 LLM
	if s.llm != nil && s.llm.Enabled() {
		suggestions, err := s.llmSuggestions(ctx, req.UserId, stats)
		if err == nil && len(suggestions) > 0 {
			return &agentpb.GetSuggestionsResp{Suggestions: suggestions}, nil
		}
		log.Printf("[agent] LLM 调用失败，回退规则引擎: %v", err)
	}

	// 3. 规则引擎兜底
	tips := s.ruleSuggestions(stats)
	return &agentpb.GetSuggestionsResp{Suggestions: tips}, nil
}

// userStats 聚合用户的关键运营指标。
type userStats struct {
	TotalNotes   int
	AvgImages    float64
	UsedTags     []string
	TopTag       string
	HasUnusedHot bool
	HotTag       string
	HotTagRate   float64
}

func (s *Server) collectUserStats(ctx context.Context, userID int64) userStats {
	rows, err := s.db.WithContext(ctx).Raw(
		"SELECT id, image_urls, tags FROM notes WHERE user_id = ? ORDER BY created_at DESC LIMIT 20",
		userID,
	).Rows()
	if err != nil {
		return userStats{}
	}
	defer rows.Close()

	var st userStats
	tagFreq := map[string]int{}
	totalImages := 0
	for rows.Next() {
		var id int64
		var imgs, ts string
		rows.Scan(&id, &imgs, &ts)
		st.TotalNotes++
		if imgs != "" {
			totalImages += len(strings.Split(imgs, ","))
		}
		for _, t := range strings.Split(ts, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagFreq[t]++
				st.UsedTags = append(st.UsedTags, t)
			}
		}
	}
	if st.TotalNotes > 0 {
		st.AvgImages = float64(totalImages) / float64(st.TotalNotes)
	}

	// 最常用 tag
	maxFreq := 0
	for t, f := range tagFreq {
		if f > maxFreq {
			maxFreq = f
			st.TopTag = t
		}
	}

	// 是否有未使用的热门标签
	topTags, _ := s.GetTopTags(ctx, &agentpb.GetTopTagsReq{Limit: 5})
	for _, t := range topTags.Tags {
		if tagFreq[t.Tag] == 0 {
			st.HasUnusedHot = true
			st.HotTag = t.Tag
			st.HotTagRate = t.EngagementRate
			break
		}
	}

	return st
}

// llmSuggestions 调用大模型生成个性化建议。
func (s *Server) llmSuggestions(ctx context.Context, userID int64, st userStats) ([]string, error) {
	tagUsage := strings.Join(st.UsedTags, "、")
	if tagUsage == "" {
		tagUsage = "无"
	}

	prompt := fmt.Sprintf(`请分析以下用户数据，给出 3-5 条运营优化建议：

用户 ID: %d
最近 20 篇笔记数: %d
平均每篇图片数: %.1f（建议≥3）
最常用标签: %s
使用过的标签: %s`,
		userID, st.TotalNotes, st.AvgImages, st.TopTag, tagUsage,
	)

	if st.HasUnusedHot {
		prompt += fmt.Sprintf("\n当前热门标签: %s（互动率 %.1f），但该用户未使用过", st.HotTag, st.HotTagRate)
	}

	prompt += "\n\n请直接给出建议，每条以emoji开头，不要编号，不要markdown。"

	result, err := s.llm.ChatCompletion(prompt)
	if err != nil {
		return nil, err
	}

	// 按行拆分，保留非空行
	lines := strings.Split(strings.TrimSpace(result), "\n")
	var suggestions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) > 3 {
			suggestions = append(suggestions, line)
		}
	}
	if len(suggestions) > 8 {
		suggestions = suggestions[:8]
	}
	return suggestions, nil
}

// ruleSuggestions 规则引擎兜底（旧逻辑）。
func (s *Server) ruleSuggestions(st userStats) []string {
	var tips []string

	if st.TotalNotes > 0 && st.AvgImages < 3 {
		tips = append(tips, "📷 你的笔记图片较少，添加 3 张以上图片可以提高曝光率")
	}

	if st.HasUnusedHot {
		tips = append(tips, fmt.Sprintf("🏷️ 当前热门标签「%s」（互动率 %.1f），建议在你的笔记中使用", st.HotTag, st.HotTagRate))
	}

	tips = append(tips, "⏰ 数据显示晚间 20:00-22:00 是用户活跃高峰，建议在此时间段发布")

	if st.TotalNotes < 5 {
		tips = append(tips, "📝 你最近发布较少，保持每周 2-3 篇的发布频率有助于提升影响力")
	}

	tips = append(tips, "💬 数据表明评论会显著提升推荐权重，尝试在笔记结尾引导用户评论互动")

	if len(tips) == 0 {
		tips = append(tips, "🎉 你的运营策略看起来很健康，继续保持！")
	}
	if len(tips) > 8 {
		tips = tips[:8]
	}
	return tips
}

// ──────────── gRPC 客户端 ────────────

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

func OpenRedis() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 2})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	log.Println("Redis（agent）连接成功")
	return rdb, nil
}
