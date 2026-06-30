// Package db 提供 GORM 数据库连接的公共工具。
// 各微服务通过 OpenGORM 获取带重试与连接池配置的 *gorm.DB 实例。
package db

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// OpenGORM 根据环境变量 MYSQL_DSN 创建 GORM 连接，带指数退避重试（最长 30s）。
// 连接后自动配置连接池参数。
func OpenGORM() (*gorm.DB, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/xhs?parseTime=true&charset=utf8mb4"
	}

	var db *gorm.DB
	var err error

	for i := range 30 {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Warn), // 生产环境可改为 Silent
		})
		if err == nil {
			sqlDB, _ := db.DB()
			sqlDB.SetMaxOpenConns(25)
			sqlDB.SetMaxIdleConns(5)
			sqlDB.SetConnMaxLifetime(5 * time.Minute)

			log.Println("MySQL（GORM）连接成功")
			return db, nil
		}
		backoff := time.Duration(math.Min(float64(i+1)*float64(time.Second), 5*float64(time.Second)))
		log.Printf("MySQL 连接失败，%v 后重试… (%v)", backoff, err)
		time.Sleep(backoff)
	}

	return nil, fmt.Errorf("MySQL 连接超时（30s）: %w", err)
}
