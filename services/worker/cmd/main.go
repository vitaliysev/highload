package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"marketplace/worker/internal/consumer/rabbitmq"
	repopg "marketplace/worker/internal/repository/postgres"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(mustEnv("DB_DSN"))
	if err != nil {
		log.Fatalf("parse db dsn: %v", err)
	}
	poolCfg.MaxConns = 5
	poolCfg.MinConns = 1
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	conn, err := amqp.Dial(mustEnv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer conn.Close()

	listingRepo := repopg.NewListingRepo(db)

	consumer, err := rabbitmq.NewModerationConsumer(conn, listingRepo)
	if err != nil {
		log.Fatalf("create moderation consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.Run(ctx); err != nil {
		log.Printf("moderation consumer stopped: %v", err)
	}

	log.Println("worker stopped")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env %s is not set", key)
	}
	return v
}
