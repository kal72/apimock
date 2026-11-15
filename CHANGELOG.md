# Changelog

Significant Mocklab changes are recorded here as the source for GitHub release notes. **Unreleased** contains changes that have not been released. No release version or tag exists yet; add a release date only when a release is actually prepared.

Categories: **Added**, **Changed**, **Fixed**, **Refactored**, **Deprecated**, **Removed**, and **Security**. Empty categories may be omitted. See [CONTRIBUTING.md](CONTRIBUTING.md) for entry conventions and the release process. Write entries in English.

## [Unreleased]

### Added

- Added GitHub Actions CI/CD workflows for direct Podman container deployment on self-managed Linux VPS servers for both `dev` and `prod` application environments without requiring Docker, Podman Compose, or host deployment directories (`DEPLOY_PATH`). The CI workflow (`.github/workflows/ci.yml`) validates frontend tests, Go tests with race detection, `go vet`, and direct Podman container image builds. The CD workflow (`.github/workflows/cd.yml`) automates deployments on push strictly to the `main` branch (deploying image tag `:dev` to the `dev` environment), while production (`prod`) deployments are strictly manual via `workflow_dispatch` with customizable version tags (e.g. `v1.0`, `v1.0.1`) and branch selection. Supports both password authentication (`SSH_PASSWORD` via `sshpass`) and key-based authentication (`SSH_PRIVATE_KEY`). The CD pipeline builds and pushes container images directly to GitHub Container Registry (GHCR), then connects to the VPS via SSH to pull the versioned image, remove the previous container, and restart the application container statelessly with directly injected environment variables without writing configuration files or code to the VPS host. Added comprehensive guide in `docs/CI_CD.md`.

- Added private, per-user environments shared across that user's workspaces. Users can create, rename, and delete environments and typed string, number, boolean, or secret variables. Secret values are masked by the management API and UI after saving. The selected environment is available to Playground and Dynamic JSON response previews through `{{ environment.variable_key }}`; it is deliberately unavailable to public `/public/{workspace}` mock requests to prevent private-value exposure. Added migration `004_user_environments.sql` and docs/ENVIRONMENTS.md. Existing workspace flows and public mock behavior are unchanged.

- Added authenticated V2 static design previews at /app/v2 for Endpoints, Environments, Mocks, Logs and Settings. Shared templ components and scoped Graphite Blue/Cloudline styles provide Light/Night themes, Node/Table views and local sample interactions. Preview pages do not call application APIs or persist changes; proposed capabilities are illustrative. Existing UI and database schema are unchanged. See docs/UI_V2.md for migration boundaries.

### Changed

- Streamlined Table mode (`TableEditor`) by removing the redundant "Headers" column, consolidating response headers into an inline badge within the Response column, and rebalancing column widths (`Priority`, `Request path`, `Conditions`, `Response`, `Action`) to match Blueprint V2.

- Refactored form inputs, layout, and styling across the Environments page (`EnvironmentsPage`): replaced the cramped environment creation card with a dedicated `.v2-env-create-card` containing a full-width input row, introduced an interactive inline environment rename form in the header (`✎ Rename`) allowing direct name updates with save and cancel controls, and redesigned the right-side variable form using a semantic `.v2-form-group` system (`.v2-label`, `.v2-input`, `.v2-select`, and `.v2-form-hint`) with type-specific value inputs (string, number, boolean, secret) without styling clashes.

- Added prominent Workspace Base URL access and clipboard actions: added a 1-click `.workspace-base-url-pill` directly below the workspace switcher in the sidebar, a "Copy base URL" action with a concise description inside the workspace settings dropdown panel (without cluttering the menu with the raw URL), and an inline "Copy URL" button beside Public Base URL on the Workspace Settings page.

- Expanded the "Generator docs" navigation and documentation into a comprehensive "Docs & Reference" guide (`/docs/generators`): renamed the sidebar navigation item from `Generator docs ↗` to `Docs & Reference ↗`, and expanded the reference documentation with structured sections for Mocklab Architecture, Dynamic JSON & templating syntax (`request.*`, `environment.*`, `generate.*`, default fallbacks), Priority-based Scenario Routing & Condition Matching (`ALL`/`ANY`, operator catalogue, terminal fallback), Environment & Secret protection, and Public Mock Endpoints URL format and CORS testing.

