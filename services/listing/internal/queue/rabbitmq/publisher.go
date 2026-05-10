package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"marketplace/listing/internal/domain"
)

const (
	moderationQueue         = "moderation"
	promotionActivatedQueue = "promotion-activated"
)

type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq open channel: %w", err)
	}

	for _, q := range []string{moderationQueue, promotionActivatedQueue} {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return nil, fmt.Errorf("rabbitmq declare queue %s: %w", q, err)
		}
	}

	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, task domain.ModerationTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal moderation task: %w", err)
	}
	return p.publish(ctx, moderationQueue, body)
}

func (p *Publisher) PublishPromotionActivated(ctx context.Context, event domain.PromotionActivatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal promotion-activated event: %w", err)
	}
	return p.publish(ctx, promotionActivatedQueue, body)
}

func (p *Publisher) publish(ctx context.Context, queue string, body []byte) error {
	return p.ch.PublishWithContext(ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (p *Publisher) Close() error {
	return p.ch.Close()
}
