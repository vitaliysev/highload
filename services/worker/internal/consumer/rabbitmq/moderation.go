package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"marketplace/worker/internal/domain"
)

const (
	moderationQueue = "moderation"
	approveRate = 0.80
)

// паттерн Async Messaging: модерация не блокирует создание объявления.
// в PoC имитирует ML API случайным решением.
type ModerationConsumer struct {
	ch   *amqp.Channel
	repo domain.ListingRepository
}

func NewModerationConsumer(conn *amqp.Connection, repo domain.ListingRepository) (*ModerationConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq open channel: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("rabbitmq qos: %w", err)
	}

	_, err = ch.QueueDeclare(moderationQueue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq declare queue %s: %w", moderationQueue, err)
	}

	return &ModerationConsumer{ch: ch, repo: repo}, nil
}

func (c *ModerationConsumer) Run(ctx context.Context) error {
	msgs, err := c.ch.Consume(moderationQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}

	log.Println("moderation worker started")

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq channel closed")
			}
			if err := c.handle(ctx, msg); err != nil {
				log.Printf("moderation error: %v — nack+requeue", err)
				_ = msg.Nack(false, true)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

func (c *ModerationConsumer) handle(ctx context.Context, msg amqp.Delivery) error {
	var task domain.ModerationTask
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		_ = msg.Nack(false, false)
		return fmt.Errorf("unmarshal moderation task: %w", err)
	}

	log.Printf("moderating listing %s: %q", task.ListingID, task.Title)

	// имитация работы ML API
	delay := time.Duration(500+rand.Intn(2000)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	status := domain.ModerationRejected
	if rand.Float64() < approveRate {
		status = domain.ModerationApproved
	}

	if err := c.repo.UpdateStatus(ctx, task.ListingID, string(status)); err != nil {
		return fmt.Errorf("update status for %s: %w", task.ListingID, err)
	}

	log.Printf("listing %s → %s", task.ListingID, status)
	return nil
}

func (c *ModerationConsumer) Close() error {
	return c.ch.Close()
}
