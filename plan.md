# Multi-hour booking plan

## Goal
Allow a user to reserve a continuous block of multiple hourly slots in one booking, with matching duration and price.

## Current behavior
- The backend already accepts bookings from the configured minimum through `BOOKING_MAX_HOUR` (default: 8), computes duration from start/end, and calculates the authoritative total price.
- The slot picker currently permits arbitrary non-adjacent selections. The confirmation page converts the earliest and latest selected slots into one continuous booking, while its displayed total is calculated from the number of selected slots. Those values can disagree.

## Implementation status
- Added contiguous selection in `ArenaGO-padel-frontend/src/lib/bookingSlots.js`: adjacent available hours extend the selected range; selecting a separate hour starts a new range; booked slots cannot be added.
- Updated the slot picker to use the range logic and explain multi-hour selection. Duration and price continue to derive from the selected contiguous hours and are passed through the existing confirmation flow.
- Kept backend duration, opening-hours, availability, and price validation authoritative; no client-supplied total is trusted.
- Added Node built-in regression tests for extending a range, starting a separate range, rejecting a booked slot, and trimming a range.

## Validation
- `npm test`: 4 passed.
- `npm run lint`: passed with existing warnings in other components.
- `npm run build`: passed.
- `go test ./...` and `go vet ./...`: passed.
- `git diff --check`: passed.
