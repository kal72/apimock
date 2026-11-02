# Mocklab

A REST API mock builder with workspaces, a visual Node/Table editor, node templates, and configuration dialogs. A single Go server serves templ HTML, frontend assets, the management API, and mock endpoints. PostgreSQL is the primary storage; the application has no in-memory fallback.

## Development documentation

- [Roadmap](ROADMAP.md): feature tables, Done/Partial/Todo status, and upcoming work.
- [Changelog](CHANGELOG.md): available changes and the source for GitHub release notes.
- [Contribution and release guide](CONTRIBUTING.md): documentation requirements for every feature, bug fix, and refactor.
- [Internal deployment](docs/DEPLOYMENT.md): internal host setup, administrator access, callable workspace URLs, and CORS.

All project documentation and code comments must be written in English.

## Stack

- Go **1.27.1**, standard library `net/http`, `slog`, and graceful shutdown.
- Clean architecture: domain → use cases; HTTP and PostgreSQL adapters depend on the inner layers.
- **templ** HTML components, **Alpine.js** editor state, and **HTMX** HTML-based server status updates.
- **Tailwind CSS 4** and component CSS; **Vite** bundles assets and **npm** manages dependencies/builds only.
- PostgreSQL **17**, `pgx/v5`, connection pooling, and JSONB flow storage.
- Multi-stage Docker build with a non-root runtime, a Go binary, and assets embedded using `go:embed`.

Dependencies are locked in `go.sum` and `package-lock.json`. The templ generator is installed as a Go tool using the same version as the templ runtime.

## Running with Docker

```sh
make up
```