- Overhauled the Response Scenarios table in Table mode (`TableEditor`) to follow the Blueprint V2 Graphite Blue + Cloudline design system (`.v2-table.scenario-table`). Added an upgraded card header with a clear execution order description and a styled `.v2-primary` `+ Add condition` action; modernized the Priority column with tabular-numeral numbered chips (`#1`, `#2`), dedicated `Default` fallback chips, and vertical micro arrow buttons for reordering; restructured the Request Path column with method badge pill and route path; redesigned the Conditions column with structured rule pills (`◇ Matches / Does not match`), condition summaries, and clear fallback indicators; modernized the Response column with color-coded HTTP status badges (`2xx` green, `3xx` blue, `4xx` amber, `5xx` red), response mode badges (`Dynamic` / `Static`), response delay indicators, and card-style monospace body snippets with 1-click clipboard copy; upgraded header counts into distinct badges; refined row actions (`View` and `Edit ↗`) using `.v2-btn-action`; and updated empty and truncation states to match V2 styling.

- Overhauled the Key-Value variables table across the Environments page and modal to match the Blueprint V2 column layout (`Key | Value | Type | Action`). Placed the Value column directly adjacent to the Key, added 1-click clipboard copying for Dynamic JSON expressions (`{{ environment.key }}`) with a dedicated `{{ }}` badge, introduced rich type-aware formatting for values (monospace code with 1-click copy for strings, emerald/slate status badges for booleans, tabular numeral chips for numbers, and masked `••••••••` indicators with security badges for secrets), refined action buttons (`Edit` and `✕ Delete`), highlighted active editing rows (`.row-editing`) with a blue accent edge, and enhanced Response scenario body previews in Table view with inline copy and monospace truncation styling.

- Converted sidebar navigation for Environments and Settings from modal popup overlays into dedicated full-page views following the Blueprint V2 architecture (`.v2-content`). The Environments page features a responsive environment cards grid (`.v2-environments`), inline new environment creation, and a two-column split (`.v2-split`) with a filterable variables table and an inline add/edit variable form. The Settings page provides tabbed navigation (`General`, `Members`, and `Overview & Defaults`) for managing workspace details, access summaries, danger zones, collaborator roles, and mock base configurations without modal obstruction.

- Enhanced Visual Flow canvas scrolling and viewport interaction: added `overscroll-behavior: contain` to prevent mouse wheel and trackpad scroll chaining to the outer page, custom grab and grabbing cursors with click-and-drag panning on empty canvas areas, smooth multi-axis touch and trackpad scrolling, custom thin scrollbars styled with `--v2-*` tokens, generous boundary margins (`Math.max(1200, ...)`), and an interactive zoom percentage button that centers and resets the canvas view to origin on click.

- Refactored the main application (`/app`) UI/UX to adopt the Blueprint V2 Graphite Blue + Cloudline design system without altering runtime logic or adding unimplemented V2 preview features (`Mocks`, `Logs`, `Mock defaults`). Introduced the compact 48px `.v2-header` with breadcrumb, `Saved`/`Unsaved` pill status, pill Sun/Moon theme toggle, account popover, and `.v2-sky` decoration; the 63px `.v2-route-toolbar` with inline endpoint title, method/path badge, copyable base URL, and `Node` / `Table` segmented switcher; unified `.v2-sidebar` navigation (`Endpoints`, `Environments`, `Settings`, `Generator docs ↗`) above the scrollable endpoint list; minimalist 240px × 90px V2 visual flow node cards (`.v2-node`) positioned on the V2 coordinate grid (`Incoming request` at `(10, 175)`, conditions at `x = 395`, responses/default at `x = 775`) with port dots (`.v2-port-in`, `.v2-port-out`) and floating priority chips (`.v2-priority`); a compact auto-dismissing floating toast notification (`.v2-toast`); an actionable-only dismissible condition priority warning banner; and relocated the Playground Environment Switcher (`ENV` selector + `Manage` button) into the 2-column Request Playground header bar. Replaced legacy full-screen SVG background layers and hard-coded light/dark rule overrides with a zero-JS-reflow CSS Custom Properties architecture (`--v2-*` on `:root` and `:root[data-theme=night]`) and `<meta name="color-scheme" content="light dark"/>`.

