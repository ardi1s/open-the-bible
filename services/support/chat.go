// Package support 实现智能客服逻辑。
// FAQ 关键词匹配 + LLM 兜底 + Redis 会话上下文 + 排行榜整合。
package support

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	rankpb "xys-clone/proto/rank"
	"xys-clone/pkg/llm"
)

// ChatHandler 封装客服处理的全部依赖。
type ChatHandler struct {
	rdb        *redis.Client
	rankClient rankpb.RankServiceClient
	llm        *llm.Client
}

// NewChatHandler 创建客服处理器。
func NewChatHandler(rdb *redis.Client, rankClient rankpb.RankServiceClient, llmClient *llm.Client) *ChatHandler {
	return &ChatHandler{rdb: rdb, rankClient: rankClient, llm: llmClient}
}

// ──────────── 会话 ────────────

type chatMsg struct {
	Role    string `json:"role"`    // "user" / "assistant"
	Content string `json:"content"`
	Time    int64  `json:"time"`
}

func (h *ChatHandler) loadHistory(sessionID string) []chatMsg {
	key := "chat:session:" + sessionID
	data, err := h.rdb.Get(context.Background(), key).Result()
	if err != nil {
		return nil
	}
	var msgs []chatMsg
	json.Unmarshal([]byte(data), &msgs)
	return msgs
}

func (h *ChatHandler) saveHistory(sessionID string, msgs []chatMsg) {
	if len(msgs) > 10 {
		msgs = msgs[len(msgs)-10:]
	}
	data, _ := json.Marshal(msgs)
	h.rdb.Set(context.Background(), "chat:session:"+sessionID, data, time.Hour)
}

// ──────────── 对话入口 ────────────

// Reply 根据用户输入生成回复。
func (h *ChatHandler) Reply(sessionID, userMsg string) string {
	history := h.loadHistory(sessionID)
	history = append(history, chatMsg{Role: "user", Content: userMsg, Time: time.Now().Unix()})

	var reply string
	if r := h.faqMatch(userMsg); r != "" {
		reply = r
	} else {
		reply = h.llmReply(history)
	}

	history = append(history, chatMsg{Role: "assistant", Content: reply, Time: time.Now().Unix()})
	h.saveHistory(sessionID, history)
	return reply
}

// ──────────── FAQ ────────────

func (h *ChatHandler) faqMatch(msg string) string {
	msgLower := strings.ToLower(msg)

	type match struct {
		count int
		h     func() string
	}

	faqs := []struct {
		keywords []string
		handler  func() string
	}{
		{[]string{"热门", "排行", "热榜", "top", "hot"}, h.faqHotNotes},
		{[]string{"发布", "publish", "怎么发"}, h.faqPublish},
		{[]string{"关注", "粉丝", "follow"}, h.faqFollow},
		{[]string{"点赞", "like", "收藏"}, h.faqInteraction},
		{[]string{"标签", "tag"}, h.faqTag},
		{[]string{"图片", "照片", "相机", "拍照"}, h.faqImage},
		{[]string{"你好", "hello", "hi", "在吗"}, h.faqGreeting},
	}

	var matches []match
	for _, f := range faqs {
		c := 0
		for _, kw := range f.keywords {
			if strings.Contains(msgLower, kw) {
				c++
			}
		}
		if c > 0 {
			matches = append(matches, match{c, f.handler})
		}
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].count > matches[j].count })
	return matches[0].h()
}

func (h *ChatHandler) faqHotNotes() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := h.rankClient.GetHotNotes(ctx, &rankpb.GetHotNotesReq{Count: 5})
	if err != nil {
		return "当前排行榜服务暂不可用，请稍后重试 😅"
	}
	if len(resp.Items) == 0 {
		return "目前还没有热门笔记，快去发布第一篇吧！🎉"
	}

	var sb strings.Builder
	sb.WriteString("📊 当前热门笔记 Top 5：\n\n")
	for i, item := range resp.Items {
		title := item.Title
		runes := []rune(title)
		if len(runes) > 20 {
			title = string(runes[:20]) + "..."
		}
		fmt.Fprintf(&sb, "%d. %s\n   👤 %s | 🔥 %.2f\n", i+1, title, item.AuthorName, item.HeatScore)
	}
	return strings.TrimSpace(sb.String())
}

func (h *ChatHandler) faqPublish() string {
	return "📝 发布笔记的方法：\n1. 点击首页\"+\"按钮\n2. 选择图片（建议 3 张以上）\n3. 写标题和正文\n4. 添加相关标签\n5. 点击发布\n\n💡 小贴士：晚上 20:00-22:00 发布更容易获得流量哦～"
}

func (h *ChatHandler) faqFollow() string {
	return "👥 关于关注和粉丝：\n- 在笔记页点击作者头像 → \"关注\"即可关注\n- 你的粉丝在\"我的\"→\"粉丝\"中查看\n- 关注后可以在 Feed 信息流看到对方的新笔记\n- 好的内容更容易吸引粉丝哦～"
}

func (h *ChatHandler) faqInteraction() string {
	return "❤️ 互动说明：\n- 双击或点击 ❤️ 点赞笔记\n- 点击 🔖 收藏笔记\n- 在笔记底部发表评论\n- 点赞+收藏+评论会影响笔记的推荐权重 🚀"
}

func (h *ChatHandler) faqTag() string {
	return "🏷️ 标签使用建议：\n- 每篇笔记建议 2-5 个标签\n- 热门标签能带来更多曝光\n- 可以使用具体标签（如 #通勤穿搭）\n- 查看热门标签：GET /api/agent/top-tags"
}

func (h *ChatHandler) faqImage() string {
	return "📷 图片建议：\n- 推荐上传 3-9 张图片\n- 清晰度高、构图精美的图片更容易被推荐\n- 支持 jpg / png / webp 格式"
}

func (h *ChatHandler) faqGreeting() string {
	return "👋 你好！我是小 X 智能客服～\n\n你可以问我：\n• 热门笔记有哪些？\n• 怎么发布笔记？\n• 如何获得更多粉丝？\n• 标签怎么选？\n\n直接打字提问，我随时在线！"
}

// ──────────── LLM 兜底 ────────────

func (h *ChatHandler) llmReply(history []chatMsg) string {
	if h.llm == nil || !h.llm.Enabled() {
		return "抱歉，我暂时无法回答这个问题。你可以尝试问我：热门笔记、发布方法、标签建议等 😊"
	}

	var b strings.Builder
	for _, m := range history {
		fmt.Fprintf(&b, "%s: %s\n", m.Role, m.Content)
	}
	b.WriteString("\n请直接回答：")

	reply, err := h.llm.ChatCompletion(b.String())
	if err != nil {
		log.Printf("[support] LLM 调用失败: %v", err)
		return "抱歉，我暂时无法回答这个问题 😅"
	}
	return strings.TrimSpace(reply)
}

// ──────────── Redis ────────────

// OpenRedis 连接 Redis（DB 3，避免和其他服务冲突）。
func OpenRedis() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 3})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	log.Println("Redis（support）连接成功")
	return rdb, nil
}
