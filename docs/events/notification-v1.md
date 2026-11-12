# Notification event contract v1

The booking/payment application owns booking and payment state. The notification
service owns inbox delivery, scheduled reminders, and WhatsApp delivery. Events
are the asynchronous integration boundary between them.

The Go transport contract lives in
[`contracts/notification/v1`](../../contracts/notification/v1). Consumers should
depend on this JSON schema, not on application `models` or GORM entities.

## Envelope

```json
{
  "id": "notification-outbox-123",
  "type": "booking.confirmed",
  "version": 1,
  "occurred_at": "2026-11-12T12:00:00Z",
  "data": {
    "booking_id": "42"
  }
}
```

`id` is opaque to consumers and is the deduplication key. IDs in payloads are
strings so clients in different languages do not need to rely on integer width.
`occurred_at` is an RFC 3339 timestamp. Consumers must reject unknown versions
and unsupported event types, and must be idempotent because delivery is
at-least-once.

## Event types

| Type | Payload |
| --- | --- |
| `booking.confirmed` | `booking_id` |
| `booking.cancelled` | `booking_id` |
| `booking.rescheduled` | `booking_id` |
| `payment.invoice_ready` | `booking_id`, `payment_url`, `expires_at` |

Example invoice event:

```json
{
  "id": "notification-outbox-124",
  "type": "payment.invoice_ready",
  "version": 1,
  "occurred_at": "2026-11-12T12:00:00Z",
  "data": {
    "booking_id": "42",
    "payment_url": "https://checkout.example/invoice",
    "expires_at": "2026-11-12T13:00:00Z"
  }
}
```

Booking references are intentionally identifiers, not copies of booking or user
entities. In the current modular-monolith bridge, the notification handler uses
the booking ID to read booking details. Before extracting it as an independent
service, replace that lookup with a booking-service API or add the minimal,
privacy-reviewed notification snapshot to the event contract.

## Versioning

Do not change the meaning or type of an existing v1 field. Add optional fields
only when old consumers can safely ignore them. For incompatible changes,
introduce a new contract version and support both versions during producer and
consumer rollout.
