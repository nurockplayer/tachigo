# Auth Architecture

## Purpose

This document records the current auth state in the repo and the migration guardrails that future auth work must follow.

It is intentionally narrower than a full architecture decision document. It does not try to lock in unresolved deployment, token, or future-client decisions ahead of dedicated migration work.

This document is intended to answer:

- what auth behavior exists today
- which client surfaces currently differ
- which existing constraints future work must respect
- which auth changes require explicit migration work

## Status

- Status: draft baseline for current state and migration guardrails
- Scope: dashboard, extension, backend auth boundaries in the current repo
- Non-goal: this document does not adopt new auth contracts by itself

## Current Client Surfaces

The repo currently contains multiple client surfaces that touch auth:

- dashboard web app
- `tachimint`
  - Twitch Extension panel
  - runs inside a Twitch-controlled iframe
- `extensions/tachigo-demo-sidepanel`
  - Chrome sidepanel demo client
  - separate from `tachimint`

These surfaces should not be treated as one single runtime. Their current auth behavior and constraints are different.

## Current Shared Identity Baseline

Current auth behavior in the repo assumes one shared tachigo user/account system across clients.

Shared identity concerns currently include:

- user accounts
- auth providers
- roles and permissions
- session-related backend state

Provider support is part of the shared identity layer, but client auth contracts may still differ by runtime.

## Current Dashboard State

Observed behavior in the current repo:

- access token is kept in memory only
- refresh token is persisted in `localStorage` under key `refresh_token`
- `login()` calls `POST /api/v1/auth/login`; login stores the returned refresh token in `localStorage`
- logout sends `refresh_token` in the request body when present
- session restore on page reload is not implemented
- axios client sends `withCredentials: true`; 401 responses trigger a single silent refresh (deduped across concurrent requests) and retry
- dashboard does not currently persist a separate `current_user` payload

## Refresh Token Migration Status

| Phase | Description | Status |
|---|---|---|
| Phase 1 — Backend | httpOnly cookie set on login/refresh/logout; refresh and logout prefer cookie with body fallback | ✅ Done (PR #220) |
| Phase 2a — Dashboard axios layer | `withCredentials: true`; `hasAuthToken()`; 401 dedupe interceptor | ✅ Done (PR #338) |
| Phase 2b — Dashboard auth contract | Remove `localStorage`; cookie-based session restore; update `auth.ts` and `main.tsx` | ⏳ Pending |

Body fallback (sending `refresh_token` in request body) remains active in the backend during the transition period. It will be removed in a dedicated follow-up once all clients are confirmed to be on the cookie-based contract.

## Current Extension State

Extension auth is not uniform across extension-shaped clients in this repo.

### `tachimint`

`tachimint` is a Twitch Extension panel and should be treated as implemented current-state behavior.

Current documented and observed flow:

- the frontend uses a Twitch Extension JWT based login exchange
- the extension calls `POST /api/v1/extension/auth/login`
- the request carries an Extension JWT
- backend returns a tachigo token for follow-up requests
- existing watch flows already assume tachigo JWT usage after successful extension login

### `tachigo-demo-sidepanel`

`extensions/tachigo-demo-sidepanel` should be treated as a demo or exploratory client, not as a production auth reference.

Observed behavior:

- it has a login UI
- the login completion is currently simulated locally
- it is not yet wired to a real backend-integrated production auth flow

## Current Cross-Client Observations

Based on the current repo state, these cross-client distinctions are observable today:

- dashboard and extension do not currently present the same auth contract
- dashboard auth behavior does not currently describe extension auth behavior
- `tachimint` and `tachigo-demo-sidepanel` do not currently behave as the same runtime
- backend auth endpoints and JWT-based flows already exist and may already be depended on by current clients

This section records current-state observations only. It does not adopt new cross-client rules by itself.

## Existing Constraints

Future auth work must respect the current state of the system:

- backend auth endpoints and JWT-based flows already exist
- dashboard and extension flows may already depend on existing backend contracts
- Twitch, Google, Web3, and future provider support must remain possible
- extension login and watch flows already assume tachigo JWT usage after successful login

Because of these constraints, auth contract changes must not be introduced casually inside unrelated feature PRs.

## Changes That Need Dedicated Migration Work

In the current repo context, changes in the following areas are broad enough that they warrant dedicated migration work rather than incidental feature edits:

- dashboard auth contract redesign
- extension auth contract redesign
- refresh contract changes
- logout contract changes
- token storage strategy changes
- cookie-based session changes

These changes should be implemented through dedicated backend and frontend work, with clear rollout steps and updated documentation.

## Migration Guardrails

Do not change auth contract behavior opportunistically inside:

- page feature PRs
- dashboard-only feature work that is not explicitly scoped as auth migration
- unrelated UI cleanup
- unrelated backend refactors

If auth behavior changes, the migration must be named, scoped, and documented.

## Open Questions

The following remain intentionally unresolved and should stay unresolved in this baseline document until dedicated migration work exists:

- exact production deployment model for dashboard and backend
- exact dashboard session transport model
- exact extension token storage implementation for non-Twitch extension clients
- exact token lifetimes
- exact cookie attributes
- exact extension login handoff mechanism for non-Twitch extension clients
- exact future-client auth contracts for Firefox, mobile, desktop, or internal tools
- exact provider onboarding and account-linking flows
- exact session introspection and admin tooling

Each of these should be decided only when a dedicated migration or implementation PR is ready to own the decision and its rollout.

## Non-Goals

This document does not define:

- final auth architecture decisions beyond current repo state
- exact endpoint shapes
- exact token lifetimes
- exact cookie attributes
- exact extension storage implementation
- exact provider onboarding flows
- exact API payload schemas

Those belong in implementation-specific follow-up documents or migration PRs.

## Summary

This document is a narrow baseline for:

- current auth state in the repo
- current client boundary distinctions
- existing auth constraints
- migration guardrails for future auth changes

It is not the source of truth for unresolved architecture decisions.
