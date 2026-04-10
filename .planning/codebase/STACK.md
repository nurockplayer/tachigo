# Technology Stack

**Analysis Date:** 2026-04-04

## Languages

**Primary:**
- Go 1.25.0 - Backend API server and core business logic
- TypeScript 5.8.0 (tachimint), 5.9.3 (dashboard) - Frontend applications with strict type safety
- JavaScript/JSX - React components (React 19.2.4)

**Secondary:**
- SQL (PostgreSQL dialect) - Database queries and migrations
- Shell scripting - Development automation and setup

## Runtime

**Environment:**
- Go runtime (1.25.0) - Backend execution
- Node.js 24-alpine - Frontend JavaScript runtime via Docker
- Docker & Docker Compose - Containerized development and deployment

**Package Manager:**
- pnpm 10.33.0 - Node.js package management (enforced via `preinstall` script)
- Go modules (go.mod/go.sum) - Dependency management

## Frameworks

**Core Backend:**
- Gin v1.12.0 - HTTP web framework and routing
- GORM v1.30.0 - Object-relational mapping with PostgreSQL driver

**Core Frontend:**
- React 19.2.4 - UI component library
- Vite 8.0.2 - Build tool and dev server with HMR
- React Router 7.6.2 (dashboard only) - Client-side routing

**UI & Styling:**
- Tailwind CSS 4.2.2 (dashboard only) - Utility-first CSS framework
- Radix UI 1.2.3 (dashboard only) - Unstyled accessible components
- Lucide React 0.511.0 (dashboard only) - Icon library
- class-variance-authority 0.7.1 (dashboard only) - Component styling utilities
- clsx/tailwind-merge - CSS class composition utilities (dashboard)

**Testing:**
- Go: Built-in testing package + testify (inferred from test files)
- TypeScript: No test framework detected in package.json (testing not configured)

**Build & Development:**
- Swag v1.16.6 - Swagger/OpenAPI code generation for Go
- air - Hot reload for Go development
- ESLint 9.39.4 - Linting (both frontend apps)
- TypeScript compiler (tsc) - Type checking before Vite build
- PostCSS 8.5.8 (dashboard only) - CSS transformation
- Autoprefixer 10.4.27 (dashboard only) - Vendor prefixes

## Key Dependencies

**Critical Backend:**
- github.com/ethereum/go-ethereum v1.17.2 - Ethereum/Web3 integration for wallet signatures and address handling
- github.com/golang-jwt/jwt/v5 v5.3.1 - JWT creation and verification for auth tokens
- golang.org/x/oauth2 v0.15.0 - OAuth2 flow implementation for Twitch and Google login
- gopkg.in/gomail.v2 - SMTP email sending via gomail

**Database:**
- gorm.io/driver/postgres v1.6.0 - PostgreSQL connection and migrations
- gorm.io/driver/sqlite v1.6.0 - SQLite support for testing

**Utilities:**
- github.com/google/uuid v1.6.0 - UUID generation
- golang.org/x/crypto v0.48.0 - Bcrypt hashing and cryptographic utilities
- github.com/joho/godotenv v1.5.1 - .env file loading

**Frontend Critical:**
- axios 1.14.0 - HTTP client for API calls (both apps)
- @types/twitch-ext 1.24.9 (tachimint only) - Twitch Extension SDK type definitions

## Configuration

**Environment:**
- Backend: `.env` file with 20+ configuration variables (see backend/.env.example)
  - Database connection (PostgreSQL DSN)
  - JWT secrets and TTLs
  - OAuth credentials (Twitch, Google)
  - SMTP configuration
  - CORS allowed origins
- Frontend: `.env` files with minimal configuration (API base URL)
  - `VITE_TACHIGO_API_URL` (tachimint)
  - `VITE_API_URL` (dashboard)

**Build:**
- `backend/Dockerfile` - Multi-stage build (dev, builder, distroless runtime)
- `dashboard/Dockerfile` - Node.js alpine with pnpm
- `tachimint/Dockerfile` - Node.js alpine with pnpm
- `docker-compose.yml` - Service orchestration (app, postgres, frontend, dashboard)
- `backend/.air.toml` - Air hot reload configuration
- `tsconfig.json` (both frontends) - TypeScript compiler options
- `.editorconfig` - Cross-editor formatting
- `.eslintrc` - ESLint configuration for both frontends

## Platform Requirements

**Development:**
- Docker & Docker Compose (primary development environment)
- Go 1.25.0 (if running backend locally outside Docker)
- Node.js 24+ with pnpm 10.33.0 (if running frontend locally)
- PostgreSQL 16-alpine (via Docker)
- Git

**Production:**
- Docker container runtime
- PostgreSQL 16+ database
- Network access to external OAuth providers (Twitch, Google)
- SMTP server for email delivery (optional, has fallback NoOpMailer)
- SSL/TLS termination (reverse proxy)

**External Service Dependencies:**
- Twitch OAuth endpoints (id.twitch.tv)
- Google OAuth endpoints (googleapis.com)
- Twitch Helix API (api.twitch.tv) - for user info during OAuth flow
- Ethereum RPC (implicit via go-ethereum, not actively used yet based on code analysis)

---

*Stack analysis: 2026-04-04*
