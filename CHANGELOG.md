# Changelog

Significant Mocklab changes are recorded here as the source for GitHub release notes. **Unreleased** contains changes that have not been released. No release version or tag exists yet; add a release date only when a release is actually prepared.

Categories: **Added**, **Changed**, **Fixed**, **Refactored**, **Deprecated**, **Removed**, and **Security**. Empty categories may be omitted. See [CONTRIBUTING.md](CONTRIBUTING.md) for entry conventions and the release process. Write entries in English.

## [Unreleased]

### Added

- Add bounded Node canvas zoom (50%–200% in 25% steps), a percentage/reset control, and zoom-aware dragging and node reveal. Zoom preserves the viewport center and does not change saved node coordinates. No migration is required.

- Typed Header, Query Params, Body, and Path Params condition groups with 1–50 fields, AND/OR matching, and per-field required checks.
- Node quick-add catalogs that immediately create connected draft nodes without a follow-up form; selected outputs support chaining checks from different sources.
- Body Form (URL-encoded/multipart text fields), Raw JSON paths/arrays/root, Raw Text, and matching playground/request examples.
- Obvious AND conflict validation, duplicate/equivalent-condition warnings, route-aware path fields, and a Bad request response preset for required failures.

- An ordered Table projection with one row per executable request-to-response path and a single complete editor in each row's Action column.
- Shared Node/Table condition creation with custom, product, authorization, and error-simulation templates, including matching and default-response presets.
- A unified scenario popup for request examples/route, condition outcomes, response headers/status/body/delay, fallback response, and advanced graph settings.
- Response-scoped delay execution with validation and request cancellation, plus bounded legacy-graph path projection.
- Scenario regression tests for priority, default retention, shared graph settings, staged atomic edits, and condition deletion.

- Sticky draft save/discard/undo controls and Ctrl/Cmd+S for valid drafts.
- Regression coverage for unconditional response recovery, conditional-to-direct conversion, response templates through header/delay chains, and edits/navigation during an in-flight save.

- Podman Compose deployment commands for local and managed internal installations.

- Connection editing dialogs accessible from graph lines and per-node actions, with explicit branch/target selection, removal, and client-side cycle rejection.
- Automatic connection wiring when inserting headers/delays and complete matching/fallback paths when adding conditions.
- Incoming request examples persisted in flow JSONB and restored into the playground, including path/query parameters, headers, and JSON body.
- Request and response header key–value editors with duplicate/header validation, and endpoint name/method/path editing through Incoming request.
- Focused Node.js graph/editor regression tests integrated into make test/check.

- Separate workspace base URL and endpoint URL copy actions, including workspace URL access from the sidebar.
- Explicit PUBLIC_BASE_URL configuration and an authenticated settings endpoint so shared URLs use the configured public origin.
- Configurable mock-only CORS with exact origin/header allowlists, optional wildcard origins, preflight handling, and CORS headers on mock errors.
- Shared administrator HTTP Basic authentication for management pages, assets, and APIs while mock endpoints remain independent of administrator credentials.
- A standalone managed internal Compose stack containing only Go and PostgreSQL, configurable host binding, environment examples, Makefile targets, and deployment instructions.
- Configuration and access tests covering production startup requirements, administrator boundaries, CORS/preflight behavior, explicit OPTIONS mocks, and reserved response headers.

- Go 1.27.1 application foundation with separate domain, use cases, HTTP/PostgreSQL adapters, and composition root.
- templ UI components with Alpine.js, HTMX, and Tailwind CSS; npm/Vite bundle assets served by Go.
- Workspace CRUD with unique slugs and the /public/{slug} base path.
- REST endpoint CRUD with flow configurations persisted in PostgreSQL JSONB.
- Mock routing with path parameters and priority based on the number of static segments.
- Visual editor for moving nodes, creating/deleting connections, and saving positions.
- Table mode sharing the graph with Node mode, using ordered request/response path rows and a unified scenario editor.
- Templates for default JSON headers, conditions, success responses, not found, unauthorized, and response delays.
- Node editing dialogs for labels, conditions, JSON headers/body, HTTP status, and delay.
- Request, Condition, Headers, Delay, and Response flow evaluation with equals, contains, and exists operators.
- Condition lookup from request path parameters, query values, headers, and JSON bodies.
- Playground for testing unsaved flows, inspecting responses, and tracing execution paths.
- Unsaved-change indicator and warnings before discarding changes.
- PostgreSQL schema migrations with version tracking and transaction locking at startup.
- Structured logging, health endpoint, timeouts, and graceful shutdown.
- Makefile targets for development, templ generation, builds, tests, and Compose operations.
- Multi-stage Dockerfile with a non-root runtime and Compose services for Go and PostgreSQL.
- APP_PORT and DB_PORT options for configuring Compose host ports.
- Unit/HTTP and PostgreSQL integration tests covering branches, graph validation, headers, delay cancellation, routing, persistence, and cascading deletion.
- README documentation for setup, architecture, editor usage, tests, and MVP limitations.
- ROADMAP.md with implementation status and future work; CHANGELOG.md as the source for release notes.
- CONTRIBUTING.md and AGENTS.md requiring feature, bug fix, refactor, and roadmap updates in the same change.

### Changed

- Standardize the web interface on English, including Node/Table editors, dialogs, validation, notifications, and accessibility labels. Existing user-created content and mock responses are preserved. No migration is required.

- Preserve legacy single-field conditions in runtime; the editor represents them as typed one-rule groups with required disabled. No database migration is required.
- Parse and bound the complete request body before evaluation; JSON Content-Types require one valid JSON value, while Form and Text use separate condition sources.
- Distinguish exists from required: missing/null/blank/empty containers fail required, while zero and false remain valid. Required applies to each clause before AND/OR evaluation and does not automatically select an HTTP error status.

