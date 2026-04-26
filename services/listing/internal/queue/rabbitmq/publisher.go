package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"marketplace/listing/internal/domain"
)

const moderationQueue = "moderation"

type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq open channel: %w", err)
	}

	_, err = ch.QueueDeclare(moderationQueue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq declare queue %s: %w", moderationQueue, err)
	}

	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, task domain.ModerationTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal moderation task: %w", err)
	}

	return p.ch.PublishWithContext(ctx,
		"",                // default exchange — роутинг по имени очереди
		moderationQueue,   // routing key = имя очереди
		false,             // mandatory
		false,             // immediate
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
