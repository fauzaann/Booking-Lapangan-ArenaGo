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
	"Booking-Lapangan/pkg/jwt"
	"Booking-Lapangan/pkg/validator"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/router"
	"Booking-Lapangan/worker"
)

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

	// ---------- dependency injection ----------
	uow := repository.NewUnitOfWork(db)
	validate := validator.New()
	tokens := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Duration)
	invoices := config.NewXenditClient(cfg)

	authController := controllers.NewAuthController(uow.User(), tokens)
	fieldController := controllers.NewFieldController(uow)
	bookingController := controllers.NewBookingController(uow, invoices, controllers.BookingPolicy{
		MinHours:           cfg.Booking.MinHours,
		MaxHours:           cfg.Booking.MaxHours,
		CancelMinHours:     cfg.Booking.CancelMinHours,
		InvoiceDuration:    cfg.Xendit.InvoiceDuration,
		SuccessRedirectURL: cfg.Xendit.SuccessRedirectURL,
		FailureRedirectURL: cfg.Xendit.FailureRedirectURL,
	})
	paymentController := controllers.NewPaymentController(uow, invoices, cfg.Xendit.WebhookToken)
	adminController := controllers.NewAdminController(uow)

	engine := router.New(router.Dependencies{
		Config:   cfg,
		JWT:      tokens,
		Auth:     handler.NewAuthHandler(authController, validate),
		User:     handler.NewUserHandler(authController, adminController, validate),
		Field:    handler.NewFieldHandler(fieldController, validate),
		Schedule: handler.NewScheduleHandler(fieldController, validate),
		Booking:  handler.NewBookingHandler(bookingController, validate),
		Payment:  handler.NewPaymentHandler(paymentController, validate),
		Admin:    handler.NewAdminHandler(adminController, validate),
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
