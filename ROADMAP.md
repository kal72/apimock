# Mocklab roadmap

Last updated: **2026-11-02**.

This roadmap tracks implementation scope, not promised release dates. No version or Git tag has been released. Available features remain part of an MVP. Internal deployment uses the application host and /public/{workspace-slug}; no domain is provisioned by the application.

## Status definitions

| Status | Meaning |
| --- | --- |
| Done | Implementation is available in the repository; verification limits are noted where applicable. |
| Partial | Part of the scope is available; additional work remains. |
| Todo | Not implemented yet. |

## Foundation and MVP

| ID | Area | Work item | Status | Outcome / limitations |
| --- | --- | --- | --- | --- |
| F01 | Backend | Go 1.27.1 as the only HTTP server | Done | Go serves the UI, assets, management API, and mock API. |
| F02 | Architecture | Separate domain, use cases, adapters, and composition root | Done | Use cases own the repository port; the server injects the PostgreSQL adapter. |
| F03 | Database | PostgreSQL connection pool and JSONB flow storage | Done | Workspaces and endpoints are persistent; database integration has been tested. |
| F04 | Database | Schema migrations with version tracking and transaction locking | Done | Embedded SQL is applied at startup. |
| F05 | UI | templ + Alpine + HTMX + Tailwind components | Done | Sidebar, toolbar, editor, playground, and dialogs are separate components. HTMX provides server status updates. |
| F06 | Build | npm + Vite for asset builds only | Done | No npm server is required at runtime; assets are embedded with go:embed. |
| F07 | Operations | Makefile, multi-stage Dockerfile, and Compose | Done | Binary/image builds passed; container runtime and health endpoint were checked. |
| F08 | Operations | Structured logging, timeouts, and graceful shutdown | Done | Uses slog and the Go server lifecycle. |
| M01 | Workspace | Create, read, update, and delete workspaces | Done | Unique slugs; deleting a workspace cascades to its endpoints. |
| M02 | Workspace | Workspace slug as the mock base path | Done | Uses /public/{slug}; display names and slugs can be changed independently. |
| M03 | Endpoint | REST endpoint CRUD | Done | The API stores method, path, name, and flow; the UI supports creation, selection, method/path/flow updates, and deletion. |
| M04 | Routing | Path parameters and static route priority | Done | Supports /products/:id; routes with more static segments take priority. |
| M05 | Editor | Node and Table modes sharing configuration | Done | Both modes share graph state; Table rows represent complete paths in execution order, with one Action editor per row. |
| M06 | Editor | Node dragging and visual connections | Done | Initial connections are rendered as SVG roots inside HTML templates; lines and per-node connection actions open an editable source/branch/target dialog. New blocks are wired automatically. |
| M07 | Editor | Condition and response templates | Done | Node quick-add offers Header/Query/Body/Path catalogs; Table uses typed condition templates. Response/default presets remain in the shared editor. |
| M08 | Editor | Unified scenario editing popup | Done | One popup edits request/route, conditions, response status/headers/body/delay, fallback, and advanced graph settings. Table has no inline editors. |
| M09 | Runtime | Request, Condition, Headers, Delay, and Response nodes | Done | Conditions support equals, contains, and exists with true/false branches. |
| M10 | Runtime | Read request headers, query, path parameters, and body | Done | Values can be referenced by conditions; request bodies are JSON objects. |
| M11 | Runtime | Response status, headers, and body configuration | Done | Static JSON bodies; interpolation and data generation are not available yet. |
| M12 | Validation | Graphs, connections, JSON, headers, and size limits | Done | Rejects cycles, unreachable nodes, incomplete branches, invalid JSON, and unsafe headers. |
| M13 | API testing | Playground responses and execution traces | Done | Tests unsaved flows; traces are displayed and highlight the canvas. |
| M14 | UX | Unsaved-change indicator and navigation warning | Done | Drafts exist only in the browser; no autosave or persistent drafts. |

## MVP improvements and release readiness

