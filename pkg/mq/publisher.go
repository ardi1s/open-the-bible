// Package mq 提供 RabbitMQ 事件发布器。
// 各微服务通过 NewPublisher 获取带重连守护的发布实例。
package mq

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/streadway/amqp"
)

// Publisher 封装 RabbitMQ 连接与发布逻辑。
type Publisher struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	exchanges []string // 需声明的 exchange 列表
	done      chan struct{}
}

// NewPublisher 创建 RabbitMQ 发布器，并启动后台重连守护。
// exchanges: 需要声明的 exchange 名称列表（topic 类型，持久化）。
//
// 环境变量 RABBITMQ_URL 指定连接地址，默认 amqp://guest:guest@127.0.0.1:5672/
func NewPublisher(exchanges []string) (*Publisher, error) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@127.0.0.1:5672/"
	}

	p := &Publisher{exchanges: exchanges, done: make(chan struct{})}

	if err := p.connect(url); err != nil {
		return nil, err
	}

	go p.reconnectLoop(url)
	return p, nil
}

// connect 建立连接、创建通道并声明所有 exchange。
func (p *Publisher) connect(url string) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("amqp.Dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("Channel: %w", err)
	}

	for _, ex := range p.exchanges {
		err = ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil)
		if err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("ExchangeDeclare(%s): %w", ex, err)
		}
		log.Printf("[mq] exchange 已声明: %s", ex)
	}

	p.conn = conn
	p.channel = ch
	log.Println("RabbitMQ 连接成功")
	return nil
}

// Publish 发送一条消息到指定 exchange / routing key。
func (p *Publisher) Publish(exchange, routingKey string, body []byte) error {
	if p.channel == nil {
		return fmt.Errorf("RabbitMQ 通道未就绪")
	}

	return p.channel.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

// reconnectLoop 监听连接关闭通知，自动重建连接。
func (p *Publisher) reconnectLoop(url string) {
	notifyClose := make(chan *amqp.Error)
	p.conn.NotifyClose(notifyClose)

	for {
		select {
		case err, ok := <-notifyClose:
			if !ok {
				return
			}
			log.Printf("[mq] 连接断开: %v，开始重连…", err)
		case <-p.done:
			return
		}

		for i := range math.MaxInt {
			if err := p.connect(url); err == nil {
				notifyClose = make(chan *amqp.Error)
				p.conn.NotifyClose(notifyClose)
				log.Println("[mq] 重连成功")
				break
			}
			backoff := time.Duration(math.Min(float64(i+1)*float64(time.Second), 10*float64(time.Second)))
			log.Printf("[mq] 重连失败，%v 后重试…", backoff)
			time.Sleep(backoff)
		}
	}
}

// Close 优雅关闭连接。
func (p *Publisher) Close() {
	close(p.done)
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
