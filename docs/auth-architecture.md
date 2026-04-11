# Auth Architecture

## Purpose

This document defines the authentication architecture for tachigo across current and future clients.

The goal is to keep one shared identity system while allowing different client types to use different auth contracts when their runtime constraints differ.

This is a decision and boundary document. It is intended to answer:

- which clients exist now
- which clients must be supported later
- which auth concerns are shared
- which auth contracts are client-specific
- which changes require explicit migration work

## Status

- Status: draft architecture baseline
- Scope: dashboard, extension, backend auth boundaries, future client compatibility
- Non-goal: this document does not directly change any API contract by itself

## Design Principles

- One shared account and identity system across all clients
- Client-specific auth contracts when runtime constraints differ
- Security decisions should not be mixed into unrelated feature PRs
- Auth contract changes must go through explicit migration work
- Dashboard and extension are both first-class clients
- Identity provider concerns and client auth contract concerns must be kept separate

## Adopted Decisions

The following decisions are adopted as the current architecture baseline:

1. Production web deployment defaults to a subdomain model.
2. tachigo keeps one shared identity and account system across clients.
3. Dashboard and extension use different auth contracts when needed.
4. The extension communicates directly with tachigo backend.
5. Users must log in to the tachigo extension before using extension-based product flows.
6. The architecture must support future clients beyond the current dashboard and Chromium extension.
7. Session governance must support current-device logout, global logout, revocation, and rotation.
8. Auth contract redesign must be handled as dedicated migration work.

## Deployment Baseline

The default production deployment model is subdomain-based:

- `dashboard.tachigo.com`
- `api.tachigo.com`

This gives better deployment separation than a single-path setup while remaining more web-auth-friendly than fully separate root domains.

### Why subdomain-based deployment

Benefits:

- cleaner separation between frontend and backend deployment
- more flexibility for infrastructure, scaling, routing, and security controls
- more compatible with future web-oriented session handling than fully separate root domains

Trade-offs:

- still requires origin-aware configuration
- cookie, `withCredentials`, CORS, and `SameSite` behavior still need explicit design
- local development may require extra setup if subdomain behavior must be simulated

Decision trigger:

- exact CORS, cookie, `withCredentials`, and `SameSite` behavior must be defined before any PR introduces a dashboard cookie-based session transport or cross-origin credentialed requests

## Client Matrix

### Current clients

- Dashboard web app
- Chromium-based browser extensions
  - Chrome
  - Edge

### Future-compatible clients

- Firefox extension
- Mobile app
- Desktop app
- Internal admin and operations tools

### Architecture implication

The auth layer must not assume that every future client behaves like a browser tab.

The dashboard is a web app.
The extension is an extension runtime.
Future mobile and desktop clients may introduce a third runtime family.

For that reason, auth must be designed as:

- one shared identity model
- multiple client-facing auth contracts

### Important distinction inside "extension"

The repository currently contains two different extension-shaped clients and they must not be treated as the same runtime:

- `tachimint`
  - Twitch Extension panel
  - runs inside a Twitch page iframe
  - already uses Twitch Extension JWT based login exchange
- `extensions/tachigo-demo-sidepanel`
  - Chrome sidepanel demo client
  - has its own login UI
  - currently behaves like a demo surface, not a fully integrated production auth client

This distinction matters because runtime constraints, storage choices, and auth transport assumptions differ significantly.

## Shared Identity Model

tachigo uses one shared identity and account system for all clients.

Shared identity concerns include:

- user account
- auth providers
- roles and permissions
- session store
- session revocation policy

Identity providers may include:

- Twitch
- Google
- Web3
- future providers

Provider support belongs to the identity layer and should not be confused with client-specific auth contracts.

### Identity layer responsibilities

The shared identity layer is responsible for:

- mapping external providers to tachigo users
- enforcing role and permission rules
- maintaining user-linked sessions
- preserving future provider extensibility

## Auth Contract Split

Dashboard and extension do not share the exact same auth contract.

They share the same identity system, but each client family can use a different auth flow when needed.

### Why contracts are split

The runtime constraints differ:

- dashboard runs as a web application
- extension runs inside browser extension primitives
- extension may execute through popup, background, and content-script boundaries
- extension must work across Twitch, YouTube, 17 Live, and future livestream platforms

