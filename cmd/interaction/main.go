// 互动服务（Interaction Service）入口。
// 启动一个 gRPC 服务器监听 :50054，连接 MySQL 与 RabbitMQ。
// 提供点赞、收藏、评论及互动汇总查询。
// 开发阶段运行：make run-interaction
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	interactionpb "xys-clone/proto/interaction"
	"xys-clone/pkg/db"
	"xys-clone/pkg/mq"
	interactionsvc "xys-clone/services/interaction"
)

func main() {
	// ── MySQL ──
	gormDB, err := db.OpenGORM()
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	// ── RabbitMQ ──
	mqPublisher, err := mq.NewPublisher([]string{"interaction.events"})
	if err != nil {
		log.Printf("[warn] RabbitMQ 连接失败（事件通知将跳过）: %v", err)
	}

	// ── gRPC 客户端 ──
	userClient, userConn, err := interactionsvc.DialUser()
	if err != nil {
		log.Fatalf("连接 UserService 失败: %v", err)
	}
	defer userConn.Close()

	// ── 创建服务 & 自动建表 ──
	svc := interactionsvc.NewServer(gormDB, mqPublisher, userClient)
	if err := svc.AutoMigrate(); err != nil {
		log.Fatalf("AutoMigrate 失败: %v", err)
	}

	// ── gRPC ──
	lis, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	s := grpc.NewServer()
	interactionpb.RegisterInteractionServiceServer(s, svc)
	reflection.Register(s)

	log.Println("InteractionService 已启动，监听 :50054")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
