// Package user —— 用户领域模型定义。
package user

// User GORM 模型，对应 MySQL users 表。
type User struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"                           json:"id"`
	Username  string `gorm:"column:username;type:varchar(64);uniqueIndex;not null" json:"username"`
	Password  string `gorm:"column:password;type:varchar(255);not null;default:''" json:"-"`
	Bio       string `gorm:"column:bio;type:varchar(512);not null;default:''"    json:"bio"`
	Avatar    string `gorm:"column:avatar;type:varchar(512);not null;default:''" json:"avatar"`
	CreatedAt int64  `gorm:"column:created_at;not null;default:0"                json:"created_at"`
}

// TableName 显式指定表名。
func (User) TableName() string { return "users" }
