package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/1119-Labs/callisto/v4/lib/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
)

// RabbitMQHeightQueue implements HeightQueue using RabbitMQ.
type RabbitMQHeightQueue struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// ConnectRabbitMQ creates a new RabbitMQ connection and returns a HeightQueue.
func ConnectRabbitMQ(cfg config.RabbitMQConfig) (types.HeightQueue, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("rabbitmq url is empty")
	}
	if cfg.QueueName == "" {
		return nil, fmt.Errorf("rabbitmq queue_name is empty")
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open rabbitmq channel: %w", err)
	}

	_, err = ch.QueueDeclare(
		cfg.QueueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare rabbitmq queue: %w", err)
	}

	if cfg.Prefetch > 0 {
		if err := ch.Qos(cfg.Prefetch, 0, false); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, fmt.Errorf("failed to set rabbitmq qos: %w", err)
		}
	}

	return &RabbitMQHeightQueue{
		conn:      conn,
		channel:   ch,
		queueName: cfg.QueueName,
	}, nil
}

// Publish enqueues a block height.
func (q *RabbitMQHeightQueue) Publish(height int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := []byte(strconv.FormatInt(height, 10))
	return q.channel.PublishWithContext(
		ctx,
		"",          // exchange
		q.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         body,
		},
	)
}

// Consume consumes block heights and invokes the handler.
func (q *RabbitMQHeightQueue) Consume(handler func(height int64) error) error {
	deliveries, err := q.channel.Consume(
		q.queueName,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to start rabbitmq consumer: %w", err)
	}

	for delivery := range deliveries {
		height, err := strconv.ParseInt(string(delivery.Body), 10, 64)
		if err != nil {
			_ = delivery.Nack(false, false)
			continue
		}

		if err := handler(height); err != nil {
			_ = delivery.Nack(false, true)
			continue
		}

		_ = delivery.Ack(false)
	}

	return nil
}

// Close closes the queue resources.
func (q *RabbitMQHeightQueue) Close() error {
	if q == nil {
		return nil
	}
	if q.channel != nil {
		_ = q.channel.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}
