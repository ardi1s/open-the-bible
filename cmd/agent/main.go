// Agent 分析服务入口。
// 启动 gRPC 服务器监听 :50056，连接 MySQL + Redis + RabbitMQ（消费者）。
// 提供粉丝增长、标签分析、运营建议等接口。
// 开发阶段运行：make run-agent
package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	agentpb "xys-clone/proto/agent"
	"xys-clone/pkg/db"
	"xys-clone/pkg/llm"
	agentsvc "xys-clone/services/agent"
)

func main() {
	gormDB, err := db.OpenGORM()
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	rdb, err := agentsvc.OpenRedis()
	if err != nil {
		log.Fatalf("Redis 初始化失败: %v", err)
	}

	noteClient, noteConn, err := agentsvc.DialNote()
	if err != nil {
		log.Fatalf("连接 NoteService 失败: %v", err)
	}
	defer noteConn.Close()

	userClient, userConn, err := agentsvc.DialUser()
	if err != nil {
		log.Fatalf("连接 UserService 失败: %v", err)
	}
	defer userConn.Close()

	llmClient := llm.New()
	if llmClient.Enabled() {
		log.Println("LLM（大模型）已配置，GetSuggestions 将使用 AI 生成建议")
	} else {
		log.Println("LLM_API_KEY 未配置，使用规则引擎生成建议")
	}

	svc := agentsvc.NewServer(rdb, gormDB, noteClient, userClient, llmClient)
	if err := svc.AutoMigrate(); err != nil {
		log.Fatalf("AutoMigrate 失败: %v", err)
	}

	go func() {
		if err := agentsvc.StartConsumer(svc.HandleEvent); err != nil {
			log.Printf("[warn] RabbitMQ 消费者启动失败: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", ":50056")
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	s := grpc.NewServer()
	agentpb.RegisterAgentServiceServer(s, svc)
	reflection.Register(s)

	log.Println("AgentService 已启动，监听 :50056")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
