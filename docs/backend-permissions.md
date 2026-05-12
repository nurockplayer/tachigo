# Backend Permission Rules

## Purpose

This document records the backend permission rules that are currently enforced
by the Go API. It is a current-state reference for dashboard, agency, streamer,
viewer, and internal service access.

It does not introduce new product policy. When this document conflicts with
code, the code is the source of truth and this document should be updated in the
same PR that changes the permission behavior.

## Role Model

Backend user roles are stored on `models.User.Role` and carried in access token
claims.

| Role | Current meaning |
|---|---|
| `viewer` | Default account role for viewers and regular users. |
| `streamer` | Channel operator role for dashboard channel, raffle, and airdrop work. |
| `agency` | Agency operator role for agency-owned streamer/channel work. |
| `admin` | Global operator role for administrative setup and cross-account access. |

## Enforcement Layers

The API uses two main permission gates:

| Layer | Mechanism | Behavior |
|---|---|---|
| Authentication | `middleware.JWTAuth` | Requires `Authorization: Bearer <token>`, validates the access token, and stores claims in Gin context. Missing or invalid tokens return 401. |
| Role authorization | `middleware.RequireRole(...)` | Allows any listed role. Authenticated callers with a non-matching role return 403. |

Handlers add ownership checks after the route-level role gate when the route is
scoped to a channel, streamer, agency, or raffle owned by a specific user.

## Public And Viewer Routes

The following surfaces are not dashboard role-gated:

| Surface | Current access |
|---|---|
| `/health` | Public liveness check. |
| `/swagger/*any` | Public only when Swagger is enabled by config. |
| Public auth routes under `/api/v1/auth` | Public, with rate limits on selected login/registration/reset paths. |
| `GET /api/v1/claim/:token` | Public claim lookup. |
| `POST /api/v1/claim/:token` | Requires a valid user JWT for the winner claim submission path. |
| `/api/v1/extension/auth/login` | Public Twitch Extension login exchange. |
| `/api/v1/extension/t-point/complete` and `/bits/complete` | Public completion callbacks with public endpoint rate limiting. |
| `/api/v1/extension/raffles/:id/result` | Public raffle result read. |
| `/api/v1/extension/watch/*` | Requires a valid tachigo JWT after extension login. |
| `/api/v1/users/me/*`, `/api/v1/spend/redeem`, provider unlink, email verification send, and address routes | Require a valid user JWT and operate on the authenticated user. |

Viewer JWTs do not pass dashboard, agency management, event, or admin role
gates unless a route explicitly lists `viewer`, which the current dashboard and
admin route groups do not.

## Dashboard Routes

All `/api/v1/dashboard/*` routes require a valid JWT and one of
`admin`, `streamer`, or `agency` before route-specific checks run.

| Route | Allowed roles | Additional ownership behavior |
|---|---|---|
| `POST /dashboard/streamers` | `admin` | Creates streamer records for a supplied user/channel. |
| `GET /dashboard/streamers` | `agency`, `admin` | Admin lists all; agency lists streamers owned by that agency user. |
| `GET /dashboard/streamers/:streamer_id/stats` | `streamer`, `agency`, `admin` | Streamer must own that streamer record; agency must own the streamer; admin can access any. Non-admin unauthorized streamer IDs return 404 to avoid existence enumeration. |
| `POST /dashboard/streamers/register` | `streamer` | Registers the caller's own streamer channel. |
| `GET /dashboard/streamers/channels` | `streamer` | Lists channels owned by the caller. |
| `GET /dashboard/channels/:channel_id/stats` | `admin`, `streamer` | Non-admin callers must own the channel. Agencies are not route-level allowed here. |
| `GET /dashboard/channels/:channel_id/config` | `admin`, `streamer`, `agency` | Streamer or agency callers must own the channel. |
| `PUT /dashboard/channels/:channel_id/config` | `admin`, `streamer` | Streamer callers must own the channel. Agencies are not route-level allowed to update config. |
| `POST /dashboard/channels/:channel_id/airdrop` | `admin`, `streamer`, `agency` | Streamer or agency callers must own the channel. |

Raffle management routes under `/api/v1/dashboard/raffles` are currently
streamer-only at the route level. The raffle service then verifies the caller's
ownership of the target raffle where a raffle ID is supplied.

## Agency Management Routes

Agency management routes use JWT authentication and route-level role checks.
Agency-scoped routes also check that an `agency` caller is operating on its own
agency user ID.

| Route | Allowed roles | Additional ownership behavior |
|---|---|---|
| `POST /api/v1/agencies` | `admin` | Creates an agency user and attempts setup email delivery. |
| `GET /api/v1/agencies/:id` | `agency`, `admin` | Agency callers can only read their own agency record. |
| `PUT /api/v1/agencies/:id/settings` | `agency`, `admin` | Agency callers can only update their own agency record. |
| `GET /api/v1/agencies/:id/streamers` | `agency`, `admin` | Agency callers can only list streamers for their own agency record. |
| `POST /api/v1/agencies/:id/resend-setup` | `admin` | Re-sends setup email for agencies that have not completed onboarding. |

## Event And Admin Stubs

The current event and admin groups are permission-gated, but their handlers are
still placeholders.

| Route group | Allowed roles | Current behavior |
|---|---|---|
| `/api/v1/events/*` | `streamer`, `agency`, `admin` | Stub handlers return 501 after authorization succeeds. |
| `/api/v1/admin/*` | `admin` | Stub handlers return 501 after authorization succeeds. |

These stubs should not be treated as complete product APIs until their handlers
are implemented and tested.

## Internal Service Route

`/api/v1/internal/tachiya/users/points/balance` is not JWT-authenticated. It is
registered only when the API has both a database handle and
`TACHIYA_SHARED_SECRET` configured. Requests must include
`X-Tachiya-Internal-Secret` matching the configured secret; otherwise the API
returns 401.

This route is intended for Tachiya-to-tachigo service calls, not browser or
dashboard traffic.

## Status Code Conventions

| Situation | Current response |
|---|---|
| Missing bearer token on JWT route | 401 |
| Invalid or expired bearer token | 401 |
| Valid token but route-level role mismatch | 403 |
| Valid token and route role passes, but handler ownership check fails | 403 or 404 depending on enumeration risk for that handler |
| Authorized caller reaches not-yet-implemented event/admin stub | 501 |

## Change Guardrails

Permission changes should be handled in dedicated PRs when they do any of the
following:

- change which roles can access an existing route
- change ownership checks for channel, streamer, agency, raffle, or user data
- expose a route that was previously JWT or internal-secret protected
- turn an event/admin stub into product behavior
- change 401/403/404 behavior in a way that affects enumeration risk

Small documentation corrections can be made with the code change they describe,
but this document should not be used to introduce policy that is not yet
implemented in code.
