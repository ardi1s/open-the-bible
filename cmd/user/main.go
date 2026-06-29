// 用户服务（User Service）入口。
// 启动一个 gRPC 服务器监听 :50051，提供用户查询等接口。
// 开发阶段可直接运行：make run-user
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	userpb "xys-clone/proto/user"
	usersvc "xys-clone/services/user"
)

func main() {
	// 监听 TCP 端口 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	// 创建 gRPC 服务器
	s := grpc.NewServer()

	// 注册 UserService 实现
	userpb.RegisterUserServiceServer(s, usersvc.NewServer())

	// 开启 gRPC Reflection，方便使用 grpcurl 等工具调试（无需 .proto 文件）
	reflection.Register(s)

	log.Println("UserService 已启动，监听 :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
