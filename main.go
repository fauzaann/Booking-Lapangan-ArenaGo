// Command booking-field menjalankan REST API Sistem Booking Lapangan Olahraga.
//
// Seluruh dependency dirakit di sini (dependency injection) lalu diturunkan
// ke router: repository -> controller -> handler -> router.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Booking-Lapangan/config"
	"Booking-Lapangan/controllers"
	"Booking-Lapangan/database"
	"Booking-Lapangan/handler"
	"Booking-Lapangan/pkg/fieldcache"
	"Booking-Lapangan/pkg/jwt"
	"Booking-Lapangan/pkg/observability"
	"Booking-Lapangan/pkg/validator"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/router"
	"Booking-Lapangan/worker"
)

// main merakit dependency aplikasi, menjalankan server, dan mengelola shutdown.
func main() {
	seedOnly := flag.Bool("seed", false, "jalankan seeder lalu keluar")
	migrateOnly := flag.Bool("migrate-only", false, "jalankan migrasi lalu keluar")
	flag.Parse()

	log.SetFlags(log.LstdFlags)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Samakan zona waktu proses dengan zona waktu database agar perhitungan
	// jam booking konsisten.
	if location, locErr := time.LoadLocation(cfg.Database.TimeZone); locErr == nil {
		time.Local = location
	} else {
		log.Printf("config: unable to load timezone %s, using system default", cfg.Database.TimeZone)
	}

	db, err := config.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() {
		if closeErr := config.CloseDatabase(db); closeErr != nil {
			log.Printf("database: failed to close connection: %v", closeErr)
		}
	}()
	log.Println("database: connected")

	if cfg.App.AutoMigrate || *migrateOnly {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("migration: %v", err)
		}
		log.Println("database: migration completed")
	}
	if *migrateOnly {
		return
	}

	if cfg.App.RunSeeder || *seedOnly {
		if err := database.Seed(db); err != nil {
			log.Fatalf("seeder: %v", err)
		}
	}
	if *seedOnly {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("database: failed to get sql.DB: %v", err)
	}
	telemetry, err := observability.New(context.Background())
	if err != nil {
		log.Fatalf("observability: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(shutdownCtx); err != nil {
			log.Printf("observability: failed to shut down tracing: %v", err)
		}
	}()

	// ---------- dependency injection ----------
	uow := repository.NewUnitOfWork(db)
	validate := validator.New()
	tokens := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Duration)
	invoices := config.NewXenditClient(cfg)
	assistant := config.NewOpenRouterClient(cfg)
	redisClient := config.NewRedisClient(cfg.Redis)
	defer func() {
		if closeErr := config.CloseRedis(redisClient); closeErr != nil {
			log.Printf("redis: failed to close connection: %v", closeErr)
		}
	}()
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := config.PingRedis(redisCtx, redisClient); err != nil {
		log.Printf("redis: not connected (%v) - field cache will fall back to PostgreSQL", err)
	} else {
		log.Println("redis: connected")
	}
	redisCancel()

	// Inisialisasi MongoDB (Notifikasi & Pengingat Jadwal Bermain)
	mongoClient, mongoDB, mongoErr := config.NewMongoDB(cfg)
	var (
		notifRepo       repository.NotificationRepository
		reminderRepo    repository.ReminderRepository
		whatsAppJobs    repository.WhatsAppJobRepository
		notifController controllers.NotificationController
		notifHandler    *handler.NotificationHandler
	)

	if mongoErr != nil {
		log.Printf("mongodb: not connected (%v) - notification and reminder features running in standby mode", mongoErr)
	} else {
		defer func() {
			if closeErr := config.CloseMongoDB(mongoClient); closeErr != nil {
				log.Printf("mongodb: failed to close connection: %v", closeErr)
			}
		}()
		log.Println("mongodb: connected")

		notifRepo = repository.NewNotificationRepository(mongoDB)
		reminderRepo = repository.NewReminderRepository(mongoDB)
		whatsAppJobs = repository.NewWhatsAppJobRepository(mongoDB)

		// Buat index di MongoDB
		indexCtx, indexCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := notifRepo.EnsureIndexes(indexCtx); err != nil {
			log.Printf("mongodb: failed to ensure notification indexes: %v", err)
		}
		if err := reminderRepo.EnsureIndexes(indexCtx); err != nil {
			log.Printf("mongodb: failed to ensure reminder indexes: %v", err)
		}
		if err := whatsAppJobs.EnsureIndexes(indexCtx); err != nil {
			log.Printf("mongodb: failed to ensure WhatsApp outbox indexes: %v", err)
		}
		indexCancel()
	}

	notifController = controllers.NewNotificationController(uow, notifRepo, reminderRepo, cfg.Mongo, controllers.NotificationOptions{
		WhatsAppConfig: controllers.WhatsAppNotificationConfig{
			PaymentReminderDelay:       cfg.WhatsApp.PaymentReminderDelay,
			BookingConfirmedTemplate:   cfg.WhatsApp.BookingConfirmedTemplate,
			BookingCancelledTemplate:   cfg.WhatsApp.BookingCancelledTemplate,
			BookingRescheduledTemplate: cfg.WhatsApp.BookingRescheduledTemplate,
			VenueAnnouncementTemplate:  cfg.WhatsApp.VenueAnnouncementTemplate,
		},
		WhatsAppJobs: whatsAppJobs,
	})
	notifHandler = handler.NewNotificationHandler(notifController, validate)

	authController := controllers.NewAuthController(uow.User(), tokens)
	fieldCache := fieldcache.New(redisClient, cfg.Redis.CacheTTL)
	fieldController := controllers.NewFieldController(uow, fieldCache)
	ratingController := controllers.NewRatingController(uow, fieldCache)
	bookingReads := repository.NewBookingReadRepository(uow.Booking())
	bookingController := controllers.NewBookingController(uow, bookingReads, invoices, controllers.BookingPolicy{
		MinHours:           cfg.Booking.MinHours,
		MaxHours:           cfg.Booking.MaxHours,
		CancelMinHours:     cfg.Booking.CancelMinHours,
		InvoiceDuration:    cfg.Xendit.InvoiceDuration,
		SuccessRedirectURL: cfg.Xendit.SuccessRedirectURL,
		FailureRedirectURL: cfg.Xendit.FailureRedirectURL,
	})
	paymentController := controllers.NewPaymentController(uow, invoices, cfg.Xendit.WebhookToken)
	checkoutController := controllers.NewCheckoutController(uow, invoices, controllers.BookingPolicy{
		MinHours: cfg.Booking.MinHours, MaxHours: cfg.Booking.MaxHours,
		CancelMinHours: cfg.Booking.CancelMinHours, InvoiceDuration: cfg.Xendit.InvoiceDuration,
		SuccessRedirectURL: cfg.Xendit.SuccessRedirectURL, FailureRedirectURL: cfg.Xendit.FailureRedirectURL,
	})
	adminController := controllers.NewAdminController(uow)

	engine := router.New(router.Dependencies{
		Config:        cfg,
		JWT:           tokens,
		Auth:          handler.NewAuthHandler(authController, validate),
		User:          handler.NewUserHandler(authController, adminController, validate),
		Field:         handler.NewFieldHandler(fieldController, validate),
		Schedule:      handler.NewScheduleHandler(fieldController, validate),
		Booking:       handler.NewBookingHandler(bookingController, validate),
		Checkout:      handler.NewCheckoutHandler(checkoutController, validate),
		Payment:       handler.NewPaymentHandler(paymentController, validate),
		Assistant:     handler.NewAssistantHandler(controllers.NewAssistantController(assistant), validate),
		Admin:         handler.NewAdminHandler(adminController, validate),
		Notification:  notifHandler,
		Rating:        handler.NewRatingHandler(ratingController, validate),
		AuditLog:      uow.AuditLog(),
		Observability: telemetry,
		Readiness: []router.ReadinessCheck{
			{Name: "postgres", Required: true, Check: func(ctx context.Context) error { return sqlDB.PingContext(ctx) }},
			{Name: "mongodb", Required: false, Check: func(ctx context.Context) error {
				if mongoClient == nil {
					return mongoErr
				}
				return mongoClient.Ping(ctx, nil)
			}},
			{Name: "redis", Required: false, Check: func(ctx context.Context) error {
				return config.PingRedis(ctx, redisClient)
			}},
		},
	})

	server := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go telemetry.StartOutboxCollector(ctx, db, mongoDB)

	expirer := worker.NewBookingExpirer(uow, time.Duration(cfg.Booking.ExpirerMinutes)*time.Minute)
	go expirer.Start(ctx)

	go func() {
		log.Printf("server: listening on http://localhost:%s (env=%s)", cfg.App.Port, cfg.App.Env)
		log.Printf("server: api docs available on http://localhost:%s/docs", cfg.App.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("server: shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server: forced shutdown: %v", err)
	}
	log.Println("server: stopped gracefully")
}
