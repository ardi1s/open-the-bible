// Package interaction —— 互动领域模型定义。
package interaction

// Like GORM 模型，对应 MySQL likes 表。
type Like struct {
	ID        int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64 `gorm:"column:user_id;uniqueIndex:uk_user_note;not null" json:"user_id"`
	NoteID    int64 `gorm:"column:note_id;uniqueIndex:uk_user_note;not null" json:"note_id"`
	CreatedAt int64 `gorm:"column:created_at;not null" json:"created_at"`
}

func (Like) TableName() string { return "likes" }

// Collect GORM 模型，对应 MySQL collects 表。
type Collect struct {
	ID        int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64 `gorm:"column:user_id;uniqueIndex:uk_user_note_collect;not null" json:"user_id"`
	NoteID    int64 `gorm:"column:note_id;uniqueIndex:uk_user_note_collect;not null" json:"note_id"`
	CreatedAt int64 `gorm:"column:created_at;not null" json:"created_at"`
}

func (Collect) TableName() string { return "collects" }

// Comment GORM 模型，对应 MySQL comments 表。
type Comment struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64  `gorm:"column:user_id;index;not null"  json:"user_id"`
	NoteID    int64  `gorm:"column:note_id;index;not null"  json:"note_id"`
	Content   string `gorm:"column:content;type:text;not null" json:"content"`
	ParentID  int64  `gorm:"column:parent_id;default:0"     json:"parent_id"`
	CreatedAt int64  `gorm:"column:created_at;not null"     json:"created_at"`
}

func (Comment) TableName() string { return "comments" }