Forcing one unified contract across all clients would create a lowest-common-denominator design and would likely make both dashboard and extension worse.

### Shared vs client-specific concerns

Shared concerns:

- identity
- providers
- roles
- revocation model
- session governance rules

Client-specific concerns:

- login initiation flow
- token or session transport
- storage model
- refresh behavior
- logout behavior

## Dashboard Auth Contract

The dashboard is a web-first client.

Its auth design should optimize for:

- secure browser-based session handling
- low-friction admin and streamer usage
- compatibility with a subdomain deployment model

### Dashboard direction

The dashboard may evolve toward a more web-oriented session model over time.

This document intentionally does not force an immediate cookie migration.
Any future cookie-based change must be treated as explicit migration work, not as an incidental feature PR change.
That future direction applies to the dashboard only and must not be generalized to extension runtimes.

### Dashboard current state

Current dashboard auth behavior in the repo is a transitional implementation:

- access token: in-memory only, not persisted
- refresh token: persisted in `localStorage` under key `refresh_token`
- login flow: `login()` calls `POST /api/v1/auth/login`, stores the returned refresh token in `localStorage`, and keeps the access token in memory
- session restore on page reload: not implemented in the current dashboard client
- 401 auto-refresh: not implemented in the current dashboard client
- persisted user metadata: none beyond the refresh token; dashboard does not currently persist a separate `current_user` payload on this branch
- backend contract in current use: logout sends `refresh_token` in the request body when present; no current frontend refresh flow is wired against `/api/v1/auth/refresh`

Architecture implication:

- the current dashboard implementation should be treated as the migration starting point
- dashboard auth contract work must explicitly define whether the product will continue with a body-based refresh model, reintroduce a restore and refresh flow, move to cookie-based refresh, or use another approach
- no feature PR should silently redefine this contract

## Extension Auth Contracts

tachigo should treat extension clients as a family, not as one single contract.

All extension clients should follow the same high-level principle:

- direct communication with tachigo backend
- independence from Twitch, YouTube, 17 Live, or other platform site cookies
- compatibility with extension runtime constraints
- future portability across browser-extension clients

However, the concrete auth flow differs by extension runtime.

### `tachimint` contract: Twitch Extension panel

`tachimint` is a Twitch Extension panel that runs in a Twitch-controlled iframe.

Its current auth assumptions are tightly coupled to Twitch Extension runtime primitives:

- Twitch provides an Extension JWT
- the frontend exchanges that JWT with tachigo backend
- backend returns a tachigo token used for follow-up requests

This is a current, implemented client contract and should be treated as current-state behavior.

### `tachigo-demo-sidepanel` contract: Chrome sidepanel demo

`extensions/tachigo-demo-sidepanel` should be treated separately from `tachimint`.

Current observed state:

- it has a login UI
- the login screen currently uses a local timed mock transition rather than a real backend login call
- it should therefore be treated as a demo or exploratory client, not as a production-auth reference implementation

Architecture implication:

- the sidepanel is useful as a design and runtime prototype
- but it must not be treated as proof that a production sidepanel auth contract is already finalized

## Extension Communication Model

The extension communicates directly with tachigo backend.

The livestream platform page is treated as runtime context or data source, not as the core auth boundary.

Content scripts, popup UI, and background scripts are implementation details inside the extension. They should not define the backend auth contract.

### Rejected direction

The extension should not rely on:

- dashboard session state as its core auth source
- platform-site cookies as auth truth
- a page bridge as the main backend auth transport strategy

Those approaches create unwanted coupling and make cross-platform support harder.

## Extension Login Model

Users must log in to the tachigo extension before using extension-based product flows.

### Current extension login state

Current documented and observed behavior is not uniform across extension clients:

#### `tachimint` current flow

Current docs already describe an extension login flow in which:

- the extension calls `POST /api/v1/extension/auth/login`
- the request carries an Extension JWT
- backend links that identity to a tachigo user
- backend returns a tachigo JWT

Current watch endpoints in existing docs also assume:

- extension watch flows use tachigo JWT after extension login succeeds
- platform identity is treated as provider context rather than long-term account ownership

