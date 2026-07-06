// Package agent —— 分析服务领域模型。
package agent

// NoteFansGrowth 每条小时级别的粉丝增长记录。
type NoteFansGrowth struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	NoteID   int64  `gorm:"column:note_id;index;not null"  json:"note_id"`
	Hour     string `gorm:"column:hour;type:varchar(13);index;not null" json:"hour"` // "2026-07-06T08"
	Count    int64  `gorm:"column:count;not null;default:0" json:"count"`
}

func (NoteFansGrowth) TableName() string { return "note_fans_growth" }

// TagAnalytics 标签级别的汇总指标，每天凌晨固化。
type TagAnalytics struct {
	ID             int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	Tag            string  `gorm:"column:tag;type:varchar(64);index;not null"   json:"tag"`
	NoteCount      int64   `gorm:"column:note_count;not null;default:0"         json:"note_count"`
	TotalLikes     int64   `gorm:"column:total_likes;not null;default:0"        json:"total_likes"`
	TotalCollects  int64   `gorm:"column:total_collects;not null;default:0"     json:"total_collects"`
	TotalComments  int64   `gorm:"column:total_comments;not null;default:0"     json:"total_comments"`
	EngagementRate float64 `gorm:"column:engagement_rate;not null;default:0"    json:"engagement_rate"`
	Date           string  `gorm:"column:date;type:varchar(10);index;not null"  json:"date"` // "2026-07-06"
}

func (TagAnalytics) TableName() string { return "tag_analytics" }

// AgentTask 定时任务模型，对应 MySQL agent_tasks 表。
type AgentTask struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64  `gorm:"column:user_id;index;not null"        json:"user_id"`
	TaskType     string `gorm:"column:task_type;type:varchar(32);not null"   json:"task_type"` // "scheduled_post"
	Status       string `gorm:"column:status;type:varchar(16);not null;default:'pending'" json:"status"` // pending / done / failed
	Title        string `gorm:"column:title;type:varchar(256)"       json:"title"`
	Content      string `gorm:"column:content;type:text"             json:"content"`
	ImageURLs    string `gorm:"column:image_urls;type:text"          json:"image_urls"`   // 逗号分隔
	Tags         string `gorm:"column:tags;type:varchar(512)"         json:"tags"`        // 逗号分隔
	ScheduleTime int64  `gorm:"column:schedule_time;not null"         json:"schedule_time"` // Unix 时间戳
	ExecutedAt   int64  `gorm:"column:executed_at;default:0"          json:"executed_at"`
	NoteID       int64  `gorm:"column:note_id;default:0"              json:"note_id"`     // 执行后回填发布的 note_id
	CreatedAt    int64  `gorm:"column:created_at;autoCreateTime:milli" json:"created_at"`
}

func (AgentTask) TableName() string { return "agent_tasks" }