- Replaced the "Save changes" primary button on the top toolbar with an "● Unsaved" status indicator positioned beside the Environment dropdown selector, matching the quiet style and placement of "✓ Saved" when no changes are pending. Saving remains available through the bottom draft bar, shortcut (`Ctrl+S` / `Cmd+S`), or by clicking the unsaved indicator.

- Repositioned the "✓ Saved" indicator in the top toolbar to the left of the Environment dropdown selector, keeping save status feedback directly alongside the workspace breadcrumb and endpoint context.

- Redesigned the scenario path view modal (`path-view-modal`) in Table mode into a balanced two-column inspector layout featuring a prominent hero header with HTTP status badge, route method pill, full URL banner with copy action, structured routing conditions, example request payloads, cascaded response headers, and response body preview with copy support.

- Isolated application shell viewport scrolling: locked the root document (`html, body, .app-shell`) to 100vh with `overscroll-behavior: none` and designated `<main>` as the sole independent scroll container, eliminating window-level rubber-band bouncing and top-space detachment when scrolling to the bottom.

- Upgraded the endpoint counter in the sidebar to a styled badge pill with tabular numerals, dedicated dark/night theme border styling, and dynamic active filter count reporting (e.g. `2 / 5`) when searching endpoints.

- Redesigned the Environments management dialog into a Master–Detail two-column layout with an environment navigator list on the left and an active environment variables table with color-coded type badges (String, Number, Boolean, Secret), masked secret values, and inline add/edit variable controls on the right.

- Relocated the Environment dropdown selector and inline **Manage** button to the top toolbar next to the theme switcher, keeping the sidebar focused on workspace selection and endpoint navigation.

- Fixed the desktop sidebar layout to remain pinned to the viewport height (`position: sticky`), making only the endpoint list independently scrollable while the brand, workspace switcher, and footer actions remain fixed.

- Refined the V2 header to follow the original concept: compact breadcrumb and Saved/theme/account row, right-aligned sky decoration, functional local account popover, and a separate endpoint/base-URL/Node–Table toolbar. Unified the endpoint list into the main sidebar to preserve toolbar width and moved the preview notice to the footer; all content remains static.

- Open the endpoint editor in Node mode by default. Table remains available through the mode switch; flow data is unchanged.

### Fixed

- Streamlined and fixed the condition/scenario editor popup (`modal === 'scenario'`) in Table mode (`TableEditor`) so it only displays forms relevant to the selected table row. Removed the out-of-place `Request` tab (endpoint route and request example form), the duplicate `Default response` tab inside condition popups, the blank `Advanced flow` tab on priority-based endpoints, and the duplicate `Condition template` dropdown when adding a new condition; updated `editScenario` so clicking `Edit ↗` on a condition row opens directly on the `Condition` tab while clicking `Edit ↗` on the `Default` row opens directly on the `Default response` form without a redundant tab bar; and reorganized the condition and response fields into a cleaner multi-column layout.

- Fixed layout distortion and overlapping cell artifacts in Table mode (`TableEditor`) where a vertical box appeared to cover parts of the table. Replaced direct `display: flex` declarations on `<td>` elements (`.scenario-route-cell` and `.scenario-response-cell`) with semantic table cells containing inner flex wrapper containers (`.scenario-route-wrap` and `.scenario-response-wrap`), eliminated obsolete `nth-child` width and `max-width` constraints from legacy `app.css` that conflicted with the 5-column layout, allowed natural wrapping with `white-space: normal` on `.scenario-table td`, and ensured dedicated column width allocations (`.scenario-th-priority`, `.scenario-th-route`, `.scenario-th-conditions`, `.scenario-th-response`, and `.scenario-th-actions`).

- Fixed deployment startup failure when administrator credentials are not configured. Made `ADMIN_USERNAME` default to empty in `cd.yml` and `deploy/podman-deploy.sh`, allowing Mocklab to boot successfully into web onboarding mode on first launch when neither `ADMIN_USERNAME` nor `ADMIN_PASSWORD` is provided.

- Resolved an issue in the Environment Management modal where environment variables could not be created or edited. Preserved reactive getters (`activeEnvironment`, `environmentVariables`, and `isEditingVariable`) across Alpine state composition, ensured environment variable forms auto-select the active or first available environment, introduced dedicated input types for boolean, number, string, and secret kinds, and surfaced inline validation/server error notices directly within the variable form.

### Refactored