#### `tachigo-demo-sidepanel` current flow

The sidepanel currently exposes a login screen, but that UI is not yet a production backend-integrated auth flow.

Observed state:

- user interaction is handled locally
- the login completion is currently simulated
- the sidepanel should therefore be treated as a demo client until a real backend auth flow is wired in

### Target extension login direction

The preferred target flow for extension-first clients is:

1. The extension initiates login.
2. The user is sent to a tachigo-controlled auth web page.
3. The user completes login using supported identity providers.
4. The backend creates the appropriate extension session or token state.
5. The extension receives the result through a handoff flow that is not yet finalized.

The extension must not depend on dashboard login state as a required prerequisite.

Current note:

- the exact handoff mechanism is intentionally undecided at this stage
- possible future options may include an OAuth-style code exchange, browser-mediated callback flow, or another extension-compatible handoff pattern
- the handoff mechanism must be defined before a production non-Twitch extension auth contract is implemented

## Session and Token Policy

tachigo should prioritize strong security with low user friction.

### Required capabilities

- current-device logout
- global logout
- server-side revocation of a specific session
- refresh or session token rotation

### User experience goals

- short-lived access tokens
- background refresh where appropriate
- minimal unnecessary login interruptions during normal usage
- explicit re-authentication only when refresh fails, a session is revoked, or a risk event occurs

### Recommended baseline

- access token lifetime: short-lived
- refresh or session lifetime: longer-lived but revocable
- rotation on refresh: enabled
- implementation reference target:
  - access token: less than or equal to 15 minutes
  - refresh or session token: less than or equal to 30 days

Exact durations should be defined at implementation time based on client-specific constraints.

## Security Roadmap

The following are recommended enhancements but are not required for the first architecture milestone:

- device and session metadata
- suspicious session detection
- replay detection
- richer operator tooling for session visibility

## Existing Constraints

The architecture must respect the current state of the system:

- backend auth endpoints and JWT-based flows already exist
- dashboard and extension flows may already depend on existing backend contracts
- Twitch, Google, Web3, and future provider support must remain possible
- extension login and watch flows already assume tachigo JWT usage after successful login

Because of these constraints, auth contract changes must not be introduced casually inside unrelated feature PRs.

## Migration Policy

Any of the following must be handled as explicit migration work:

- dashboard auth contract redesign
- extension auth contract redesign
- refresh or logout contract changes
- token storage strategy changes
- cookie-based session changes

These changes should be implemented through dedicated backend and frontend work, with clear rollout steps and updated documentation.

### Migration guardrail

Do not change auth contract behavior opportunistically inside:

- page feature PRs
- dashboard-only feature work
- unrelated UI cleanup
- unrelated backend refactors

If auth behavior changes, the migration must be named, scoped, and documented.

## Open Questions

The following are intentionally left open for follow-up work:

- exact dashboard session transport model
  - trigger: must be decided before any PR changes refresh or logout contract behavior
- whether dashboard should eventually move to a cookie-based model
  - trigger: must be decided before any backend or frontend PR proposes cookie-based migration
- exact extension token storage implementation
  - trigger: must be decided before the sidepanel or other non-Twitch extension client becomes a production auth surface
- exact token lifetimes
  - trigger: must be decided before production hardening or environment config finalization
- exact cookie attributes
  - trigger: must be decided before any dashboard cookie-based session rollout
- exact provider onboarding and linking flows
  - trigger: must be decided before expanding provider coverage or changing account-linking UX
- exact session introspection and admin tooling
  - trigger: must be decided before session governance features are exposed to operators or end users

## Non-Goals

This document does not define:

- exact endpoint shapes
- exact token lifetimes
- exact cookie attributes
- exact extension storage implementation
- exact provider onboarding flows
- exact API payload schemas

Those belong in implementation-specific follow-up documents or PRs.

## Summary

tachigo should use:

- one shared identity system
- subdomain-based production deployment as the default web baseline
- a dedicated dashboard auth contract
- dedicated extension auth contracts by runtime, not a single generic extension contract
- direct extension-to-backend communication
- extension-first login for extension usage
- strong session governance with revocation and rotation
- explicit migration work for auth contract changes
