---
title: Dashboard Auth And Session Boundary
status: active
owner: engineering
last_reviewed: 2026-06-01
source_of_truth: true
code_areas:
  - apps/dashboard
  - services/api
related_issues:
  - 763
---

# Dashboard Auth And Session Boundary

## Purpose

This document records the current dashboard authentication, session restore, and
frontend RBAC boundary. It is a current-state reference for issue #763 and does
not introduce new auth behavior by itself.

When this document conflicts with code, the code is the source of truth. Update
this document in the same PR that changes dashboard auth, session persistence,
or role-boundary behavior.

## Current Runtime Contract

The dashboard uses `apps/dashboard/src/services/api.ts` as its Axios client and
`apps/dashboard/src/providers/authProvider.ts` as the Refine auth bridge.

| Area | Current behavior |
|---|---|
| Login | `login()` posts to `/api/v1/auth/login` and stores the returned access token in the dashboard API client. |
| Access token storage | The access token is kept in module memory only. The dashboard app does not persist it in `localStorage` or `sessionStorage`. |
| Refresh/session state | The Axios client sends `withCredentials: true`. The refresh token is owned by the backend-managed httpOnly cookie. |
| App bootstrap | `main.tsx` calls `restoreSession()` before rendering the app so a valid backend refresh cookie can restore an in-memory access token after reload. |
| Silent refresh | A 401 response on an authenticated request triggers one deduped `/api/v1/auth/refresh` call, then retries the original request with the refreshed access token. |
| Logout | `logout()` posts to `/api/v1/auth/logout` and clears the in-memory access token. |

The dashboard should not start persisting refresh tokens or user payloads in
browser storage without a dedicated auth migration PR.

## Route And Role Boundary

Dashboard page routing uses Refine's `<Authenticated>` wrapper around the main
authenticated route tree in `apps/dashboard/src/App.tsx`. Unauthenticated users
are redirected to `/login`.

Frontend role handling is currently advisory:

- `authProvider.getPermissions()` returns the role decoded from the current
  access token.
- `Layout` filters some navigation items by decoded role.
- Some pages use the decoded role for local presentation decisions.

The backend remains the source of truth for access control. Dashboard API routes
under `/api/v1/dashboard/*` must still rely on backend JWT authentication,
route-level role gates, and handler ownership checks documented in
[`backend-permissions.md`](./backend-permissions.md).

## Known Gaps For Issue #763

Issue #763 is broader than this document. The current dashboard still needs
dedicated implementation PRs for the remaining hardening work:

| Gap | Expected follow-up |
|---|---|
| Route-level role guards | Add explicit frontend route guard behavior for dashboard pages whose visibility differs by role. |
| Component-level RBAC | Keep role-sensitive controls hidden or disabled on the frontend while preserving backend enforcement. |
| Session failure UX | Normalize expired-session, failed-refresh, and logout states so the user lands on `/login` predictably. |
| API error handling | Normalize 401/403 handling across dashboard data providers and direct service calls. |
| Regression coverage | Add tests for auth restore, expired session, wrong-role navigation, and backend-denied dashboard calls. |

These follow-ups should not change backend role policy casually. Permission
changes require a dedicated PR and an update to
[`backend-permissions.md`](./backend-permissions.md).

## Manual Verification Checklist

Use this checklist when changing dashboard auth or session behavior:

1. Login with valid dashboard credentials and confirm authenticated routes render.
2. Reload an authenticated dashboard route and confirm the session is restored
   when the backend refresh cookie is valid.
3. Expire or remove the backend refresh cookie, reload, and confirm the app
   redirects to `/login` without keeping a stale access token.
4. Logout and confirm subsequent dashboard API calls do not include the old
   access token.
5. Use a `viewer` or otherwise unauthorized role and confirm backend dashboard
   calls return 403 even if a frontend page or control is reachable.
6. Confirm no dashboard auth change writes access tokens, refresh tokens, or
   current-user payloads to `localStorage` or `sessionStorage`.

## Non-Goals

This document does not:

- complete issue #763
- change dashboard auth, route guard, or API behavior
- define new backend permissions
- change token lifetime, cookie attributes, or refresh semantics
- replace [`auth-architecture.md`](./auth-architecture.md) or
  [`backend-permissions.md`](./backend-permissions.md)
