package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	notificationv1 "Booking-Lapangan/contracts/notification/v1"
	"Booking-Lapangan/config"
)

const (
	deadLetterRoutingKey = "notification.dead"
	retryHeader          = "x-notification-retries"
)

type Client struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	config     config.RabbitMQConfig
	confirms   chan amqp.Confirmation
}

func New(cfg config.RabbitMQConfig) (*Client, error) {
	parsed, err := url.Parse(cfg.URL)
	if err != nil || (parsed.Scheme != "amqp" && parsed.Scheme != "amqps") || parsed.Host == "" {
		return nil, fmt.Errorf("RABBITMQ_URL must be a valid amqp or amqps URL")
	}
	if cfg.Exchange == "" || cfg.Queue == "" || cfg.RetryExchange == "" ||
		cfg.RetryQueue == "" || cfg.DeadLetterExchange == "" || cfg.DeadLetterQueue == "" {
		return nil, fmt.Errorf("RabbitMQ exchange and queue names must not be empty")
	}
	if cfg.RetryDelay <= 0 || cfg.MaxRetries <= 0 {
		return nil, fmt.Errorf("RabbitMQ retry delay and max retries must be greater than zero")
	}

	connection, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	client := &Client{connection: connection, channel: channel, config: cfg}
	if err := client.declareTopology(); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, err
	}
	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, fmt.Errorf("enable RabbitMQ publisher confirms: %w", err)
	}
	client.confirms = channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	return client, nil
}

func (c *Client) declareTopology() error {
	for _, exchange := range []struct {
		name string
		kind string
	}{
		{c.config.Exchange, amqp.ExchangeTopic},
		{c.config.RetryExchange, amqp.ExchangeTopic},
		{c.config.DeadLetterExchange, amqp.ExchangeDirect},
	} {
		if err := c.channel.ExchangeDeclare(exchange.name, exchange.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare RabbitMQ exchange %s: %w", exchange.name, err)
		}
	}

	if _, err := c.channel.QueueDeclare(c.config.DeadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare RabbitMQ dead-letter queue: %w", err)
	}
	if err := c.channel.QueueBind(c.config.DeadLetterQueue, deadLetterRoutingKey, c.config.DeadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("bind RabbitMQ dead-letter queue: %w", err)
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    c.config.DeadLetterExchange,
		"x-dead-letter-routing-key": deadLetterRoutingKey,
	}
	if _, err := c.channel.QueueDeclare(c.config.Queue, true, false, false, false, mainArgs); err != nil {
		return fmt.Errorf("declare RabbitMQ notification queue: %w", err)
	}
	for _, routingKey := range []string{"booking.*", "payment.invoice_ready"} {
		if err := c.channel.QueueBind(c.config.Queue, routingKey, c.config.Exchange, false, nil); err != nil {
			return fmt.Errorf("bind RabbitMQ notification queue for %s: %w", routingKey, err)
		}
	}

	retryArgs := amqp.Table{
		"x-message-ttl":             int32(c.config.RetryDelay / time.Millisecond),
		"x-dead-letter-exchange":    c.config.Exchange,
	}
	if _, err := c.channel.QueueDeclare(c.config.RetryQueue, true, false, false, false, retryArgs); err != nil {
		return fmt.Errorf("declare RabbitMQ retry queue: %w", err)
	}
	if err := c.channel.QueueBind(c.config.RetryQueue, "#", c.config.RetryExchange, false, nil); err != nil {
		return fmt.Errorf("bind RabbitMQ retry queue: %w", err)
	}
	return nil
}

func (c *Client) Publish(ctx context.Context, event notificationv1.Envelope) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate notification event before publish: %w", err)
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode notification event: %w", err)
	}
	if err := c.publish(ctx, c.config.Exchange, string(event.Type), event.ID, body, nil); err != nil {
		return err
	}
	return c.waitConfirmation(ctx)
}

func (c *Client) Consume(ctx context.Context, handler func(context.Context, notificationv1.Envelope) error) error {
	if err := c.channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("set RabbitMQ consumer QoS: %w", err)
	}
	deliveries, err := c.channel.ConsumeWithContext(ctx, c.config.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("start RabbitMQ notification consumer: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("RabbitMQ notification delivery channel closed")
			}
			c.processDelivery(ctx, delivery, handler)
		}
	}
}

