package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/1119-Labs/callisto/v4/lib/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
)

// ---------------------------------------------------------------------------------------------------------------------
// RabbitMQHeightQueue - Block Height Queue
// ---------------------------------------------------------------------------------------------------------------------

// RabbitMQHeightQueue implements HeightQueue using RabbitMQ.
type RabbitMQHeightQueue struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// connectHeightQueue creates a new RabbitMQ connection for a block height queue.
func connectHeightQueue(cfg config.RabbitMQConfig, queueName string) (types.HeightQueue, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("rabbitmq url is empty")
	}
	if queueName == "" {
		return nil, fmt.Errorf("rabbitmq queue name is empty")
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
		queueName,
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
		queueName: queueName,
	}, nil
}

// ConnectNewBlockQueue creates a new RabbitMQ connection for the new block queue (high priority).
func ConnectNewBlockQueue(cfg config.RabbitMQConfig) (types.HeightQueue, error) {
	return connectHeightQueue(cfg, cfg.NewBlockQueueName)
}

// ConnectOldBlockQueue creates a new RabbitMQ connection for the old block queue (backfill).
func ConnectOldBlockQueue(cfg config.RabbitMQConfig) (types.HeightQueue, error) {
	return connectHeightQueue(cfg, cfg.OldBlockQueueName)
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

// ---------------------------------------------------------------------------------------------------------------------
// RabbitMQTxQueue - Transaction Hash Queue
// ---------------------------------------------------------------------------------------------------------------------

// TxMessage represents a message in the tx queue.
type TxMessage struct {
	TxHash string `json:"tx_hash"`
	Height int64  `json:"height"`
}

// RabbitMQTxQueue implements TxQueue using RabbitMQ.
type RabbitMQTxQueue struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// connectTxQueue creates a new RabbitMQ connection for a tx queue.
func connectTxQueue(cfg config.RabbitMQConfig, queueName string) (types.TxQueue, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("rabbitmq url is empty")
	}
	if queueName == "" {
		return nil, fmt.Errorf("rabbitmq queue name is empty")
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
		queueName,
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

	return &RabbitMQTxQueue{
		conn:      conn,
		channel:   ch,
		queueName: queueName,
	}, nil
}

// ConnectNewTxQueue creates a new RabbitMQ connection for the new tx queue (high priority).
func ConnectNewTxQueue(cfg config.RabbitMQConfig) (types.TxQueue, error) {
	return connectTxQueue(cfg, cfg.NewTxQueueName)
}

// ConnectOldTxQueue creates a new RabbitMQ connection for the old tx queue (backfill).
func ConnectOldTxQueue(cfg config.RabbitMQConfig) (types.TxQueue, error) {
	return connectTxQueue(cfg, cfg.OldTxQueueName)
}

// Publish enqueues a transaction hash with its block height.
func (q *RabbitMQTxQueue) Publish(txHash string, height int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := TxMessage{TxHash: txHash, Height: height}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal tx message: %w", err)
	}

	return q.channel.PublishWithContext(
		ctx,
		"",          // exchange
		q.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
}

// Consume consumes transaction hashes and invokes the handler.
func (q *RabbitMQTxQueue) Consume(handler func(txHash string, height int64) error) error {
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
		var msg TxMessage
		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			_ = delivery.Nack(false, false)
			continue
		}

		if err := handler(msg.TxHash, msg.Height); err != nil {
			_ = delivery.Nack(false, true)
			continue
		}

		_ = delivery.Ack(false)
	}

	return nil
}

// Close closes the queue resources.
func (q *RabbitMQTxQueue) Close() error {
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
