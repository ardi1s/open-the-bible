// 用户服务（User Service）入口。
// 启动一个 gRPC 服务器监听 :50051，通过 GORM 连接 MySQL。
// get_user 优先查库，未命中时返回 mock 数据。
// 开发阶段可直接运行：make run-user
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"xys-clone/pkg/db"
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

	// ── 创建服务实例 & 自动建表 ──
	svc := usersvc.NewServer(gormDB)
	if gormDB != nil {
		if err := svc.AutoMigrate(); err != nil {
			log.Printf("[warn] AutoMigrate 失败: %v", err)
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