- Rename the editor tabs to Node and Table. Both modes now add conditions only; standalone request/header/response/delay add controls are removed from the UI.
- Replace section-based Table forms with flow rows; move all Table editing, including endpoint name/method/path, into the single row Action popup.
- Append new condition scenarios before the existing default response while preserving earlier rule priority. Removing a condition retains fallback and later rules.

- Open endpoints in Table mode by default and explain the automatic Request → default response when creating an endpoint.
- Remove optional configurations without breaking the path: reconnect around headers/delays and retain the matching path when removing a condition.

- Stage new configuration dialogs without mutating the endpoint until changes are applied; response templates update terminal responses instead of creating unreachable nodes.
- Extract editor state and graph operations into independent JavaScript modules for testing.

- **Breaking change:** move callable mock endpoints from /mock/{workspace}/... to /public/{workspace}/... on the existing application/internal domain. Update consumer base URLs; stored endpoint paths and flows are unchanged. No legacy route alias is provided.
- Add APP_ENV=internal for managed HTTP(S) origins with required administrator credentials and Host/Origin checks; production mode retains its HTTPS requirement.
- Use PUBLIC_BASE_URL directly for internal deployments instead of PUBLIC_DOMAIN and ACME_EMAIL. The managed stack publishes Go on APP_BIND/APP_PORT; no certificate provisioning is required.

- Translate project documentation into English and require English for future documentation and code comments in the project instructions and contribution guide.

### Fixed

- Restore a missing direct success response without requiring a condition or manual edge repair.
- Allow response templates to update terminal responses through an unconditional header/delay chain.
- Preserve newer local edits and endpoint selection when an earlier save request completes.

- Render connection SVG roots inside HTML templates so Alpine can display initial edges and update their paths when nodes move; use separate port positions for condition branches.
- Create a connected Request → Response default when the endpoint creation API receives no flow, so a new endpoint is immediately callable.
- Preserve other condition branches when changing the branch selected in the connection editor.
- Generate node IDs on internal HTTP origins where crypto.randomUUID is unavailable.

### Removed

- Remove the Caddy service, Caddyfile, certificate volumes, and automatic public HTTPS setup from the deployment configuration in favor of the existing internal domain.

### Security

- Require administrator credentials and an HTTPS public origin in production; reject unexpected public hosts and cross-origin management writes.
- Reserve Access-Control-* flow headers for server-managed policy and disable credentialed cross-origin mock requests. **Breaking change:** flows that previously configured those headers must remove them and configure MOCK_CORS_ORIGINS/MOCK_CORS_HEADERS instead.
- Exclude local and production environment files from source control and Docker build contexts while retaining examples.

- Graph validation rejects cycles, unreachable nodes, and incomplete condition branches.
- JSON and header validation, restricted HTTP framing/Set-Cookie headers, and a 1 MiB request payload limit.
- Reject management API writes with a different Origin when an Origin header is supplied.
- Local binding by default and loopback host port mappings in development Compose. Production uses a TLS proxy and a shared administrator boundary; per-user authorization is not available yet.

### Verification notes and limitations

- Canvas zoom verification passed: 42 JavaScript tests and application build. Browser checks confirmed 50%/200% limits disable the corresponding controls, reset returns to 100%, and zoom leaves the endpoint saved. Screenshot: docs/screenshots/canvas-zoom.jpg.

- English-interface verification passed: 40 JavaScript tests, Go race tests, go vet, template/asset build, and browser checks of Table rows, the shared editor, Required guidance, and the Node catalog. Screenshot: docs/screenshots/english-interface.jpg.

- Typed-condition verification passed: 40 JavaScript tests, Go race tests, PostgreSQL integration, go vet, and container build. Desktop QA verified immediate connected-node creation, multi-field required editing, and ordered Table rows with one Action popup. Live requests to the browser-saved endpoint returned fallback 400 for missing/blank required headers and 200 when both were filled. Screenshot: docs/screenshots/typed-conditions.jpg.

- Direct-response desktop QA passed: creating a default 200 endpoint, editing/saving a 201 JSON response without conditions, condition-to-direct conversion, undo, and draft discard. Both the public mock URL and playground returned the saved status/body with a Request → Response trace. The editor has 21 passing JavaScript regression tests.

- Internal-mode HTTP checks passed with a simulated internal Host and real PostgreSQL: /public/{workspace}/... responses, protected management, same-origin CORS, rejected unexpected hosts, and removal of the old /mock route. No client DNS or hosts files were changed.

- Go binary builds, Docker builds, go test -race, PostgreSQL integration, and go vet passed during local verification.
- HTTP smoke tests covered workspace/endpoint CRUD, response branches, persisted response updates, execution traces, health, pages, and assets; container runtime was also checked.
- New access checks passed against a live local Go server and PostgreSQL: administrator authentication, mock access without administrator credentials, public URL settings, CORS/preflights, rejected origins, and management isolation.
- Earlier HTTPS configuration checks have been superseded by the internal deployment configuration. Internal hostname resolution and LAN reachability depend on deployment infrastructure.
- Desktop browser QA verified visible initial connections, connection dialogs, Node/Table switching, condition insertion, request/header edits, saved examples after reload, playground responses, and line updates while dragging. No JavaScript errors were observed in the tested path. Screenshots are available under docs/screenshots; mobile and full accessibility QA remain incomplete.
- Response bodies remain static JSON; per-user authorization, collaboration, stateful data CRUD, and version history are not available yet. Internal deployment configuration is available; target internal DNS and client reachability have not been verified because the actual hostname/server has not been provided.