- Restructured internal packages to align with idiomatic Clean Architecture conventions by replacing `internal/adapter/postgres` and `internal/adapter/http` with top-level `internal/repository` and `internal/handler`.
- Decoupled `internal/handler` from the underlying persistence layer by exposing usecase service operations for workspaces, endpoints, environments, and memberships on `usecase.Service`.
- Added constructor functions `NewService` and `repository.New`, named interface parameters on `WorkspaceRepository`, standardized error messages in English, and replaced raw numeric status codes with standard `http.Status*` constants.
- Migrated HTTP routing to pinned Chi v5.5.0 with public-authentication, session-protected management, administrator and public-mock groups. templ remains on standard net/http; domain/use-case dependencies and database schema are unchanged. Preserved normal route URLs, methods, session/CSRF and CORS behavior, decoded wildcard paths, query-preserving canonical redirects and 405 Allow headers.
- Replaced URL-substring workspace policy inference with explicit Viewer/Editor/Owner middleware. Missing workspace parameters and unknown permission roles deny access. Corrected Viewer HEAD reads to use Viewer permission rather than Editor permission. Added routing/authorization regression coverage and routing documentation.
- Preserved public preflight handling before normalization and kept unknown mock methods outside management authentication.
- Canonical redirects run before destination session checks; a redirected management URL still enforces its complete authentication/authorization policy. No RealIP, automatic method rewriting, or blanket path/CORS middleware was enabled.

### Security

- Upgraded the Go toolchain and container build image from 1.27.1 to 1.27.3, resolving nine reachable standard-library advisories. govulncheck v1.9.0 reports no vulnerabilities after the upgrade. No database migration or API configuration change is required; local development now requires Go 1.27.3. Details are documented in docs/ROUTING.md.


### Generator documentation page

- Verification: 67 JavaScript tests, Go race tests, PostgreSQL-backed authentication checks, vet, standalone build, and desktop Light/Night browser checks passed. Confirmed catalog counts, search, example expansion, copy feedback, and login protection.

- Added an authenticated, searchable generator reference at `/docs/generators`, linked from the sidebar and Dynamic JSON editor. Four available generators include copyable JSON examples; the 25 proposed categorized generators are explicitly marked Planned with no runnable copy action. Added quick-start, type/seed/header rules, troubleshooting, responsive layout and Light/Night styles.


### Documentation

- Added refreshed viewport and full-page main screenshots showing the default Node editor in docs/img/main-node-mode.jpg and docs/img/main-node-mode-full.jpg. The full-page capture includes the playground and footer, with canvas zoom at 75% to show every node.

- Reorganized README around description, screenshot, technology/dependencies, architecture, configuration and startup, with Contributions as the final section. Added aligned Docker/Podman and local-development setup instructions while keeping detailed usage in dedicated guides.

- Shortened README to the product overview, screenshot, quick start, development commands and documentation links. Moved editor/condition guidance, architecture/development details, and account/session information into dedicated guides; removed repeated styling history and corrected obsolete editor labels/condition-insertion descriptions.

- Added a main workspace screenshot near the top of README, showing Table response scenarios and the request playground.

- Standardized the proposed fake-data catalog and examples on the existing `generate.*` namespace (for example, `generate.person.full_name`). Existing generator names remain unchanged; categorized generators are still planned, not implemented.

- Added a proposed fake-data implementation plan selecting gofakeit/v7 for evaluation, 25 allowlisted generators, delivery phases, seed/version contracts, provider boundaries and acceptance checks. This is a plan only; no fake-data library or runtime feature has been installed.

- Added a Dynamic JSON usage guide with Static/Dynamic mode selection, Node/Table workflows, type preservation, defaults, source reference, generators/seeds, header templates, preview versus full-flow tests, curl examples, limits, and troubleshooting. Linked it from README. Verified all 13 JSON examples, validated 10 dynamic templates against the current parser, checked quick-start rendered output, and checked guide links/anchors.


### Static and Dynamic JSON responses

- Verification: 67 JavaScript tests, Go race tests and vet, PostgreSQL integration (including mode/seed persistence), standalone build, and desktop browser checks passed. Browser checks covered literal Static templates, variable insertion, typed Dynamic output, default and missing-field behavior, saving, and real public mock execution.