| ID | Priority | Work item | Status | Completion criteria / notes |
| --- | --- | --- | --- | --- |
| R01 | P0 | Editor/Go tests, race detection, and static analysis | Done | Node.js editor tests, go test -race, and go vet passed, including PostgreSQL request-example persistence. |
| R02 | P0 | HTTP smoke tests and Docker runtime checks | Done | Checked CRUD, branched responses, persisted updates, traces, pages, assets, and health. |
| R03 | P0 | Browser interaction, responsive layout, and accessibility QA | Partial | Desktop browser checks passed for initial lines, connection dialogs, Node/Table switching, condition insertion, request/header editing, saving/reloading examples, playground responses, and drag-following lines. Mobile and comprehensive accessibility coverage remain pending. |
| R04 | P1 | Header key–value table editor | Partial | Request and response header dialogs support editable key–value rows with add/remove and duplicate validation. Enabled toggles and direct inline editing outside dialogs are not available. |
| R05 | P1 | Typed string/number/boolean scalar editor | Partial | Status and delay use numeric inputs; conditions use text; a general value-type selector is not available. |
| R06 | P1 | Persist endpoint request header/query/body examples | Done | Request examples persist in flow JSONB and populate the playground; they do not constrain actual incoming requests. |
| R07 | P1 | Rename endpoints in the UI | Done | The Request tab of the row Action popup updates endpoint name, method, and path. |
| R08 | P1 | Mock request log UI and persistent history | Partial | Server logs exist; request lists, details, replay, and database retention are not available. |
| R09 | P1 | Release roadmap and changelog documentation | Done | ROADMAP.md, CHANGELOG.md, CONTRIBUTING.md, and AGENTS.md maintenance rules are available. |
| R10 | P1 | GitHub CI for builds, tests, and PostgreSQL integration | Todo | Run generation, asset builds, database tests, vet, and binary builds in a workflow. |
| R11 | P1 | First GitHub release | Todo | Select a version when ready, complete QA, move Unreleased into a dated version, and create the tag and release notes. |

## Response setup UX

| ID | Area | Work item | Status | Outcome / limitations |
| --- | --- | --- | --- | --- |
| U01 | Response setup | Unconditional default response | Done | Every new endpoint includes one connected default response; edit it from the row Action without adding conditions. |
| U02 | Response setup | Condition-only creation and templates | Done | Node selects a type from a catalog and creates a connected draft immediately; Table uses the complete popup. Each typed node supports multiple fields, AND/OR, required, and response/default presets. |
| U11 | Scenario priority | First-match branch canvas | Done | Independent conditions branch from Incoming Request, priorities are reorderable in Node/Table, default stays last, and response destinations remain editable. Nested legacy graphs retain their original editor. |
| U10 | Table preview | Four-column read-only View | Done | View shows Request, Conditions, effective Response Headers, and formatted Response Body; adapts to two/one columns on smaller screens. |
| U09 | Table flow | One node path per row | Done | Request Path shows endpoint name, HTTP method and URL path. Conditions shows match outcomes and rules without condition names. The separate Node Flow column is removed; distinct paths that converge remain separate rows. |
| U08 | Node editing | Scoped node forms | Done | Node Edit opens only its own Request, Condition, Response, Headers, or Delay form. Table retains the complete path popup. |
| U07 | Appearance | Light / Night / System | Done | Persistent browser preference, automatic system appearance, and Moonlight night surfaces with static stars and a crescent moon. |
| U06 | Visual design | Cloudline theme | Done | Navy sidebar, azure actions, sky gradients, layered static clouds and a softly glowing sun in Light mode, and unified editor/dialog colors. |
| U05 | Node canvas | Bounded canvas zoom | Done | Zoom controls support 50%–200%, 25% steps, reset to 100%, viewport centering, and zoom-aware dragging. |
| U04 | Interface language | English web interface | Done | Navigation, editors, dialogs, validation, notifications, and accessibility labels use English. User-authored content is preserved. |
| U03 | UX | Flow-ordered Table rows with one complete editor | Done | Each row represents a request-to-response path; all field edits are in its Action popup. Nested legacy graphs remain intact and path projection is capped at 128. |
| U04 | UX | Draft save/discard/undo and in-flight save safety | Done | Sticky controls and keyboard save; earlier save responses do not overwrite newer edits or another selected endpoint. |
| U05 | Graph editing | Preserve default and later rules when removing a condition | Done | Condition removal follows the fallback path and prunes only unused matching branches. |

