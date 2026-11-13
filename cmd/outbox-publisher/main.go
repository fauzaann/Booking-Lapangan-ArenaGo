package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"Booking-Lapangan/config"
	"Booking-Lapangan/pkg/rabbitmq"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/worker"
)

func main() {
	log.SetFlags(log.LstdFlags)

	cfg, err := config.LoadOutboxPublisher()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := config.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() {
		if err := config.CloseDatabase(db); err != nil {
			log.Printf("database: failed to close connection: %v", err)
		}
	}()

	rabbitClient, err := rabbitmq.New(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer func() {
		if err := rabbitClient.Close(); err != nil {
			log.Printf("rabbitmq: failed to close connection: %v", err)
		}
	}()

	publisher := worker.NewOutboxPublisherWorker(
		repository.NewNotificationOutboxRepository(db),
		rabbitClient,
		cfg.RabbitMQ.PublisherInterval,
	)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go publisher.Start(ctx)

	log.Println("outbox-publisher: running")
	<-ctx.Done()
	log.Println("outbox-publisher: shutting down")
}
