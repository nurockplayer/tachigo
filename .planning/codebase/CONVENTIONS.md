# Coding Conventions

**Analysis Date:** 2026-04-04

## Naming Patterns

**Files:**
- Components: `PascalCase.tsx` (e.g., `LoginPage.tsx`, `ProtectedRoute.tsx`, `Button.tsx`)
- Services: `camelCase.ts` (e.g., `api.ts`, `auth.ts`)
- Hooks: `useCamelCase.ts` (e.g., `useBits.ts`, `useTwitch.ts`, `useHeartbeat.ts`)
- Types/interfaces: `camelCase.ts` (e.g., `twitch.ts`)
- Models (Go): `snake_case.go` (e.g., `auth_service.go`, `points_service.go`)
- Handlers (Go): `snake_case_handler.go` (e.g., `auth_handler.go`, `channel_config_handler.go`)

**Functions:**
- React components: `PascalCase` (e.g., `function LoginPage()`, `export default function ProtectedRoute()`)
- Regular functions/utilities: `camelCase` (e.g., `setAuthToken()`, `getMessages()`, `parseBalanceFromHeartbeatResponse()`)
- Go functions: `PascalCase` for exported (e.g., `func (s *AuthService) Register()`, `func MustClaims()`)
- Go helper functions: `camelCase` for unexported (e.g., `func newTestDB()`, `func newAuthSvc()`)

**Variables:**
- React state: `camelCase` (e.g., `const [email, setEmail]`, `const [isLoading, setIsLoading]`)
- TypeScript types: `PascalCase` (e.g., `BitsTransaction`, `TwitchContext`, `LoginResponse`)
- Constants: `UPPER_SNAKE_CASE` for truly constant values (e.g., `const testAccessSecret =`, `const claimsKey =`)
- Interfaces (TypeScript): `PascalCase` (e.g., `interface ButtonProps`, `interface TwitchContext`)

**Types:**
- Go: Exported structs `PascalCase` (e.g., `type AuthService struct`, `type PointsBalance struct`)
- Go: Error variables `Err...` (e.g., `var ErrInsufficientBalance`, `var ErrEmailExists`)
- TypeScript: `interface ...` or `type ...` for both domain types and component props
- String unions for states: lowercase (e.g., `type Status = 'idle' | 'pending' | 'success' | 'error'`)

## Code Style

**Formatting:**
- **Frontend:** ESLint 9.39.4 + typescript-eslint 8.58.0 (tachimint) and 8.57.0 (dashboard)
- **Backend:** Go standard formatting (gofmt)
- Line length: TypeScript follows default ESLint, Go follows standard 100-char convention
- Indentation: 2 spaces (TypeScript/TSX), tab (Go)

**Linting:**
- ESLint configs:
  - Base: `@eslint/js.configs.recommended`
  - TypeScript: `tseslint.configs.recommended`
  - React: `reactHooks.configs.flat.recommended` + `reactRefresh.configs.vite`
  - Ignores: `dist` directories
- Key enforcement:
  - React Hooks rules: enforced via `eslint-plugin-react-hooks`
  - React Refresh: enforced via `eslint-plugin-react-refresh`
  - TypeScript strict mode enabled in tsconfig (see TESTING.md for Go test settings)

**TypeScript Strict Mode:**
```json
{
  "strict": true,
  "noUnusedLocals": true,
  "noUnusedParameters": true,
  "noFallthroughCasesInSwitch": true,
  "noUncheckedSideEffectImports": true
}
```

## Import Organization

**Order:**
1. React/external libraries (`import { ... } from 'react'`, `import axios from 'axios'`)
2. Project services (`import { login } from '@/services/auth'`, `import client from '@/services/api'`)
3. Components (`import { Button } from '@/components/ui/button'`)
4. Types (`import type { BitsTransaction } from '../types/twitch'`)
5. Local relative imports at end (rarely needed with path aliases)

**Path Aliases:**
- **Dashboard:** `@/*` → `src/*` (defined in `dashboard/tsconfig.app.json`)
- **Tachimint:** No path aliases defined (uses relative imports)
- **Backend (Go):** Package imports always fully qualified (e.g., `github.com/tachigo/tachigo/internal/services`)

**Example (Frontend):**
```typescript
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { isAxiosError } from 'axios'
import { login } from '@/services/auth'
import { Button } from '@/components/ui/button'
import type { BitsTransaction } from '../types/twitch'
```