func (c *Client) processDelivery(
	ctx context.Context,
	delivery amqp.Delivery,
	handler func(context.Context, notificationv1.Envelope) error,
) {
	var event notificationv1.Envelope
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		c.rejectToDeadLetter(ctx, delivery, fmt.Errorf("decode notification event: %w", err))
		return
	}
	if err := event.Validate(); err != nil {
		c.rejectToDeadLetter(ctx, delivery, fmt.Errorf("invalid notification event: %w", err))
		return
	}
	if err := handler(ctx, event); err != nil {
		c.retryOrDeadLetter(ctx, delivery, err)
		return
	}
	if err := delivery.Ack(false); err != nil {
		log.Printf("rabbitmq: failed to acknowledge event %s: %v", event.ID, err)
	}
}

func (c *Client) retryOrDeadLetter(ctx context.Context, delivery amqp.Delivery, cause error) {
	retries := headerInt(delivery.Headers[retryHeader])
	if retries >= c.config.MaxRetries {
		c.rejectToDeadLetter(ctx, delivery, cause)
		return
	}
	headers := copyHeaders(delivery.Headers)
	headers[retryHeader] = int32(retries + 1)
	if err := c.publish(ctx, c.config.RetryExchange, delivery.RoutingKey, delivery.MessageId, delivery.Body, headers); err != nil {
		log.Printf("rabbitmq: failed to publish retry for event %s: %v", delivery.MessageId, err)
		c.nackAndRequeue(delivery)
		return
	}
	if err := c.waitConfirmation(ctx); err != nil {
		log.Printf("rabbitmq: retry publish was not confirmed for event %s: %v", delivery.MessageId, err)
		c.nackAndRequeue(delivery)
		return
	}
	if err := delivery.Ack(false); err != nil {
		log.Printf("rabbitmq: failed to acknowledge retried event %s: %v", delivery.MessageId, err)
	}
	log.Printf("rabbitmq: event %s scheduled for retry (%d/%d): %v", delivery.MessageId, retries+1, c.config.MaxRetries, cause)
}

func (c *Client) rejectToDeadLetter(ctx context.Context, delivery amqp.Delivery, cause error) {
	if err := c.publish(ctx, c.config.DeadLetterExchange, deadLetterRoutingKey, delivery.MessageId, delivery.Body, delivery.Headers); err != nil {
		log.Printf("rabbitmq: failed to dead-letter event %s after %v: %v", delivery.MessageId, cause, err)
		c.nackAndRequeue(delivery)
		return
	}
	if err := c.waitConfirmation(ctx); err != nil {
		log.Printf("rabbitmq: dead-letter publish was not confirmed for event %s: %v", delivery.MessageId, err)
		c.nackAndRequeue(delivery)
		return
	}
	if err := delivery.Ack(false); err != nil {
		log.Printf("rabbitmq: failed to acknowledge dead-lettered event %s: %v", delivery.MessageId, err)
	}
	log.Printf("rabbitmq: event %s moved to dead-letter queue: %v", delivery.MessageId, cause)
}

func (c *Client) publish(
	ctx context.Context,
	exchange string,
	routingKey string,
	messageID string,
	body []byte,
	headers amqp.Table,
) error {
	err := c.channel.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    messageID,
		Timestamp:    time.Now().UTC(),
		Headers:      headers,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish RabbitMQ message %s: %w", messageID, err)
	}
	return nil
}

func (c *Client) waitConfirmation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case confirmation, ok := <-c.confirms:
		if !ok {
			return fmt.Errorf("RabbitMQ publisher confirmation channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ did not confirm the published message")
		}
		return nil
	}
}

func (c *Client) nackAndRequeue(delivery amqp.Delivery) {
	if err := delivery.Nack(false, true); err != nil {
		log.Printf("rabbitmq: failed to requeue event %s: %v", delivery.MessageId, err)
	}
}

func headerInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}

func copyHeaders(headers amqp.Table) amqp.Table {
	copy := amqp.Table{}
	for key, value := range headers {
		copy[key] = value
	}
	return copy
}

func (c *Client) Close() error {
	channelErr := c.channel.Close()
	connectionErr := c.connection.Close()
	if channelErr != nil {
		return fmt.Errorf("close RabbitMQ channel: %w", channelErr)
	}
	if connectionErr != nil {
		return fmt.Errorf("close RabbitMQ connection: %w", connectionErr)
	}
	return nil
}
