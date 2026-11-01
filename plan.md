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

---

# Admin field management

## Goal

Let admins create and maintain the courts shown to users, including each
court's description, image, price, location, facilities, operational status,
and weekly opening hours.

## Admin workflows

- Add a court with its name, type, location, description, hourly price,
  facilities, image, status, and opening hours for each day of the week.
- Edit any of those details later, including replacing or removing the image
  and changing days/hours or marking a day closed.
- Preview the image and see clear validation or save errors before/after
  submission.
- Access these actions only from an admin-protected management screen; public
  field listings should continue to show only active courts.
- Warn admins when changing operating hours or status may affect future
  availability. Never silently alter or cancel existing bookings.

## Implementation notes

- Build on the existing admin field create/update endpoints and schedule
  management routes rather than duplicating backend behavior.
- Add an admin field list and create/edit form, with weekday schedule controls
  and client-side validation that complements server-side validation.
- The field model currently stores a single `image_url`. Implement image
  selection, preview, and persistence using the application's configured image
  storage; if no upload storage exists, decide and document that integration
  before implementation rather than pretending a local preview is persisted.
- Keep price, status, schedule, and field edits consistent across the admin
  form, public field cards/details, and booking availability.

## Acceptance criteria

- An admin can create a court with all required details and weekly hours; it
  appears in public browsing when active and its availability matches the
  saved schedule.
- An admin can edit a court, including its description, image, price, and
  hours; refreshed public pages and availability use the saved values.
- Invalid prices, incomplete required fields, unsupported images, and invalid
  or contradictory hours are rejected with actionable messages.
- Non-admin users cannot access management actions or the underlying write
  endpoints.
- Existing bookings remain intact when a court is edited or deactivated.

## Implementation checklist

- [x] Add an admin field-management route and list with create/edit actions.
- [x] Implement a validated form for field details and per-weekday schedules.
- [x] Persist image URL changes and show a live image preview.
- [x] Wire create/update APIs with atomic field-and-schedule updates; handle
  loading, errors, and success.
- [x] Confirm admin write routes require authentication and the admin role;
  update flow leaves bookings untouched.

## Validation plan

- Add focused tests for form validation, API payloads, and schedule/image
  handling.
- Test create, edit, image replacement, closed days, and invalid input as an
  admin; verify a non-admin cannot write.
- Confirm saved details appear in public listings and detail pages and that
  availability reflects saved opening hours without changing existing
  bookings.

## Validation results

- `go test ./...` and `go vet ./...`: passed.
- Frontend build passed; `npm test`: 18 passed.
- `npm run lint`: passed with existing warnings in other files.
- Unauthenticated requests to the admin field list and schedule endpoint
  returned HTTP 401. Browser-driven flow was not verified because the
  integrated browser connection timed out.

---

# AI assistant: real-time availability suggestions

## Goal

When a user asks the ArenaGO Assistant which courts are available for a date
and optionally a requested hour/range, check live field availability and show
the matching courts as tappable cards. A card should open that court's booking
schedule with the requested date and any matching hours preselected.

## Behavior

- Read availability from the existing fields and per-field availability APIs;
  never let the language model invent booking availability.
- Understand common Indonesian date phrases (today, tomorrow, the day after
  tomorrow, weekdays, numeric dates, and written month dates) and hour/range
  forms. Use recent user context only when the user explicitly refers to the
  earlier date or time.
- Ask a concise follow-up when a date is missing or a referenced hour cannot
  be resolved. Preserve the pending availability question so a short answer
  such as "besok" continues the check.
- Show available courts, location, hourly price, and matching/free slots with
  the prompt "Ketuk lapangan pilihanmu kalau ingin booking." If no court matches,
  say so and invite the user to try another date or time.
- Tapping a court opens `/jadwal/:id`, selects the requested date, and
  preselects requested hours only after the server confirms they remain free.

## Implementation status

- [x] Add deterministic Indonesian availability intent/date/time parsing with
  Node regression tests.
- [x] Query active courts and live availability from the existing API.
- [x] Render tappable court cards in the assistant and preserve clarification
  context.
- [x] Carry suggested dates and hours into the schedule picker, validating
  suggested slots again against the latest availability response.

## Validation

- `npm run build`: passed.
- `npm test`: 11 passed, 0 failed.
- `npm run lint`: passed with existing warnings in unrelated effects/context files.
- `go test ./...`: passed.
- Browser/API: a request for 15 November 2026, 19:00–21:00 showed five live matching courts; tapping one opened its schedule with the date and both slots selected.
- Browser: a missing-date question asked for clarification; replying "15 November 2026" then showed available courts and correctly listed all available times without interpreting the day number as an hour.
- `git diff --check`: passed.