## Typed condition implementation

| ID | Work item | Status | Outcome / limitations |
| --- | --- | --- | --- |
| C01 | Header, Query Params, Body, and Path Params node types | Done | Each node is scoped to one source and can check 1–50 fields. |
| C02 | Multi-field AND/OR with required | Done | Exists differs from nonempty required; zero/false remain valid. Required is per clause, and failures follow the false branch. |
| C03 | Body Form/Raw JSON/Raw Text | Done | URL-encoded and multipart text fields, JSON paths/indexes/root, and whole-text checks; file checks are not supported. |
| C04 | Quick-add without a follow-up form | Done | Node palette creates a highlighted connected draft with defaults; selected outputs can chain further checks before success. |
| C05 | Conflict and duplicate diagnostics | Done | Obvious AND contradictions are blocked; duplicate fields/equivalent nodes warn. General logical overlap analysis remains out of scope. |
| C06 | Legacy condition compatibility | Done | Existing single-field JSONB flows run unchanged; the editor upgrades them to one-rule groups with required disabled. |

## Callable mock API access

| ID | Area | Work item | Status | Outcome / limitations |
| --- | --- | --- | --- | --- |
| P01 | URLs | Separate base URL and endpoint copy controls | Done | Workspace URLs are available in the sidebar; endpoint URLs retain path placeholders until replaced by the caller. |
| P02 | URLs | Configured public origin | Done | PUBLIC_BASE_URL is returned through authenticated settings and used by the editor instead of assuming the browser origin. |
| P03 | Browser clients | Mock CORS and preflights | Done | Exact origins/headers, optional wildcard origins, supported methods, error responses, and explicit OPTIONS handling; credentialed CORS is disabled. |
| P04 | Security | Administrator management access boundary | Done | Shared Basic authentication protects pages/assets/APIs; mock requests are independent. Internal/production modes require credentials and check the configured Host; production additionally requires HTTPS. |
| P05 | Deployment | Internal-domain deployment configuration | Done | Go + PostgreSQL only, /public/{workspace-slug}, managed internal mode, private host binding, optional existing HTTPS ingress, environment examples, Makefile targets, and English instructions. |
| P06 | Deployment | Verify internal hostname and network reachability | Todo | Requires the target internal hostname/server and client-side DNS or hosts resolution; local runtime can be tested without changing machine DNS. |

## Future development

These items form the development backlog; their order and scope can be adjusted.

| ID | Area | Work item | Status | Target capability |
| --- | --- | --- | --- | --- |
| N01 | Flow | Multi-field AND/OR and additional nesting | Partial | Flat AND/OR groups with required and five operators are available in each typed node. Nested subgroups remain future work. |
| N02 | Flow | Set Variable and response interpolation | Todo | Use request values and earlier step outputs in responses. |
| N03 | Data | Dummy data generator and pagination presets | Todo | Generate varied example responses. |
| N04 | Data | Stateful mock CRUD | Todo | POST/PATCH/DELETE requests update a simulated dataset. |
| N05 | Interoperability | Workspace export/import | Todo | Transfer configurations using a validated format. |
| N06 | Interoperability | OpenAPI import | Todo | Generate endpoints and example responses from specifications. |
| N07 | Editor | User-defined templates | Todo | Save and reuse custom configurations. |
| N08 | Versioning | Persistent drafts, version history, and rollback | Todo | Separate draft changes from active versions. |
| N09 | Security | Login and workspace authorization | Partial | Shared administrator HTTP Basic authentication protects management UI/API; per-user accounts and workspace authorization remain unavailable. |
| N10 | Collaboration | Membership, roles, and user data isolation | Todo | Share workspaces with explicit access permissions. |
| N11 | Security | Rate limiting and persistent audit logs | Todo | Control traffic and trace configuration changes. |

## Maintenance rules

- Update related rows in the same change as the implementation or fix.
- Use Partial when only part of the scope is complete; distinguish implementation from browser QA and release readiness.
- Give new items unique IDs and update the document date whenever its contents change.
- Write all documentation and code comments in English.
- Record completed changes in [CHANGELOG.md](CHANGELOG.md); use the roadmap for status and future work.
