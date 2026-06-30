// Package note —— 笔记领域模型定义。
package note

// Note 笔记 GORM 模型，对应 MySQL notes 表。
// image_urls 与 tags 在库中存为逗号分隔字符串，入出参时由 service 层做 []string 互转。
type Note struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"                             json:"id"`
	UserID    int64  `gorm:"column:user_id;index;not null"                        json:"user_id"`
	Title     string `gorm:"column:title;type:varchar(256);not null;default:''"   json:"title"`
	Content   string `gorm:"column:content;type:text;not null"                    json:"content"`
	ImageURLs string `gorm:"column:image_urls;type:text;not null"                 json:"image_urls"`
	Tags      string `gorm:"column:tags;type:varchar(512);not null;default:''"    json:"tags"`
	CreatedAt int64  `gorm:"column:created_at;not null;default:0"                 json:"created_at"`
}

// TableName 显式指定表名。
func (Note) TableName() string { return "notes" }
