package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Booking-Lapangan/config"
	"Booking-Lapangan/controllers"
	"Booking-Lapangan/pkg/rabbitmq"
	"Booking-Lapangan/pkg/whatsapp"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/worker"
)

func main() {
	log.SetFlags(log.LstdFlags)

	cfg, err := config.LoadNotificationWorker()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if (cfg.WhatsApp.PhoneNumberID == "") != (cfg.WhatsApp.AccessToken == "") {
		log.Fatal("config: WHATSAPP_PHONE_NUMBER_ID and WHATSAPP_ACCESS_TOKEN must be configured together")
	}
	rabbitClient, err := rabbitmq.New(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer func() {
		if err := rabbitClient.Close(); err != nil {
			log.Printf("rabbitmq: failed to close connection: %v", err)
		}
	}()
	if location, err := time.LoadLocation(cfg.Database.TimeZone); err == nil {
		time.Local = location
	} else {
		log.Printf("config: unable to load timezone %s, using system default", cfg.Database.TimeZone)
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

	mongoClient, mongoDB, err := config.NewMongoDB(cfg)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	defer func() {
		if err := config.CloseMongoDB(mongoClient); err != nil {
			log.Printf("mongodb: failed to close connection: %v", err)
		}
	}()

	notifications := repository.NewNotificationRepository(mongoDB)
	reminders := repository.NewReminderRepository(mongoDB)
	whatsAppJobs := repository.NewWhatsAppJobRepository(mongoDB)
	indexCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	for name, ensure := range map[string]func(context.Context) error{
		"notifications": notifications.EnsureIndexes,
		"reminders":     reminders.EnsureIndexes,
		"WhatsApp jobs": whatsAppJobs.EnsureIndexes,
	} {
		if err := ensure(indexCtx); err != nil {
			cancel()
			log.Fatalf("mongodb: failed to ensure %s indexes: %v", name, err)
		}
	}
	cancel()

	var sender whatsapp.Sender
	if cfg.WhatsApp.PhoneNumberID != "" && cfg.WhatsApp.AccessToken != "" {
		sender = whatsapp.NewClient(cfg.WhatsApp.PhoneNumberID, cfg.WhatsApp.AccessToken, cfg.WhatsApp.BaseURL)
		log.Println("whatsapp: Cloud API delivery enabled")
	} else {
		log.Println("whatsapp: Cloud API credentials are not configured; WhatsApp jobs remain queued")
	}

	userRepository := repository.NewUserRepository(db)
	notifController := controllers.NewNotificationController(
		nil, notifications, reminders, cfg.Mongo,
		controllers.NotificationOptions{
			WhatsAppConfig: controllers.WhatsAppNotificationConfig{
				PaymentReminderDelay:       cfg.WhatsApp.PaymentReminderDelay,
				BookingConfirmedTemplate:   cfg.WhatsApp.BookingConfirmedTemplate,
				BookingCancelledTemplate:   cfg.WhatsApp.BookingCancelledTemplate,
				BookingRescheduledTemplate: cfg.WhatsApp.BookingRescheduledTemplate,
				VenueAnnouncementTemplate:  cfg.WhatsApp.VenueAnnouncementTemplate,
			},
			WhatsAppJobs: whatsAppJobs,
		},
	)
	reminderWorker := worker.NewReminderWorker(
		reminders, notifications, cfg.Mongo.WorkerInterval,
		worker.ReminderWorkerOptions{
			WhatsAppConfig: cfg.WhatsApp, WhatsAppSender: sender,
			UserRepository: userRepository,
		},
	)
	whatsAppWorker := worker.NewWhatsAppDeliveryWorker(
		whatsAppJobs, userRepository, sender, cfg.WhatsApp, cfg.Mongo.WorkerInterval,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go reminderWorker.Start(ctx)
	go whatsAppWorker.Start(ctx)
	consumerErr := make(chan error, 1)
	go func() {
		consumerErr <- rabbitClient.Consume(ctx, notifController.HandleOutboxEvent)
	}()

	log.Println("notification-worker: consuming RabbitMQ events")
	select {
	case <-ctx.Done():
	case err := <-consumerErr:
		if err != nil {
			log.Fatalf("notification-worker: RabbitMQ consumer stopped: %v", err)
		}
		if ctx.Err() == nil {
			log.Fatal("notification-worker: RabbitMQ consumer stopped unexpectedly")
		}
	}
	log.Println("notification-worker: shutting down")
}
