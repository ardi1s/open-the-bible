-- ────────────────────────────────────────────
-- 数据库初始化脚本
--   Docker Compose 首次启动时自动执行（挂载到 /docker-entrypoint-initdb.d/）
--   本地开发可直接运行：mysql -u root -p < db/init.sql
--
--   注意：GORM AutoMigrate 会自动建表，此脚本主要用于：
--   1. 创建数据库本身（GORM 不会建库）
--   2. 作为表结构的文档参考
-- ────────────────────────────────────────────

CREATE DATABASE IF NOT EXISTS xhs
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;
