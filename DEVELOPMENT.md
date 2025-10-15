# SSOReady Local Development Guide

This guide will help you set up and run SSOReady locally for development.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker Desktop** (or Docker Engine + Docker Compose)
  - [Install Docker Desktop](https://www.docker.com/products/docker-desktop)
- **Node.js 18+**
  - [Install Node.js](https://nodejs.org/) or use [nvm](https://github.com/nvm-sh/nvm)
  - Check version: `node -v`
- **Go 1.21+**
  - [Install Go](https://golang.org/dl/)
  - Check version: `go version`

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/YOUR_USERNAME/ssoready.git
cd ssoready
```

### 2. Run Setup Script

The setup script will:
- Verify prerequisites
- Create `.env` file with sensible defaults
- Install npm dependencies
- Start PostgreSQL
- Run database migrations

```bash
./bin/dev-setup
```

### 3. Start All Services

```bash
./bin/dev-start
```

This starts all SSOReady services using Docker Compose. You can also use:

```bash
docker compose up
```

To run in the background:

```bash
docker compose up -d
```

## Service URLs

Once all services are running, you can access:

| Service | URL | Description |
|---------|-----|-------------|
| **Admin UI** | http://localhost:8083 | Management interface for configuring SAML/SCIM |
| **Self-Serve UI** | http://localhost:8082 | Customer-facing setup interface |
| **API Service** | http://localhost:8080 | REST API backend |
| **Auth Service** | http://localhost:8081 | Authentication and SAML handler |
| **PostgreSQL** | localhost:5433 | Database (user: `postgres`, password: `password`) |

## Architecture

SSOReady consists of five main components:

1. **postgres**: PostgreSQL 15.3 database
2. **api**: Go-based REST API service (port 8080)
3. **auth**: Go-based authentication and SAML service (port 8081)
4. **admin**: React-based admin UI (port 8083)
5. **app**: React-based self-serve customer UI (port 8082)

## Differences from Official Docker Compose Setup

Our `compose.yaml` differs from the [official self-hosting guide](https://ssoready.com/docs/self-hosting-ssoready#example-docker-compose-setup) in several ways to optimize for **local development**:

### 1. Building from Source vs. Pre-built Images

**Official:** Uses published Docker images
```yaml
image: ssoready/ssoready-api:sha-18090f8
```

**Ours:** Builds from local source code
```yaml
build:
  context: .
  dockerfile: cmd/api/Dockerfile
```

**Why:** This allows you to modify the code and test changes immediately. Perfect for a development fork.

### 2. Hot-Reloading for Frontend Development

**Official:** Production builds only

**Ours:** Development mode with volume mounts
```yaml
volumes:
  - ./admin/src:/app/src
  - ./admin/public:/app/public
command: npm run dev
```

**Why:** Changes to React components automatically refresh in the browser without rebuilding containers.

### 3. Multi-Stage Dockerfiles

**Official:** Single production stage

**Ours:** Separate development and production stages
```dockerfile
FROM node:22.4 AS development
# ... development setup

FROM node:22.4 AS production
# ... production setup
```

**Why:** Development stage is optimized for fast iteration, production stage for deployment.

### 4. Service Configuration

Both use the same official environment variable naming convention (`AUTH_*`, `API_*`, `APP_*`, `ADMIN_*`) for consistency with SSOReady's documentation.

### When to Use Which Setup

**Use our development setup when:**
- You're modifying SSOReady code
- You want hot-reloading for frontend changes
- You're testing features or fixes

**Use the official setup when:**
- You want to run SSOReady in production
- You need a stable, tested version
- You don't need to modify the code

## Configuration

### Environment Variables

All configuration is managed through the `.env` file, which is created from `.env.example` during setup.

Our setup uses the same environment variable naming convention as the official SSOReady Docker images:
- **Backend Go services** expect prefixed variables: `AUTH_*` for auth service, `API_*` for api service
- **Frontend React apps** expect prefixed variables: `APP_*` for self-serve UI, `ADMIN_*` for admin UI

### Environment File Syncing

The root `.env` file is automatically synced to `admin/.env` and `app/.env`:
- **On setup**: `./bin/dev-setup` creates and syncs all `.env` files
- **On start**: `./bin/dev-start` syncs before starting services

This ensures consistency - you only need to edit the root `.env` file, and changes will be propagated to the frontend services automatically.

**Why separate files?** Docker build contexts for `admin/` and `app/` can only access files within their own directories, so they need local copies of `.env`.

#### Generated Development Secrets

The following secrets were auto-generated for local development. **DO NOT use these in production!**

```bash
# Pagination token encoding (64-char hex)
PAGE_ENCODING_VALUE=4288e9ce63df47bf0dfc5098787c94bfca2e8dd1f3b9ec7244e168a3f711e958

# SAML state signing (64-char hex)
SAML_STATE_SIGNING_KEY=f94052c6011652ead62004b533e80604a3717190565bb5cd85f19a7c67b83cd0

# OAuth ID token signing (Base64-encoded RSA private key)
OAUTH_ID_TOKEN_PRIVATE_KEY_BASE64=LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0t...
```

#### Database Configuration

```bash
DATABASE_URL=postgres://postgres:password@postgres:5432/postgres?sslmode=disable
```

**Important:** The hostname is `postgres` (the Docker service name), not `localhost`. This allows Docker containers to communicate with each other. If you run services outside Docker on your host machine, you would use `localhost` instead.

#### Service URLs

These configure how services communicate with each other:

```bash
DEFAULT_AUTH_URL=http://localhost:8081
DEFAULT_ADMIN_SETUP_URL=http://localhost:8083
BASE_URL=http://localhost:8081
```

#### Frontend Configuration

Configure where the UIs connect:

```bash
# Admin UI
ADMIN_API_URL=http://localhost:8080

# Self-Serve App UI
APP_API_URL=http://localhost:8080
APP_APP_URL=http://localhost:8082
APP_PUBLIC_API_URL=http://localhost:8080
```

### Optional Configuration

These are optional and can be left empty for local development:

- **Sentry**: Error tracking (`SENTRY_DSN`, `SENTRY_ENVIRONMENT`)
- **Segment**: Analytics (`SEGMENT_WRITE_KEY`)
- **PostHog**: Product analytics (`POSTHOG_API_KEY`)
- **Google OAuth**: `GOOGLE_OAUTH_CLIENT_ID`
- **Microsoft OAuth**: `MICROSOFT_OAUTH_CLIENT_ID`, `MICROSOFT_OAUTH_CLIENT_SECRET`, `MICROSOFT_OAUTH_REDIRECT_URI`

## Development Workflow

### Modifying Environment Variables

To change environment variables:

1. **Edit the root `.env` file** (not `admin/.env` or `app/.env`)
2. **Restart services** using `./bin/dev-start` (which auto-syncs) or manually:
   ```bash
   cp .env admin/.env
   cp .env app/.env
   docker compose restart app admin
   ```

The `./bin/dev-start` script automatically syncs `.env` files before starting, so you don't need to manually copy them.

### Making Changes

#### Backend (Go Services)

The Go services (`api` and `auth`) are rebuilt when you restart the containers:

```bash
docker compose up --build api auth
```

Or rebuild everything:

```bash
docker compose up --build
```

#### Frontend (React Apps)

The frontend services (`admin` and `app`) have hot-reloading enabled. Changes to files in `admin/src` or `app/src` will automatically refresh the browser.

### Database Migrations

#### Run Migrations

To run new migrations:

```bash
./bin/dev-migrate
```

Or manually:

```bash
docker compose run --rm migrate -d "$DATABASE_URL" up
```

#### Create New Migrations

Migrations are located in `cmd/migrate/migrations/`. Follow the existing naming convention:

```
000xxx_description.up.sql
000xxx_description.down.sql
```

### Accessing the Database

#### Using psql

Connect to the database using the existing script:

```bash
./bin/psql
```

Or connect directly:

```bash
docker compose exec postgres psql -U postgres
```

#### Using a GUI Client

You can connect using any PostgreSQL client (TablePlus, pgAdmin, DBeaver, etc.):

- **Host**: `localhost`
- **Port**: `5433`
- **Database**: `postgres`
- **User**: `postgres`
- **Password**: `password`

**Note:** The database is exposed on port **5433** externally to avoid conflicts with any local PostgreSQL installation you may have. Inside Docker, containers communicate on the standard port 5432.

### Viewing Logs

View logs for all services:

```bash
docker compose logs -f
```

View logs for a specific service:

```bash
docker compose logs -f api
docker compose logs -f auth
docker compose logs -f admin
docker compose logs -f app
```

## Troubleshooting

### Services Won't Start

1. **Check if ports are already in use:**
   ```bash
   lsof -i :8080  # API
   lsof -i :8081  # Auth
   lsof -i :8082  # App
   lsof -i :8083  # Admin
   lsof -i :5433  # Postgres
   ```

2. **Check Docker status:**
   ```bash
   docker ps
   docker compose ps
   ```

3. **View service logs for errors:**
   ```bash
   docker compose logs api
   ```

### Database Connection Issues

1. **Ensure PostgreSQL is running:**
   ```bash
   docker compose up -d postgres
   docker compose exec postgres pg_isready -U postgres
   ```

2. **Check DATABASE_URL in .env:**
   ```bash
   grep DATABASE_URL .env
   ```

### Frontend Not Hot-Reloading

1. **Ensure volumes are mounted correctly in compose.yaml**
2. **Restart the frontend service:**
   ```bash
   docker compose restart admin
   # or
   docker compose restart app
   ```

### Clean Slate / Reset Everything

To completely reset your local environment:

```bash
# Stop all services
docker compose down

# Remove database data
rm -rf .postgres

# Remove node_modules (optional)
rm -rf admin/node_modules app/node_modules

# Re-run setup
./bin/dev-setup
```

### Build Errors

If you encounter build errors:

```bash
# Rebuild all containers from scratch
docker compose build --no-cache

# Or rebuild specific services
docker compose build --no-cache api auth
```

## Testing SAML/SCIM

### Setting Up a Test Organization

1. Open Admin UI: http://localhost:8083
2. Create an environment
3. Create an organization
4. Configure SAML connection with test IdP metadata

### Using Test IdPs

For testing SAML, you can use:
- [SAML Test IdP](https://samltest.id/)
- [Okta Developer Account](https://developer.okta.com/)
- [Google Workspace](https://workspace.google.com/)

## Additional Resources

- [SSOReady Documentation](https://ssoready.com/docs)
- [SAML Quickstart](https://ssoready.com/docs/saml/saml-quickstart)
- [SCIM Quickstart](https://ssoready.com/docs/scim/scim-quickstart)
- [Self-Hosting Guide](https://ssoready.com/docs/self-hosting-ssoready)

## Getting Help

- Check existing [GitHub Issues](https://github.com/ssoready/ssoready/issues)
- Join the community discussions
- Read the [main README](./README.md)

## Development Tips

### Using Go Locally (Without Docker)

If you prefer to run Go services outside Docker:

1. **Install dependencies:**
   ```bash
   go mod download
   ```

2. **Run API service:**
   ```bash
   source .env
   # Export prefixed environment variables
   export API_SERVE_ADDR=0.0.0.0:8080
   # Change postgres hostname to localhost when running outside Docker
   export API_DB=$(echo $DATABASE_URL | sed 's/@postgres:/@localhost:/')
   export API_DEFAULT_AUTH_URL=$DEFAULT_AUTH_URL
   export API_DEFAULT_ADMIN_SETUP_URL=$DEFAULT_ADMIN_SETUP_URL
   export API_PAGE_ENCODING_VALUE=$PAGE_ENCODING_VALUE
   export API_SAML_STATE_SIGNING_KEY=$SAML_STATE_SIGNING_KEY
   go run cmd/api/main.go
   ```

3. **Run Auth service:**
   ```bash
   source .env
   # Export prefixed environment variables
   export AUTH_SERVE_ADDR=0.0.0.0:8081
   # Change postgres hostname to localhost when running outside Docker
   export AUTH_DB=$(echo $DATABASE_URL | sed 's/@postgres:/@localhost:/')
   export AUTH_BASE_URL=$BASE_URL
   export AUTH_DEFAULT_ADMIN_TEST_MODE_URL=$DEFAULT_ADMIN_TEST_MODE_URL
   export AUTH_PAGE_ENCODING_VALUE=$PAGE_ENCODING_VALUE
   export AUTH_SAML_STATE_SIGNING_KEY=$SAML_STATE_SIGNING_KEY
   export AUTH_OAUTH_ID_TOKEN_PRIVATE_KEY_BASE64=$OAUTH_ID_TOKEN_PRIVATE_KEY_BASE64
   go run cmd/auth/main.go
   ```

> **Notes:**
> - The Go services expect prefixed environment variables (`API_*`, `AUTH_*`). The `.env` file contains unprefixed base values, so you need to export them with the proper prefixes.
> - The `DATABASE_URL` uses `postgres` as the hostname for Docker networking. When running services outside Docker, we use `sed` to change it to `localhost`.

### Using npm Locally (Without Docker)

For faster frontend development:

1. **Admin UI:**
   ```bash
   cd admin
   source ../.env
   npm run dev
   ```

2. **App UI:**
   ```bash
   cd app
   source ../.env
   npm run dev
   ```

### Generating API Clients

If you modify the proto files:

```bash
make proto
```

### Regenerating Database Code

If you modify the database schema:

```bash
make queries
```

## Contributing

When submitting changes:

1. Test your changes locally
2. Ensure all services start successfully
3. Run any relevant tests
4. Update documentation if needed
5. Submit a pull request

---

**Happy coding! 🚀**
