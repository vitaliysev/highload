package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"marketplace/listing/internal/domain"
)

type PromotionConsumer struct {
	ch    *amqp.Channel
	cache domain.ListingCache
}

func NewPromotionConsumer(conn *amqp.Connection, cache domain.ListingCache) (*PromotionConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq open channel: %w", err)
	}

	_, err = ch.QueueDeclare(promotionActivatedQueue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq declare queue %s: %w", promotionActivatedQueue, err)
	}

	return &PromotionConsumer{ch: ch, cache: cache}, nil
}

func (c *PromotionConsumer) Run(ctx context.Context) error {
	msgs, err := c.ch.Consume(promotionActivatedQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq channel closed")
			}
			if err := c.handle(ctx, msg); err != nil {
				log.Printf("promotion-activated handler error: %v", err)
				_ = msg.Nack(false, true)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

func (c *PromotionConsumer) handle(ctx context.Context, msg amqp.Delivery) error {
	var event domain.PromotionActivatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("unmarshal promotion-activated: %w", err)
	}

	if err := c.cache.Delete(ctx, event.ListingID); err != nil {
		return fmt.Errorf("cache delete listing %s: %w", event.ListingID, err)
	}

	log.Printf("cache invalidated for listing %s (promotion activated)", event.ListingID)
	return nil
}

func (c *PromotionConsumer) Close() error {
	return c.ch.Close()
}
