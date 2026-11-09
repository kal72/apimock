# Development and architecture

## Local setup

Requirements: Go 1.27.3, Node.js 24+ with npm, and PostgreSQL 17 or a Compose provider.

```sh
cp .env.example .env
docker compose up -d db
make install
make dev
```

Make reads `.env`; the standalone binary does not load it. Default address: `127.0.0.1:8080`. If PostgreSQL port 5432 is occupied, start the database with `DB_PORT=55432 docker compose up -d db` and update `DATABASE_URL` in `.env` to use port 55432.

To change Compose ports, use `APP_PORT=18080 DB_PORT=55432 make up`. For Podman, use the equivalent `podman compose` commands; see [deployment instructions](DEPLOYMENT.md).

## Build and watch

```sh
make build
DATABASE_URL='postgres://mocklab:mocklab@127.0.0.1:5432/mocklab?sslmode=disable' ./bin/mocklab
```

Run `make watch-assets` and `make watch-templ` in separate terminals. Vite builds assets in watch mode; Go is the only HTTP server. Assets are embedded, so asset changes require a Go rebuild/restart. The templ watcher restarts the server for template changes.

Dependencies are locked in `go.sum` and `package-lock.json`. The templ Go tool version matches its runtime dependency.

## Architecture

```text
cmd/server/                 Composition root and server lifecycle
internal/domain/            Entities, graph and template validation
internal/usecase/           Repository ports, CRUD, matching and rendering
internal/repository/        pgx repositories and embedded migrations
internal/handler/           Routes, sessions, middleware and transport
internal/web/               templ components and embedded assets
web/src/                    Alpine state, graph helpers and styling
```

Chi v5.5.0 is limited to the HTTP adapter. Explicit route groups separate public authentication, session-protected management, administrator endpoints and mock CORS. Workspace permissions are declared per route. See [routing and security boundaries](ROUTING.md), including the Go toolchain security verification.

Domain code does not depend on HTTP, PostgreSQL, UI, or third-party generator providers. Use cases own ports, adapters implement them, and the server composes concrete dependencies.

Endpoints store flows as JSONB. Execution follows edges and configuration; coordinates and layout are presentation metadata. Independent scenario priority is encoded by the fallback-chain order rather than a second priority field. Compatible flows are displayed as branches; complex legacy graphs retain their representation.

Node/Table share Alpine state. Configuration writes use the JSON management API; HTMX polls an HTML status partial. templ components separate navigation, editor, dialogs and reference pages. Go embeds generated CSS/JS and serves all runtime HTTP traffic; npm/Vite only manage dependencies and asset builds.

## Verification

```sh
make check
TEST_DATABASE_URL='postgres://mocklab:mocklab@127.0.0.1:5432/mocklab?sslmode=disable' make test
```

Checks run JavaScript tests, Go race tests and vet. Database integration is skipped unless `TEST_DATABASE_URL` is set. Prefer a dedicated test database; isolated authentication schemas and random test workspace identifiers are cleaned up by their fixtures.

Use `make build` and browser checks for UI changes. Verify container builds and runtime for container changes. Follow [CONTRIBUTING](../CONTRIBUTING.md) for documentation, validation and release requirements.

## Deployment and limits

The non-root container runs a standalone Go binary. PostgreSQL is required; there is no in-memory fallback. Migrations run at startup with transaction/advisory locking. See [deployment instructions](DEPLOYMENT.md) and [account/session details](ACCESS_CONTROL.md) for bootstrap, security boundaries, CORS and upgrades. Feature limitations are tracked in the [roadmap](../ROADMAP.md).