**Example (Backend):**
```go
import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)
```

## Error Handling

**Frontend:**
- Axios errors checked with `isAxiosError()` helper imported from `axios`
- HTTP status codes examined: `err.response?.status === 401` pattern
- Error messages stored in state (e.g., `const [error, setError] = useState('')`)
- Errors displayed conditionally in JSX: `{error && <p className="...">{error}</p>}`
- Async/await with try/catch blocks for async operations
- Fallback messages for unexpected errors: `catch (err) { setError(t.errorConnection) }`

**Backend (Go):**
- Package-level error variables: `var ErrEmailExists = errors.New("email already exists")`
- Sentinel error matching: `if errors.Is(err, ErrEmailExists) { ... }`
- Transaction rollback automatic via `db.Transaction()` on error return
- HTTP error responses: `c.AbortWithStatusJSON(401, gin.H{"success": false, "error": "..."})`
- Helper functions with `.catch(() => {})` for non-critical external calls

## Logging

**Framework:**
- Frontend: `console` (no library)
- Backend: `log` package (built-in Go)

**Patterns:**
- Frontend: Minimal logging; errors logged only for debugging (`console.error()` in edge cases)
- Backend: `log.Fatalf()` for critical startup errors (e.g., DB connection, config load)
- Silent test mode: `logger.Default.LogMode(logger.Silent)` in test database setup

## Comments

**When to Comment:**
- Complex logic that isn't self-documenting (e.g., response payload fallbacks in `tachimint/src/services/api.ts`)
- Non-obvious business rules (e.g., "points ledger index: only one active session per user/channel")
- Workarounds and temporary solutions (e.g., `// Non-fatal: bits flow still works via extension JWT directly`)
- Gotchas and tricky patterns (e.g., "Accept a few common API shapes to keep frontend resilient while backend evolves")

**JSDoc/TSDoc:**
- Exported functions: Use JSDoc comments (e.g., `/** Exchange a Twitch Extension JWT... */`)
- Function docstring pattern:
  ```typescript
  /**
   * Exchange a Twitch Extension JWT + Bits transaction receipt for a tachigo token.
   */
  export async function completeBitsTransaction(...)
  ```
- Go comments: Exported functions use `// FunctionName does ...` pattern (implicit per Go convention)

## Function Design

**Size:**
- Prefer small, focused functions (<30 lines when practical)
- Large functions broken into logical helpers with meaningful names

**Parameters:**
- Frontend: Props passed as single object with destructuring: `({ email, setEmail }) => { ... }`
- Backend (Go): Multiple named parameters; error always last: `func (s *Service) Method(input Input) (*Output, error)`
- Callback functions accept event objects: `(event: { preventDefault(): void }) => {}`

**Return Values:**
- React hooks return objects with multiple values: `const { context, jwt, products, bitsEnabled, authError } = useTwitch()`
- Service methods return result + error (Go pattern): `user, tokens, err := svc.Register(...)`
- Frontend async functions return promises: `async function handleSubmit()`
- Null/undefined coalescing in JSX: `{balance?.toLocaleString() ?? '—'}`

## Module Design

**Exports:**
- React components: Default export `export default function ComponentName()`
- Services: Named exports for functions (e.g., `export function setAuthToken()`, `export async function login()`)
- Types: Named exports: `export interface BitsTransaction`, `export type TachigoToken`
- UI library items: Both named and default: `export { Button, buttonVariants }`

**Barrel Files:**
- Not heavily used; most imports are direct paths
- Component structure: `src/components/` organized by type (e.g., `ui/`, `pages/`)
- Services all at `src/services/` (flat; no subdirectories)

## Accessibility & Responsive Design

**Tailwind/CVA Pattern (Dashboard):**
- Button variants: `const buttonVariants = cva('...', { variants: { variant: { ... }, size: { ... } } })`
- Utility classes for responsive: `flex min-h-screen items-center justify-center`
- Component composition: Slot pattern for flexible rendering `<Comp ... />`

**Extensions (tachimint):**
- No component library; custom CSS for Twitch extension styles
- Class naming: `ext-*` prefix for Twitch extension UI (e.g., `ext-panel`, `ext-balance`, `ext-btn`)

---

*Convention analysis: 2026-04-04*