- Added opt-in Dynamic JSON rendering for request path/query/header/JSON/form/raw-body variables, JSON defaults for missing fields, type-preserving whole-value bindings, sentence interpolation, and dynamic response header values. Static JSON remains the default and retains literal template text and original formatting.
- Added UUID v4, live UTC timestamp, bounded integer, and JSON choice generators. Optional seeds reproduce random values independently of key order; identical generator expressions reuse values within one response, including headers. Timestamps always remain live.
- Added an Insert variable picker and isolated rendered preview in the shared Node/Table response editor, mode indicators, and template labeling in View. Existing status/delay and first-match/default behavior are preserved. Built-in response presets explicitly select Static JSON.
- Added bounded, allowlisted template validation, safe JSON serialization, rendered-header validation, and descriptive Playground errors for missing variables. Public render failures return generic 500 responses. No SQL migration is needed; optional mode/seed fields use existing flow JSONB. Set Variable nodes, Faker datasets, list generators, pagination, and stateful storage remain outside this increment.


### Workspace navigation refinement

- Verification: standalone build and desktop browser checks passed in Light and Night, including opening Edit/Members, keyboard entry with Arrow Down, Escape, and outside-click dismissal.

- Replaced the crowded workspace action row with a single settings button beside the workspace selector. The popover groups Edit workspace and Manage members, separates Delete workspace, and keeps access level as quiet read-only information. Viewer/Editor accounts do not see owner controls. Supports Escape, outside-click dismissal, keyboard entry, and focus leaving the panel. Existing deletion confirmation remains in place.


### Condition conflict diagnostics

- Verification: 64 JavaScript tests, Go race tests and vet, standalone build, and desktop browser checks passed. Browser checks confirmed warnings in Node/Table, real first-match diagnosis, priority changes, and saving valid overlapping scenarios.

- Added advisory checks for equivalent conditions, conditions covered by an earlier priority, and potential overlap in independent scenario flows. Node/Table and staged condition editors share diagnostics; warnings never reorder nodes, block valid overlapping branches, or change default execution. Priority warnings above the editor are collapsible.
- Added Playground explanations showing the selected response, per-condition match results, per-field pass/fail, body format mismatches, and whether a node was actually visited. Later conditions are checked diagnostically without changing first-match execution. The existing result fields remain available alongside additive `conditions` and `response_id` fields in `/api/test`.
- Analysis is conservative: unsupported legacy conditions and nested graphs do not receive priority claims; warnings are capped to keep the editor usable. Arbitrary JSON field dependencies and collective coverage by several earlier branches are not proven. Existing contradiction and Required validation remains enforced by both UI and server.


### Workspace access management

- Verification: Go race tests, vet, PostgreSQL integration tests (including permission boundaries, legacy migration and concurrent owner protection), 57 JavaScript tests, and standalone build passed. Desktop browser checks covered member creation and read-only Table/Node controls.
- Added workspace membership with Viewer (browse/test), Editor (endpoint CRUD), and Owner (workspace and member administration) permissions, enforced by management APIs and reflected in the editor UI. Administrators retain recovery access to all workspaces.
- Added an owner-only Members dialog for assigning existing accounts by username, changing roles, and removing members. Concurrent membership changes and user deletion/deactivation cannot remove the last active workspace owner.
- **Breaking change:** regular users no longer have global workspace management access. Migration `003_workspace_members.sql` assigns legacy workspaces to the earliest active administrator; an owner/admin must explicitly grant other accounts access. First-admin setup claims existing unowned workspaces when upgrading an installation without accounts. Public mock URLs and flow configurations are preserved.


### Added

- Add a compact Table footer showing the condition count and a default-response ordering hint, with a thin divider and neutral Light/Night surfaces.

- Add PostgreSQL-backed user accounts, session login/logout, admin/user roles, first-admin setup/bootstrap, admin-only user management, and own-password changes. Admins can create/edit/deactivate/delete accounts and reset passwords; shared workspace data is preserved.

- Add first-match scenario branches using the existing Incoming Request, Condition, and Response nodes. Conditions have reorderable priorities in Node and Table; default response is always last and cannot be prioritized. Priority changes update the persisted execution order and are undoable.

- Add a read-only Table View action with four detail columns: Request, Conditions, effective Response Headers, and formatted Response Body. Include status, accumulated delay, request examples, and draft indication; no API request is executed. Smaller screens use two or one column.

