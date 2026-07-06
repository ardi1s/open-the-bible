// Package agent —— RabbitMQ 延迟发布。
// 使用 AMQP TTL（per-message）+ 死信队列实现延迟投递，无需安装额外插件。
package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

const (
	liveEx = "agent.live"
	liveQ  = "agent_task_queue"
	delayQ = "delay_queue"
)

var (
	mqConn    *amqp.Connection
	mqCh      *amqp.Channel
	mqMu      sync.Mutex
	mqReady   bool
)

// InitTaskMQ 初始化 RabbitMQ 连接（常驻），同时声明 liveQ 和 delayQ。
func InitTaskMQ() error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@127.0.0.1:5672/"
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("channel: %w", err)
	}

	// 声明 exchange
	ch.ExchangeDeclare(liveEx, "direct", true, false, false, false, nil)

	// 声明接收队列（被消费者绑定）
	ch.QueueDeclare(liveQ, true, false, false, false, nil)
	ch.QueueBind(liveQ, "task.execute", liveEx, false, nil)

	// 声明延迟队列（设死信路由，但不设 queue 级 TTL——TTL 在消息级控制）
	args := amqp.Table{
		"x-dead-letter-exchange":    liveEx,
		"x-dead-letter-routing-key": "task.execute",
	}
	if _, err := ch.QueueDeclare(delayQ, true, false, false, false, args); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("declare delay_queue: %w", err)
	}

	mqConn = conn
	mqCh = ch
	mqReady = true
	log.Println("[schedule] RabbitMQ 任务队列已就绪")
	return nil
}

// publishDelayedTask 发送一条带单消息 TTL 的延迟消息到 delay_queue。
func publishDelayedTask(taskID int64, delaySeconds int64) error {
	mqMu.Lock()
	ready := mqReady
	mqMu.Unlock()

	if !ready {
		return fmt.Errorf("MQ 未就绪")
	}

	body, _ := json.Marshal(map[string]int64{"task_id": taskID})
	expiration := strconv.FormatInt(delaySeconds*1000, 10)

	mqMu.Lock()
	defer mqMu.Unlock()

	err := mqCh.Publish("", delayQ, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Expiration:   expiration, // per-message TTL（毫秒）
			Body:         body,
		})
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	log.Printf("[schedule] 延迟消息已发送 task=%d delay=%ds", taskID, delaySeconds)
	return nil
}

// StartTaskConsumer 消费 agent_task_queue，每条消息回调 executor(taskID)。
func StartTaskConsumer(executor func(taskID int64)) error {
	mqMu.Lock()
	if !mqReady {
		mqMu.Unlock()
		return fmt.Errorf("MQ 未就绪，无法启动消费者")
	}
	mqMu.Unlock()

	mqMu.Lock()
	ch := mqCh
	mqMu.Unlock()

	ch.Qos(1, 0, false)
	msgs, err := ch.Consume(liveQ, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Println("[schedule-consumer] 开始消费 agent_task_queue")

	go func() {
		for d := range msgs {
			var m struct{ TaskID int64 `json:"task_id"` }
			if err := json.Unmarshal(d.Body, &m); err == nil && m.TaskID > 0 {
				log.Printf("[schedule] 收到延迟任务 task=%d", m.TaskID)
				executor(m.TaskID)
			}
			d.Ack(false)
			time.Sleep(100 * time.Millisecond)
		}
		log.Println("[schedule-consumer] channel 关闭")
	}()

	closeErr := make(chan *amqp.Error)
	mqConn.NotifyClose(closeErr)
	err = <-closeErr
	log.Printf("[schedule-consumer] 连接关闭: %v", err)
	return nil
}
