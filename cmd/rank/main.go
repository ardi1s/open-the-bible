// 排行榜服务（Rank Service）入口。
// 启动 gRPC 服务器监听 :50055，连接 MySQL + Redis + RabbitMQ（消费者）。
// 提供 GetHotNotes 接口，消费事件实时更新 Redis 热度分。
// 开发阶段运行：make run-rank
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"xys-clone/pkg/db"
	rankpb "xys-clone/proto/rank"
	ranksvc "xys-clone/services/rank"
)

func main() {
	// ── MySQL ──
	gormDB, err := db.OpenGORM()
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	// ── Redis ──
	rdb, err := ranksvc.OpenRedis()
	if err != nil {
		log.Fatalf("Redis 初始化失败: %v", err)
	}

	// ── gRPC 客户端 ──
	noteClient, noteConn, err := ranksvc.DialNote()
	if err != nil {
		log.Fatalf("连接 NoteService 失败: %v", err)
	}
	defer noteConn.Close()

	userClient, userConn, err := ranksvc.DialUser()
	if err != nil {
		log.Fatalf("连接 UserService 失败: %v", err)
	}
	defer userConn.Close()

	// ── 创建服务 ──
	svc := ranksvc.NewServer(rdb, gormDB, noteClient, userClient)
	if err := svc.AutoMigrate(); err != nil {
		log.Fatalf("AutoMigrate 失败: %v", err)
	}

	// ── 启动 RabbitMQ 消费者 ──
	go func() {
		if err := ranksvc.StartConsumer(svc.HandleEvent); err != nil {
			log.Printf("[warn] RabbitMQ 消费者启动失败（排行榜仍可查询）: %v", err)
		}
	}()

	// ── gRPC ──
	lis, err := net.Listen("tcp", ":50055")
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	s := grpc.NewServer()
	rankpb.RegisterRankServiceServer(s, svc)
	reflection.Register(s)

	log.Println("RankService 已启动，监听 :50055")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