- Add Light, Night, and System appearance modes with a saved browser preference and automatic device-theme updates. Moonlight uses dark navy surfaces, static stars, and a softly lit crescent moon outside the editor content. No data migration is required.

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
- Management authentication keeps mock endpoints independent of administrator credentials; session login supersedes the earlier shared Basic authentication.
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

- Make Table View/Edit actions compact 29px soft buttons: blue for View and amber for Edit, with matching Light/Night hover and focus behavior. Keep visible labels and the primary Add condition action.

- Simplify Table mode to a Response scenarios heading with a small count and Add condition action. Remove repeated flow-summary and footer text; use uniform row backgrounds, thin dividers, subtle hover, and a Default badge without a separate default-row tint. Existing columns, priority, View, and Edit behavior remain unchanged.

- Replace theme dropdowns with an accessible Light/Night switch in the top bar and login/setup pages. Remove System from the UI; existing System preferences resolve to the device theme on load, while explicit Light/Night choices remain saved.

- Keep the Light/Night/System selector directly in the top bar and show the account first name beside its avatar. Account actions remain grouped in the popover.

- Simplify the top bar to workspace breadcrumbs, a quiet saved indicator (or primary Save action for drafts), and one account popover. Group appearance, admin user management, password changes, and sign out inside the account menu with keyboard/Escape and outside-click support.

- Reuse Cloudline daylight clouds/sun and Moonlight stars/crescent artwork on login and initial setup, with static decoration behind the readable authentication form.

- Lower the minimum password length to 8 characters for initial setup, administrator bootstrap, user creation/reset, and own-password changes. Keep the 1024-byte maximum and existing password hashes/sessions.

- **Breaking change:** the running server replaces shared HTTP Basic authentication with session login, including development. Existing workspaces remain shared by authenticated users. Migration 002_identity.sql adds users and sessions automatically. ADMIN_USERNAME/ADMIN_PASSWORD seed only an empty user table and never reset existing accounts; management API clients must retain login cookies and send X-CSRF-Token on writes.

- Enrich Light mode with layered cloud artwork, a softly glowing sun and subtle rays, plus a warmer sky gradient. Decoration remains static behind content; Night styling and editor behavior are unchanged.

- Display independent condition scenarios as direct Incoming Request branches with no visible No edges, and one Table row per scenario plus default. Keep scoped node editing, editable response destinations, and bounded zoom. Existing linear fallback graphs use the new presentation without changing evaluation semantics; nested graphs and legacy header/delay chains retain their original graph editor. The optional flow.layout field stores canvas presentation; no database migration is required.

- Simplify the Table Conditions column by hiding condition names; retain match outcomes and rules.

- Remove Node Flow from Table mode. Conditions now display each condition name, match outcome, and rules while preserving one request-to-response path per row.

- Present each Table row as one complete request-to-response node chain with explicit Yes/No branches. Request Path now shows the endpoint name, HTTP method, and URL path. Distinct paths remain separate even when they share a response; Action Edit retains full path editing.

- Apply the Cloudline visual theme with a navy sidebar, azure actions, pale sky gradients, static cloud silhouettes, and consistent Node/Table/dialog styling. Preserve existing flows and editor behavior; no migration is required.

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

- Scope Node Edit to the selected node instead of opening the complete path editor. Preserve other nodes and connections; Table Action Edit retains the full path popup. No migration is required.

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

- Hash passwords with individually salted PBKDF2-HMAC-SHA256 (600,000 iterations), store only hashed session tokens, expire sessions after 12 hours, use HttpOnly/SameSite cookies with Secure on HTTPS, and require session-bound CSRF tokens. Apply bounded in-process sign-in throttling, revoke sessions after identity changes, serialize role changes to preserve the last admin, and prevent self-deletion/demotion/deactivation.

- Require administrator credentials and an HTTPS public origin in production; reject unexpected public hosts and cross-origin management writes.
- Reserve Access-Control-* flow headers for server-managed policy and disable credentialed cross-origin mock requests. **Breaking change:** flows that previously configured those headers must remove them and configure MOCK_CORS_ORIGINS/MOCK_CORS_HEADERS instead.
- Exclude local and production environment files from source control and Docker build contexts while retaining examples.

- Graph validation rejects cycles, unreachable nodes, and incomplete condition branches.
- JSON and header validation, restricted HTTP framing/Set-Cookie headers, and a 1 MiB request payload limit.
- Reject management API writes with a different Origin when an Origin header is supplied.
- Local binding by default and loopback host port mappings in development Compose. Production uses existing HTTPS infrastructure and a session/role boundary; workspace ownership/tenancy is not available yet.

