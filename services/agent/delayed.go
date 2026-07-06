// Package agent —— RabbitMQ 延迟发布。
// 使用 AMQP TTL + 死信队列实现延迟投递（无需安装额外插件）。
package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

const (
	liveEx  = "agent.live"
	liveQ   = "agent_task_queue"
	delayQ  = "delay_queue"
)

// publishDelayedTask 发送一条带 TTL 的消息到 delay_queue，TTL 到期后由死信路由到 agent_task_queue。
func publishDelayedTask(taskID int64, delaySeconds int64) error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@127.0.0.1:5672/"
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}
	defer ch.Close()

	// 声明接收队列
	ch.ExchangeDeclare(liveEx, "direct", true, false, false, false, nil)
	q, _ := ch.QueueDeclare(liveQ, true, false, false, false, nil)
	ch.QueueBind(q.Name, "task.execute", liveEx, false, nil)

	// 声明延迟队列（TTL + 死信路由到 liveEx）
	args := amqp.Table{
		"x-dead-letter-exchange":    liveEx,
		"x-dead-letter-routing-key": "task.execute",
		"x-message-ttl":             int32(delaySeconds * 1000),
	}
	ch.QueueDeclare(delayQ, true, false, false, false, args)

	body, _ := json.Marshal(map[string]int64{"task_id": taskID})
	err = ch.Publish("", delayQ, false, false,
		amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: body})
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	log.Printf("[schedule] 延迟消息已发送 task=%d delay=%ds", taskID, delaySeconds)
	return nil
}

// StartTaskConsumer 消费 agent_task_queue，每条消息回调 executor(taskID)。
func StartTaskConsumer(executor func(taskID int64)) error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@127.0.0.1:5672/"
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	ch.ExchangeDeclare(liveEx, "direct", true, false, false, false, nil)
	q, _ := ch.QueueDeclare(liveQ, true, false, false, false, nil)
	ch.QueueBind(q.Name, "task.execute", liveEx, false, nil)

	args := amqp.Table{
		"x-dead-letter-exchange":    liveEx,
		"x-dead-letter-routing-key": "task.execute",
	}
	ch.QueueDeclare(delayQ, true, false, false, false, args)

	ch.Qos(1, 0, false)
	msgs, err := ch.Consume(liveQ, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("[schedule-consumer] agent_task_queue 已就绪")

	go func() {
		for d := range msgs {
			var m struct{ TaskID int64 `json:"task_id"` }
			if err := json.Unmarshal(d.Body, &m); err == nil && m.TaskID > 0 {
				log.Printf("[schedule] 收到任务 task=%d", m.TaskID)
				executor(m.TaskID)
			}
			d.Ack(false)
		}
	}()

	closeErr := make(chan *amqp.Error)
	conn.NotifyClose(closeErr)
	<-closeErr
	log.Printf("[schedule-consumer] 连接关闭")
	return nil
}
