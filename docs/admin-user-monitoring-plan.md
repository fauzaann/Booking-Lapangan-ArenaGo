# Plan: Admin User Monitoring and Account Status

## Requirements

- **REQ-001**: Admin can browse all user accounts using pagination and see each
  account's name, email, role, active status, and registration date.
- **REQ-002**: Admin can activate or deactivate regular user accounts from the
  user monitoring page.
- **REQ-003**: Inactive accounts cannot log in or continue using an existing
  authenticated session.
- **REQ-004**: User-management responses expose only the selected account data;
  status changes remain admin-only and cannot deactivate administrator accounts
  or the current administrator.

## Implementation Plan

1. **Persistence and authentication**
   - Add an `is_active` flag defaulting to `true`, and backfill existing rows.
   - Reject inactive accounts at login and validate active status on each
     authenticated API request so previously issued tokens stop working.
2. **Admin API**
   - Keep `GET /api/v1/admin/users` paginated and project only the approved
     fields.
   - Add an admin-only status update endpoint. Disallow self-deactivation and
     deactivation of administrator accounts.
3. **Admin UI**
   - Add an authenticated admin page listing account name, email, role, status,
     and registration date, with pagination and status controls.
   - Add navigation links and show clear loading, empty, success, and error
     states.
4. **Validation**
   - Test inactive login/session rejection, admin access control, status
     transition safeguards, and response field projection.
   - Run Go tests/vet and frontend tests/build.

## Tasks

- [x] T001 Add account-active persistence and migration in `models/users.go` and `database/migration.go`.
- [x] T002 Enforce account status in login and authenticated request middleware.
- [x] T003 Add the admin user status API and privacy-limited user response.
- [x] T004 Add the admin user monitoring page, route, and navigation link.
- [x] T005 Add focused tests and run backend/frontend validation.

## Requirement Mapping

| REQ ID | Description | Plan items | Implementation evidence |
|---|---|---|---|
| REQ-001 | Browse paginated user names and selected profile fields | 2, 3 | `controllers/admin_controller.go`, `dto/auth_request.go`, `ArenaGO-padel-frontend/src/pages/AdminUsers.jsx` |
| REQ-002 | Activate/deactivate regular accounts | 2, 3 | `controllers/admin_controller.go`, `handler/user_handler.go`, `ArenaGO-padel-frontend/src/pages/AdminUsers.jsx` |
| REQ-003 | Block inactive accounts from login and existing sessions | 1 | `controllers/auth_controller.go`, `middleware/auth_middleware.go`, `repository/user_status_repository.go` |
| REQ-004 | Limit disclosure and protect administrator access | 1, 2 | `dto/auth_request.go`, `router/router.go`, `controllers/admin_controller.go` |

## Validation

- `go test ./...` — passed.
- `go vet ./...` — passed.
- `npm test` and `npm run build` in `ArenaGO-padel-frontend` — passed.
- `npm run lint` — passed with existing warnings in other frontend files; the new page has no lint warnings.
