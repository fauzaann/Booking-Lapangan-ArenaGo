# Runtime Validation Report

**Generated**: 2026-10-31T19:11:30+07:00  
**Target**: Booking-Lapangan (Go API + React SPA)

## Summary

| Step | Status | Exit Code | Details |
|---|---|---:|---|
| Startup | PASS | 0 | Backend connected to local PostgreSQL; `/health` returned 200. Frontend dev server returned 200 and served the React root. |
| Schema migration | PASS | 0 | `APP_AUTO_MIGRATE=true RUN_SEEDER=false go run . -migrate-only` completed. |
| Backend tests | PASS | 0 | `go test ./...`; `go vet ./...` completed. |
| Frontend build | PASS | 0 | `npm run build` completed. |
| Frontend tests | PASS | 0 | `npm test`: 4 passed, 0 failed. |
| Frontend lint | PASS | 0 | `npm run lint` completed with existing warnings in unrelated effects/context files. |
| Browser/API checks | PARTIAL | N/A | Admin page and scanner camera initialized; ticket QR rendered as a 320px PNG. Live API auth, admin dashboard, field list, unauthorized check-in, and unknown-ticket rejection passed. A real paid ticket was not scanned. |

## Environment

- Docker: UNAVAILABLE — Docker CLI exists, but `docker info` exited 1 before reporting server details.
- Node.js: AVAILABLE — v26.3.1.
- Playwright: AVAILABLE — Chromium installed; browser E2E exercised through the integrated browser.
- Infrastructure: local PostgreSQL was available and used; Redis and MongoDB were unavailable. The app logged that cache falls back to PostgreSQL and notification/reminder features are in standby.
- Payment provider: no real Xendit invoice/payment was created during this validation.

## Check-in evidence

- Paid-payment unit tests verify a secure check-in token is issued for individual bookings and checkout bookings.
- Controller tests verify only admins can check in, a paid unused ticket is accepted, the token is not returned in the check-in response, and repeat check-in is rejected.
- Live API returned 401 without authentication and 404 for an unknown ticket with admin authentication.
- Browser showed the check-in page with camera ready and no new page/console errors after scanner cleanup.
- Browser rendered a valid QR PNG for a mocked confirmed/paid booking and showed its booking details.

## Known gaps

- No full live Xendit → real paid booking → physical QR scan → successful API check-in flow was performed. Doing so requires a test payment or a dedicated paid-ticket fixture; current tests cover the backend success path.
- Redis/Mongo-dependent features were not validated because those services were not running.

**Overall**: PARTIAL — core app startup and tested QR/check-in components work locally, but the complete real-payment-to-camera-scan flow still needs a controlled paid test booking.
