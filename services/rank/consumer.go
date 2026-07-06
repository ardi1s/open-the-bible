// Package rank —— RabbitMQ 多事件消费者。
// 消费 note.created / interaction.like / interaction.collect / interaction.comment
// 四个 routing key，更新排行榜 Redis 热度分。
package rank

import (
	"log"
	"os"

	"github.com/streadway/amqp"
)

// StartConsumer 连接 RabbitMQ，声明 rank_queue 并绑定 4 个 routing key。
// handler 在 goroutine 中回调，每条消息自动 Ack。
// 该函数阻塞当前协程直到连接断开。
func StartConsumer(handler func(routingKey string, body []byte)) error {
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

	// 声明两个 exchange（幂等）
	for _, ex := range []string{"note.events", "interaction.events"} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}

	// 声明队列
	q, err := ch.QueueDeclare("rank_queue", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 绑定 4 个 routing key
	bindings := []struct{ exchange, key string }{
		{"note.events", "note.created"},
		{"interaction.events", "interaction.like"},
		{"interaction.events", "interaction.collect"},
		{"interaction.events", "interaction.comment"},
	}
	for _, b := range bindings {
		if err := ch.QueueBind(q.Name, b.key, b.exchange, false, nil); err != nil {
			return err
		}
	}

	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("[rank-consumer] rank_queue 已绑定 4 个 routing key，开始消费")

	go func() {
		for d := range msgs {
			handler(d.RoutingKey, d.Body)
			d.Ack(false)
		}
		log.Println("[rank-consumer] channel 关闭")
	}()

	closeErr := make(chan *amqp.Error)
	conn.NotifyClose(closeErr)
	err = <-closeErr
	log.Printf("[rank-consumer] RabbitMQ 连接关闭: %v", err)
	return nil
}
