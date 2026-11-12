package observability

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"Booking-Lapangan/models"
)

type Observability struct {
	registry           *prometheus.Registry
	requests           *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	bookingConflicts   prometheus.Counter
	paymentFailures    *prometheus.CounterVec
	outboxEvents       *prometheus.GaugeVec
	outboxOldestAge    *prometheus.GaugeVec
	tracerProvider     *sdktrace.TracerProvider
	traceExportEnabled bool
}

func New(ctx context.Context) (*Observability, error) {
	registry := prometheus.NewRegistry()
	o := &Observability{
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "arenago_http_requests_total",
			Help: "Total HTTP requests handled by the API.",
		}, []string{"method", "route", "status"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "arenago_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		bookingConflicts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arenago_booking_conflicts_total",
			Help: "Booking and checkout requests rejected due to slot conflicts.",
		}),
		paymentFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "arenago_payment_operation_failures_total",
			Help: "Payment-related API operations that returned a server error.",
		}, []string{"route"}),
		outboxEvents: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "arenago_notification_outbox_events",
			Help: "Notification outbox events by status.",
		}, []string{"source", "status"}),
		outboxOldestAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "arenago_notification_outbox_oldest_pending_age_seconds",
			Help: "Age of the oldest pending or processing notification outbox event.",
		}, []string{"source"}),
	}
	if err := registry.Register(o.requests); err != nil {
		return nil, fmt.Errorf("register HTTP request counter: %w", err)
	}
	for _, collector := range []prometheus.Collector{o.requestDuration, o.bookingConflicts, o.paymentFailures, o.outboxEvents, o.outboxOldestAge} {
		if err := registry.Register(collector); err != nil {
			return nil, fmt.Errorf("register observability collector: %w", err)
		}
	}

	o.tracerProvider = sdktrace.NewTracerProvider()
	if endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); endpoint != "" {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
		}
		provider := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resource.NewWithAttributes("", attribute.String("service.name", serviceName()))),
		)
		o.tracerProvider = provider
		o.traceExportEnabled = true
	}
	otel.SetTracerProvider(o.tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return o, nil
}

func (o *Observability) Middleware() gin.HandlerFunc {
	tracer := o.tracerProvider.Tracer("arenago/http")
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		ctx, span := tracer.Start(ctx, c.Request.Method, trace.WithSpanKind(trace.SpanKindServer))
		c.Request = c.Request.WithContext(ctx)
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := c.Writer.Status()
		o.requests.WithLabelValues(c.Request.Method, route, fmt.Sprint(status)).Inc()
		o.requestDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
		if (route == "/api/v1/bookings" || route == "/api/v1/checkouts") && status == http.StatusConflict {
			o.bookingConflicts.Inc()
		}
		if isPaymentOperation(route) && status >= http.StatusInternalServerError {
			o.paymentFailures.WithLabelValues(route).Inc()
		}
		span.SetName(c.Request.Method + " " + route)
		span.SetAttributes(
			attribute.String("http.route", route),
			attribute.String("server.address", c.Request.Host),
			attribute.Int("http.response.status_code", status),
		)
		if status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(status))
		}
		span.End()
	}
}

func serviceName() string {
	if name := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); name != "" {
		return name
	}
	return "arenago-api"
}

func (o *Observability) MetricsHandler() gin.HandlerFunc {
	handler := promhttp.HandlerFor(o.registry, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

func (o *Observability) Shutdown(ctx context.Context) error {
	if !o.traceExportEnabled {
		return nil
	}
	if err := o.tracerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown trace provider: %w", err)
	}
	return nil
}

func (o *Observability) StartOutboxCollector(ctx context.Context, db *gorm.DB, mongoDB *mongo.Database) {
	o.collectOutboxStatsWithTimeout(ctx, db, mongoDB)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.collectOutboxStatsWithTimeout(ctx, db, mongoDB)
		}
	}
}

func (o *Observability) collectOutboxStatsWithTimeout(ctx context.Context, db *gorm.DB, mongoDB *mongo.Database) {
	collectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	o.collectOutboxStats(collectCtx, db, mongoDB)
}

func (o *Observability) collectOutboxStats(ctx context.Context, db *gorm.DB, mongoDB *mongo.Database) {
	var stats struct {
		Status string
		Count  float64
	}
	for _, status := range []string{
		models.NotificationEventPending,
		models.NotificationEventProcessing,
		models.NotificationEventFailed,
		models.NotificationEventProcessed,
	} {
		if err := db.WithContext(ctx).Model(&models.NotificationEvent{}).
			Select("status, COUNT(*) AS count").
			Where("status = ?", status).
			Group("status").
			Scan(&stats).Error; err != nil {
			log.Printf("observability: failed to collect notification outbox count for %s: %v", status, err)
			continue
		}
		o.outboxEvents.WithLabelValues("postgres", strings.ToLower(status)).Set(stats.Count)
		stats.Count = 0
	}

	var oldest models.NotificationEvent
	err := db.WithContext(ctx).Model(&models.NotificationEvent{}).
		Where("status IN ?", []string{models.NotificationEventPending, models.NotificationEventProcessing}).
		Order("created_at ASC").
		First(&oldest).Error
	switch {
	case err == nil:
		o.outboxOldestAge.WithLabelValues("postgres").Set(max(0, time.Since(oldest.CreatedAt).Seconds()))
	case errors.Is(err, gorm.ErrRecordNotFound):
		o.outboxOldestAge.WithLabelValues("postgres").Set(0)
	default:
		log.Printf("observability: failed to collect oldest notification outbox age: %v", err)
	}

	if mongoDB != nil {
		o.collectWhatsAppOutboxStats(ctx, mongoDB)
	}
}

func (o *Observability) collectWhatsAppOutboxStats(ctx context.Context, db *mongo.Database) {
	collection := db.Collection("whatsapp_outbox")
	for _, status := range []string{"scheduled", "processing", "failed", "sent", "cancelled"} {
		count, err := collection.CountDocuments(ctx, bson.M{"status": status})
		if err != nil {
			log.Printf("observability: failed to collect WhatsApp outbox count for %s: %v", status, err)
			continue
		}
		o.outboxEvents.WithLabelValues("mongodb", status).Set(float64(count))
	}

	var oldest struct {
		CreatedAt time.Time `bson:"created_at"`
	}
	now := time.Now()
	err := collection.FindOne(ctx, bson.M{
		"$or": bson.A{
			bson.M{"status": "scheduled", "scheduled_for": bson.M{"$lte": now}},
			bson.M{"status": "processing"},
		},
	}, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: 1}})).Decode(&oldest)
	switch {
	case err == nil:
		o.outboxOldestAge.WithLabelValues("mongodb").Set(max(0, time.Since(oldest.CreatedAt).Seconds()))
	case errors.Is(err, mongo.ErrNoDocuments):
		o.outboxOldestAge.WithLabelValues("mongodb").Set(0)
	default:
		log.Printf("observability: failed to collect oldest WhatsApp outbox age: %v", err)
	}
}

func isPaymentOperation(route string) bool {
	return route == "/api/v1/payments/webhook" ||
		route == "/api/v1/checkouts" ||
		route == "/api/v1/checkouts/:id/sync-payment" ||
		route == "/api/v1/bookings" ||
		route == "/api/v1/bookings/:id/sync-payment"
}
