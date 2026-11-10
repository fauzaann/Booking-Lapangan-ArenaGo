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

	uow := repository.NewUnitOfWork(db)
	notifController := controllers.NewNotificationController(
		uow, notifications, reminders, cfg.Mongo,
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
			UserRepository: uow.User(), PaymentRepository: uow.Payment(),
		},
	)
	whatsAppWorker := worker.NewWhatsAppDeliveryWorker(
		whatsAppJobs, uow.User(), sender, cfg.WhatsApp, cfg.Mongo.WorkerInterval,
	)
	outboxWorker := worker.NewNotificationOutboxWorker(
		uow.NotificationOutbox(), notifController, cfg.Mongo.WorkerInterval,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go reminderWorker.Start(ctx)
	go whatsAppWorker.Start(ctx)
	go outboxWorker.Start(ctx)

	log.Println("notification-worker: running")
	<-ctx.Done()
	log.Println("notification-worker: shutting down")
}
