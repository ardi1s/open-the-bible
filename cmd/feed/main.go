// Feed 服务入口。
// 启动 gRPC 服务器监听 :50053，连接 MySQL + Redis + RabbitMQ（消费者）。
// 提供 GetUserFeed 推拉结合接口。
// 开发阶段运行：make run-feed
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"xys-clone/pkg/db"
	feedpb "xys-clone/proto/feed"
	feedsvc "xys-clone/services/feed"
)

func main() {
	// ── MySQL ──
	gormDB, err := db.OpenGORM()
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	// ── Redis ──
	rdb, err := feedsvc.OpenRedis()
	if err != nil {
		log.Fatalf("Redis 初始化失败: %v", err)
	}

	// ── gRPC 客户端（调用 user / note）──
	userClient, userConn, err := feedsvc.DialUser()
	if err != nil {
		log.Fatalf("连接 UserService 失败: %v", err)
	}
	defer userConn.Close()

	noteClient, noteConn, err := feedsvc.DialNote()
	if err != nil {
		log.Fatalf("连接 NoteService 失败: %v", err)
	}
	defer noteConn.Close()

	// ── 创建服务实例 ──
	svc := feedsvc.NewServer(gormDB, rdb, userClient, noteClient)

	// ── 启动 RabbitMQ 消费者（非阻塞 goroutine）──
	go func() {
		if err := feedsvc.StartConsumer(svc.HandleNoteCreated); err != nil {
			log.Printf("[warn] RabbitMQ 消费者启动失败（feed 仍可查询）: %v", err)
		}
	}()

	// ── 启动 gRPC ──
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	s := grpc.NewServer()
	feedpb.RegisterFeedServiceServer(s, svc)
	reflection.Register(s)

	log.Println("FeedService 已启动，监听 :50053")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