Open [http://localhost:8080/app](http://localhost:8080/app). Compose runs Go and PostgreSQL, waits for the database to become healthy, and applies the schema automatically. Host ports are restricted to loopback. There is no Node.js server at runtime.

```sh
make down
```

The `postgres_data` volume persists after `down`.

For a managed internal installation, configure `.env.production` from `.env.production.example` and run `make production-up`. The stack contains only Go and PostgreSQL, uses the existing internal application domain, and exposes mock endpoints at `/public/{workspace-slug}`. No Caddy or public domain is required. See [deployment instructions](docs/DEPLOYMENT.md).

## Local development

Requirements: Go 1.27.1, Node.js 24+ with npm, and PostgreSQL 17 or Compose.

```sh
cp .env.example .env
docker compose up -d db
make install
make dev
```

`make` reads `.env`. When running the binary directly, supply `DATABASE_URL` and optionally `ADDR` through the environment; the binary does not load `.env` itself. The default bind address is `127.0.0.1:8080`.

If port 5432 is already in use, run `DB_PORT=55432 docker compose up -d db` and change the port in your `.env` `DATABASE_URL` to `55432`.

The Compose HTTP port is configurable too. For example, run `APP_PORT=8081 DB_PORT=55432 make up` and open `http://localhost:8081/app`.

For watch mode, run `make watch-assets` and `make watch-templ` in separate terminals. Vite uses **build watch**, not a development server. Go remains the only HTTP server; asset changes require a Go restart because assets are embedded in the binary. The templ watcher restarts the server when templates change.

```sh
make build
DATABASE_URL='postgres://mocklab:mocklab@127.0.0.1:5432/mocklab?sslmode=disable' ./bin/mocklab
```

## Using the editor

1. Create a workspace with a display name and unique slug, such as `Demo Store` / `demo-store`.
2. Add an endpoint, such as `GET /products/:id`. A connected Incoming request → default response (HTTP 200) is provided immediately; a condition is not required.
3. Use **Node** for the visual graph or **Table** for an ordered list of complete request-to-response paths. Each table row describes a path, its condition outcomes, response status/body preview, and effective header count. Matching paths appear before the fallback path.
4. **Node** exposes **Tambah node**, which opens a four-type catalog: Header, Query Params, Body, and Path Params. Selecting a type immediately inserts a connected draft node with one required field and AND matching; it does not open another form. **Table** retains **Tambah kondisi** and its complete editing popup. Both support response/default-response presets.
5. By default, new conditions are appended to the current fallback path so earlier rules keep their priority. In Node mode, selecting an Output before quick-add inserts the check on that connection, so checks from different sources can be chained before success. The default response remains connected; the selected template may update its status/body, which can be reviewed in the Default response tab before applying.
6. In Table mode, use the **Edit** button in a row's **Action** column. There are no inline field editors or separate request/header/body dialogs. The same popup provides Request, Conditions, Response, Default response (when distinct), and Advanced flow tabs.
7. In Node mode, editing a node opens the same complete path popup. Graph lines and per-node connection controls still support relationship editing. Existing header/delay graph steps remain available in the popup's Advanced flow tab.
8. Configure endpoint name/method/path, request examples, typed condition fields/operators/required flags with AND/OR, body Form/Raw settings, response headers/status/JSON body/delay, and fallback response in the popup. Changes to shared request data, conditions, or responses affect every path that uses them; the popup explains this explicitly.
9. Apply the popup to the draft, run a playground test, then use **Simpan perubahan** or **Simpan & aktifkan** to activate it. Canceling the popup leaves the endpoint unchanged. Undo/discard controls remain available for draft changes; Ctrl/Cmd+S saves a valid draft when no dialog is open.
10. Use **Copy base URL** for the workspace URL or **Copy endpoint** for the complete endpoint path. Replace path placeholders such as `:id` before calling the mock. Set `PUBLIC_BASE_URL` to advertise an internal origin, or leave it empty to use the browser origin.

The default row can be edited without adding a condition. Removing a condition through the popup keeps its default/fallback path and any subsequent conditions. Template selection stages all changes locally until applied; it never changes the active API immediately.

Request examples, including JSON/Form/Text modes, are persisted in flow JSONB and loaded into the playground. They are not runtime validation rules. Response delays are limited to 10 seconds and honor request cancellation. Existing standalone header/delay nodes remain supported; new scenario delay values are stored directly on the corresponding response configuration, without a schema migration.

Table projection is limited to 128 paths to bound rendering for highly branching legacy graphs. The original graph is preserved; Node mode remains available for graphs beyond that limit. A default path is resolved independently of the table projection limit.

A save in progress does not overwrite newer local edits or a different selected endpoint. Newer edits remain a draft until saved again.

Example local mock request:

```sh
curl http://localhost:8080/public/demo-store/products/123
```

The `/public` prefix separates mock routes from the UI and management API. A workspace base path is `/public/{slug}`; changing the display name does not automatically change its slug. Explicitly changing the slug changes mock URLs.

Playground input:

```json
{
  "params": {"id": "123"},
  "query": {"page": ["1"]},
  "headers": {"authorization": "Bearer demo"},
  "body": {"user": {"active": true}}
}
```

Legacy conditions retain `params.id`, `query.page`, `headers.authorization`, and `body.user.active` references. Typed nodes use local field names scoped to their source, such as `id`, `page`, `Authorization`, or `user.active`. Header lookup is case-insensitive. Numbers and booleans are compared using their JSON representation (`123`, `true`). See the typed-condition rules below.

## Typed conditions, required fields, and body formats

Each condition node has exactly one source: **Header**, **Query Params**, **Body**, or **Path Params**. Use **Tambah field** to add up to 50 checks within that node. Choose **AND** (all clauses) or **OR** (any clause). Supported operators are `exists`, `equals`, `not_equals`, `contains`, and `empty` (missing or empty).

- `exists` only checks whether the field is present, even if its value is empty or JSON null.
- `required` additionally rejects missing values, null, empty/whitespace-only strings, and empty JSON arrays/objects. Numeric zero and boolean false are valid values.
- Required is evaluated per clause. With OR, another matching clause can still make the group match; use AND or a separate chained node for mandatory global checks.
- A failed required check follows the node's false/fallback branch. Configure the fallback status/body yourself; a **Bad request · 400** response template is provided.
- For repeated header/query/form fields, checks inspect the first value. Path fields must exist in the endpoint route. The Path Params quick-add option is disabled when the route has no parameters.
- Obvious contradictions within AND, such as one field equaling two different values, are rejected. Exact duplicates and equivalent nodes produce warnings rather than blocking edits. Analysis is intentionally limited to these clear cases; it is not a general logical solver. Existing rule priority and graph connections determine overlapping scenarios.

Body nodes offer **Form** and **Raw** modes. Form reads text fields from URL-encoded or multipart bodies without merging query parameters. Uploaded file fields are ignored; multipart requests are capped at 200 parts and the same 1 MiB body limit. Raw JSON uses dotted paths and numeric array indexes (`user.email`, `items.0.id`), with `$` for the JSON root. Raw Text uses `$raw` for the complete text body. JSON, Form, and Text conditions only match their corresponding request format; use the appropriate Content-Type. Malformed JSON with a JSON Content-Type is rejected before flow evaluation.

Example grouped configuration:

```json
{
  "condition_type": "body",
  "match": "all",
  "body_mode": "raw",
  "raw_format": "json",
  "rules": [
    {"field": "email", "operator": "exists", "required": true},
    {"field": "role", "operator": "equals", "value": "admin", "required": true}
  ]
}
```

For the playground, existing JSON inputs still work. Form examples use `"body_mode":"form"` and `"form":{"email":["demo@example.com"]}`. Text examples use `"body_mode":"text"` and `"raw_body":"hello"`. For JSON array/scalar samples, use `"body_mode":"json"` and a JSON-encoded `raw_body`; it takes precedence over `body`.

Legacy single-field conditions remain readable and executable without a database migration. Opening them in the editor presents a one-rule typed group with required disabled, preserving the previous matching behavior until explicitly changed.

## Structure

```text
cmd/server/                     Composition root and server lifecycle
internal/domain/                Entities, flow graphs, and validation rules
internal/usecase/               Repository port, CRUD validation, mock evaluator
internal/adapter/http/          HTTP routing, JSON transport, middleware
internal/adapter/postgres/      pgx repository and embedded SQL schema
internal/web/                   templ components and embedded asset bundle
web/src/                        Alpine editor and Tailwind styling
```

The domain does not import transport, database, or UI packages. Use cases depend on their own repository interface. `cmd/server` injects concrete implementations. UI components include `Sidebar`, `Toolbar`, `EndpointBar`, `FlowCanvas`, `TableEditor`, `ScenarioHeaders`, `ScenarioResponse`, `FlowValidation`, `TestPanel`, and `Dialogs`. Editor state is isolated in `web/src/editor.js`, with graph operations/SVG geometry in `web/src/flow.js` and path projection, ordered condition insertion, and scenario form mapping in `web/src/scenarios.js`.

Each endpoint stores its flow as one JSONB document. Node coordinates are presentation metadata; the evaluator uses only types, configuration, and edges. Table and canvas share Alpine state. CRUD requests use JSON to save the graph atomically; HTMX updates an HTML server-status partial without duplicating graph state.

## Validation and MVP limitations

- Exactly one Request node; every node must be reachable and every path must end at a Response.
- Conditions require exactly one `true` branch and one `false` branch; cycles are rejected.
- Maximum 100 nodes and 50 field checks per condition node; each delay is limited to 10 seconds; the entire request body is limited to 1 MiB.
- Response bodies are static JSON. User-supplied JavaScript/Go execution and template interpolation are not supported.
- Headers are validated; HTTP framing headers, `Set-Cookie`, and server-managed `Access-Control-*` headers cannot be configured.
- Workspace/method/path combinations are unique, including parameter-name variants such as `/:id` and `/:productId`.
- Routes with more static segments take priority.
- Endpoints become active after a valid save; unsaved drafts exist only in the browser.
- Shared administrator HTTP Basic authentication protects management UI/API when `ADMIN_USERNAME` and `ADMIN_PASSWORD` are set; internal mode requires them and a configured `PUBLIC_BASE_URL`; production mode additionally requires HTTPS. Public mock endpoints remain callable without administrator credentials.
- Per-user accounts, workspace roles/tenancy, persistent audit logs, rate limiting, stateful mock CRUD, OpenAPI import, nested groups within a condition, version history, and collaboration are not implemented.

## Internal workspace URLs and browser clients

Set `PUBLIC_BASE_URL` to an HTTP(S) origin without a path, query, or credentials. Internal mode accepts HTTP(S); production mode requires HTTPS. No domain is provisioned automatically. Set `MOCK_CORS_ORIGINS` to exact comma-separated frontend origins, or `*` alone for any origin without browser credentials. Empty allows same-origin requests only. `MOCK_CORS_HEADERS` defaults to `Content-Type,Authorization,X-Api-Key`; add custom headers used by your clients.

CORS applies only to `/public/...`, including browser preflights and mock errors. Management APIs do not inherit mock CORS permissions. Cookie/credential-based cross-origin requests are not enabled; use `credentials: 'omit'` in frontend fetch calls. See [deployment examples](docs/DEPLOYMENT.md).

## Testing

```sh
make check
TEST_DATABASE_URL='postgres://mocklab:mocklab@127.0.0.1:5432/mocklab?sslmode=disable' make test
```

`make test` runs Node.js editor tests followed by Go race tests. Tests cover default connected flows, automatic insertion and fallback wiring, connection edits and cycle rejection, canceled dialogs, persisted request examples, header rows, graph validation, flow branches, delay cancellation, request lookup, route matching, HTTP transport, and PostgreSQL round trips/cascading deletion. Database integration tests are skipped unless `TEST_DATABASE_URL` is supplied; use a dedicated test database when appropriate. Tests create workspaces with random slugs and clean them up afterward.

References: [Go releases](https://go.dev/dl/), [templ tooling](https://templ.guide/quick-start/installation/), [Tailwind + Vite](https://tailwindcss.com/docs/installation/using-vite).

The web interface uses English for navigation, editors, validation, and notifications. Workspace names, endpoint labels, and mock response content remain user-defined.

In Node mode, use the canvas −/+ controls to zoom from 50% to 200% in 25% steps. Click the percentage to reset to 100%. Zoom affects only the view; dragging remains accurate and saved node coordinates are unchanged by zooming.
