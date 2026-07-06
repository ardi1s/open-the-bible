// Package agent —— RabbitMQ 消费者。
// 消费 follow.add / note.created / interaction.* 事件，分别更新粉丝增长和标签统计。
package agent

import (
	"log"
	"os"

	"github.com/streadway/amqp"
)

// StartConsumer 连接 RabbitMQ，声明 agent_queue 并绑定 5 个 routing key。
// handler 在 goroutine 中回调，每条消息自动 Ack。
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

	// 需要 3 个 exchange
	for _, ex := range []string{"follow.events", "note.events", "interaction.events"} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}

	q, err := ch.QueueDeclare("agent_queue", true, false, false, false, nil)
	if err != nil {
		return err
	}

	bindings := []struct{ exchange, key string }{
		{"follow.events", "follow.add"},
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

	log.Println("[agent-consumer] agent_queue 已绑定 5 个 routing key，开始消费")

	go func() {
		for d := range msgs {
			handler(d.RoutingKey, d.Body)
			d.Ack(false)
		}
		log.Println("[agent-consumer] channel 关闭")
	}()

	closeErr := make(chan *amqp.Error)
	conn.NotifyClose(closeErr)
	err = <-closeErr
	log.Printf("[agent-consumer] RabbitMQ 连接关闭: %v", err)
	return nil
}
