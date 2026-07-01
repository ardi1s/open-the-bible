// Package feed —— RabbitMQ 消费者。
// 消费 note.created 事件，触发 Feed 推模式分发。
package feed

import (
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

// StartConsumer 连接到 RabbitMQ，声明队列并绑定 note.events / note.created。
// 收到消息后异步回调 handler（不阻塞主协程）。
// 该函数阻塞当前协程，在 channel 关闭前持续消费。
func StartConsumer(handler func([]byte)) error {
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

	// 声明 exchange（幂等，已存在则跳过）
	err = ch.ExchangeDeclare("note.events", "topic", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 声明队列（持久化）
	q, err := ch.QueueDeclare("feed_queue", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 绑定 routing key
	err = ch.QueueBind(q.Name, "note.created", "note.events", false, nil)
	if err != nil {
		return err
	}

	// 设置 prefetch，每次只取一条
	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("[feed-consumer] 已绑定 feed_queue → note.events / note.created，开始消费")

	// 同步消费，每条 ack
	go func() {
		for d := range msgs {
			handler(d.Body)
			d.Ack(false)
			time.Sleep(10 * time.Millisecond) // 微小间隔，避免瞬间打爆下游
		}
		log.Println("[feed-consumer] channel 关闭，消费停止")
	}()

	// 阻塞，等待连接关闭
	closeErr := make(chan *amqp.Error)
	conn.NotifyClose(closeErr)
	err = <-closeErr
	log.Printf("[feed-consumer] RabbitMQ 连接关闭: %v", err)
	return nil
}
