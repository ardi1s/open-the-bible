// Package rank —— 排行榜领域模型。
package rank

// RankSnapshot GORM 模型，对应 MySQL rank_snapshots 表。
// 每小时将热门笔记 Top N 快照写入，方便历史追溯。
type RankSnapshot struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	NoteID    int64  `gorm:"column:note_id;index;not null"    json:"note_id"`
	Hour      string `gorm:"column:hour;type:varchar(13);index;not null" json:"hour"` // 格式 "2026-07-04T20"
	HeatScore float64 `gorm:"column:heat_score;not null"      json:"heat_score"`
	RankPos   int32  `gorm:"column:rank_pos;not null"         json:"rank_pos"`
}

func (RankSnapshot) TableName() string { return "rank_snapshots" }
