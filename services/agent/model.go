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
