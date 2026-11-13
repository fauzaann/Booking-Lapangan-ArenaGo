# Notification RabbitMQ migration plan

## Goal

Move notification event delivery off the notification worker's direct
PostgreSQL outbox polling. The booking/payment application remains the owner of
its database and writes notification events in the same transaction as business
changes. A separate publisher forwards those records to RabbitMQ; the
notification worker consumes messages and does not query booking or payment
tables.

## Delivery sequence

1. Extend notification contract v1 with the minimum booking, venue, schedule,
   customer-contact, consent, and price snapshots needed by notification
   processing. Persist the serialized event data in the PostgreSQL transactional
   outbox at the time the business transaction commits.
2. Add durable RabbitMQ exchange/queue topology and an outbox publisher. Claim
   rows using the existing lease mechanism, publish persistent messages, wait
   for publisher confirms, then mark the outbox row processed. Retry failures
   through the existing outbox backoff.
3. Change the notification worker to consume the durable notification queue
   with manual acknowledgements. Acknowledge only after handling succeeds;
   retry transient errors and route exhausted/invalid messages to a dead-letter
   queue. Preserve the existing MongoDB inbox/reminder/WhatsApp repositories.
4. Remove booking/payment repository access from event handling. Retain the
   existing live WhatsApp-consent check until a user-preferences event/API is
   introduced; booking snapshots alone must not weaken consent revocation.
5. Document environment, topology, startup order, and rollback. Validate using
   focused publisher/consumer tests and `go test ./...`.

## RabbitMQ topology

- Durable topic exchange: `arenago.events`.
- Durable queue: `arenago.notification.events`, bound to booking lifecycle and
  payment invoice-ready routing keys.
- Durable retry queue with a TTL and dead-letter routing back to the main
  exchange.
- Durable dead-letter queue for invalid or exhausted messages.
- Persistent JSON messages use the versioned contract in
  `contracts/notification/v1`.

## Delivery guarantees and rollout

Delivery is at-least-once. A publisher crash after broker confirmation but
before marking the outbox row processed can publish a duplicate. Reminder
records already have a unique booking/type key; in-app notifications and
WhatsApp outbox jobs use the event ID as an idempotency key. The outbox row is
marked processed only after broker confirmation, and a broker delivery is
acknowledged only after successful handling. Keep existing outbox rows during
rollout. Start the publisher before switching the notification worker to
consume RabbitMQ; stop the legacy outbox consumer to avoid processing the same
event through both paths.

## Not included in this step

This migration does not extract the public notification HTTP API or move MongoDB
ownership. Live WhatsApp consent still uses the user repository; extracting
that dependency requires a separate user-preferences contract/API.
