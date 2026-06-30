// 用户服务（User Service）入口。
// 启动一个 gRPC 服务器监听 :50051，通过 GORM 连接 MySQL，连接 RabbitMQ 发布事件。
// 首次启动自动建表（GORM AutoMigrate）。
// 开发阶段运行：make run-user
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"xys-clone/pkg/db"
	"xys-clone/pkg/mq"
	userpb "xys-clone/proto/user"
	usersvc "xys-clone/services/user"
)

func main() {
	// ── 连接 MySQL（GORM + 重试）──
	gormDB, err := db.OpenGORM()
	if err != nil {
		log.Printf("[warn] MySQL 连接失败（将以 mock 模式运行）: %v", err)
		gormDB = nil
	}

	// ── 连接 RabbitMQ（带重连）──
	mqPublisher, err := mq.NewPublisher([]string{"follow.events"})
	if err != nil {
		log.Printf("[warn] RabbitMQ 连接失败（事件通知将跳过）: %v", err)
	}

	// ── 创建服务实例 & 自动建表 ──
	svc := usersvc.NewServer(gormDB, mqPublisher)
	if gormDB != nil {
		if err := svc.AutoMigrate(); err != nil {
			log.Fatalf("AutoMigrate 失败: %v", err)
		}
	}

	// ── 启动 gRPC ──
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, svc)
	reflection.Register(s)

	log.Println("UserService 已启动，监听 :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
