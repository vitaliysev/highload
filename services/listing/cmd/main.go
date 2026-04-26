package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	rediscache "marketplace/listing/internal/cache/redis"
	"marketplace/listing/internal/handler"
	repopg "marketplace/listing/internal/repository/postgres"
	"marketplace/listing/internal/queue/rabbitmq"
	"marketplace/listing/internal/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(mustEnv("DB_DSN"))
	if err != nil {
		log.Fatalf("parse db dsn: %v", err)
	}
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         mustEnv("REDIS_ADDR"),
		PoolSize:     10,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	conn, err := amqp.Dial(mustEnv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer conn.Close()

	listingRepo := repopg.NewListingRepo(db)
	listingCache := rediscache.NewListingCache(rdb)

	publisher, err := rabbitmq.NewPublisher(conn)
	if err != nil {
		log.Fatalf("create publisher: %v", err)
	}
	defer publisher.Close()

	svc := service.New(listingRepo, listingCache, publisher)
	h := handler.New(svc)

	mux := http.NewServeMux()
	h.Register(mux)

	promotionConsumer, err := rabbitmq.NewPromotionConsumer(conn, listingCache)
	if err != nil {
		log.Fatalf("create promotion consumer: %v", err)
	}
	defer promotionConsumer.Close()

	go func() {
		if err := promotionConsumer.Run(ctx); err != nil {
			log.Printf("promotion consumer: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:         ":" + getEnv("HTTP_PORT", "8080"),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listing-service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("listing-service shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