### Verification notes and limitations

- Table footer build/container and desktop browser verification passed: the condition count and default-order hint render below the table. Screenshot: docs/screenshots/table-footer.jpg.

- Compact Table action build and browser checks passed in Light/Night: View/Edit measure 29px high, have blue/amber surfaces, and open their original dialogs. Screenshot: docs/screenshots/table-compact-actions.jpg.

- Table clarity build/container verification and desktop browser review passed in Light/Night. Computed backgrounds match across normal/default rows in both themes, the compact heading/count is present, and repeated summary/footer elements are absent. Screenshots: docs/screenshots/table-quiet-light.jpg and docs/screenshots/table-quiet-night.jpg.

- Light/Night switch verification passed: 56 JavaScript tests, build, and browser checks of accessible switch state and both theme transitions. Screenshot: docs/screenshots/theme-switch.jpg.

- Revised top-bar build and browser QA passed: the theme selector is visible outside the account menu, the trigger displays the first name, and the menu contains only account/admin actions. Screenshot: docs/screenshots/topbar-theme-name.jpg.

- Compact top-bar build and browser QA passed in Light/Night: account menu, Arrow Down focus, Escape dismissal, appearance changes, admin navigation, and Save/Saved transitions after persisting a draft. Screenshots: docs/screenshots/account-topbar.jpg and docs/screenshots/account-topbar-light.jpg.

- Login sky-theme build and browser review passed in Light and Night. Screenshots: docs/screenshots/login-light.jpg and docs/screenshots/login-night.jpg.

- Identity verification passed: 56 JavaScript tests, Go race tests, PostgreSQL integration in isolated schemas, and go vet. Tests cover initial setup, bootstrap without reset, login/logout, CSRF, admin/user API separation, expiration, session revocation, stale credential rejection, public mock access, and concurrent last-admin protection. Browser QA in a disposable database verified admin login, profile editing, user-only UI, logout, and persisted sessions after restart. Screenshots: docs/screenshots/login.jpg and docs/screenshots/user-management.jpg.

- Daylight artwork build and desktop browser review passed: Light mode shows layered clouds, soft sunlight, and readable solid table/editor surfaces. Screenshot: docs/screenshots/daylight-theme.jpg.

- Scenario-priority checks passed: 53 JavaScript tests, Go race tests, go vet, and application build. Browser QA covered Node/Table priority controls, undo, and persisted order after reload. A real mock with two matching conditions changed from 201 to 202 after raising the second condition; unmatched requests retained default 404. Screenshot: docs/screenshots/scenario-priority.jpg.

- Table View checks passed: 48 JavaScript tests, application build, and desktop browser verification of all four read-only columns, formatted JSON, and unchanged saved state after closing View. Screenshot: docs/screenshots/table-view.jpg.

- Named-condition Table checks passed: 46 JavaScript tests, application build, and browser verification of the removed Node Flow column and condition names, outcomes, and rules. Screenshot: docs/screenshots/table-named-conditions.jpg.

- Table node-flow verification passed: 46 JavaScript tests, application build, and desktop browser review of endpoint name/path, single-line node chains, and distinct Yes/No rows. Wide chains scroll horizontally without overlapping adjacent columns. Screenshot: docs/screenshots/table-node-flow.jpg.

- Scoped-node verification passed: 45 JavaScript tests, application/container builds, and browser checks of Condition, Request, and Response node forms plus the full Table popup. Screenshot: docs/screenshots/scoped-node-editor.jpg.

- Appearance verification passed: 44 JavaScript tests, application build, and desktop browser checks of Light/Night switching, reload persistence, Node canvas, Table surfaces, and response dialogs. System-theme changes and blocked storage are covered by unit tests. Screenshot: docs/screenshots/moonlight-theme.jpg.

- Cloudline application build and desktop browser review passed for Node canvas, zoom controls, Table layout, and the shared response popup. Screenshot: docs/screenshots/cloudline-theme.jpg. Full mobile/accessibility audits remain pending.

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
- Response bodies remain static JSON; workspace ownership/tenancy, collaboration, stateful data CRUD, and version history are not available yet. Internal deployment configuration is available; target internal DNS and client reachability have not been verified because the actual hostname/server has not been provided.
