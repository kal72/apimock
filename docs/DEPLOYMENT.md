# Internal deployment

Mocklab uses the application's existing internal domain and the `/public/{workspace-slug}` prefix. It does not create a domain per workspace or require a public internet domain, Caddy, or automatic certificate issuance.

Example using an internal hostname:

```text
Application: http://apimock.internal:8080/app
Workspace:   http://apimock.internal:8080/public/demo-store
Endpoint:    http://apimock.internal:8080/public/demo-store/products/123
```

The workspace segment uses its unique slug. A display name such as `Demo Store` can use the slug `demo-store`.

## Local development

Run the development stack normally:

```sh
APP_PORT=18080 DB_PORT=55432 make up
```

Open `http://localhost:18080/app`. Workspace base URLs automatically use the current browser origin, for example `http://localhost:18080/public/demo-store`. No hostname or DNS changes are required for localhost.

To use an internal hostname, configure existing internal DNS or add a hosts entry on each client. For a server running on the same machine:

```text
127.0.0.1 apimock.internal
```

For other clients, resolve the hostname to the server's private IP instead. A hosts entry or DNS record resolves the hostname only; it does not configure ports, firewall rules, or TLS. No machine hosts files or DNS records are modified automatically.

## Managed internal installation

Copy the environment example:

```sh
cp .env.production.example .env.production
chmod 600 .env.production
```

Configure these values:

| Variable | Purpose |
| --- | --- |
| APP_ENV | Use internal for direct internal HTTP/HTTPS origins; production still requires an HTTPS origin. |
| PUBLIC_BASE_URL | Application origin only, such as http://apimock.internal:8080. Do not append /public or a workspace path. |
| APP_BIND | Host interface to publish Go on; defaults to 127.0.0.1. Set a private interface IP for direct LAN access. |
| APP_PORT | Published Go port; defaults to 8080. Include it in PUBLIC_BASE_URL unless your existing infrastructure serves a standard port. |
| ADMIN_USERNAME | First administrator bootstrap username (3–64 lowercase letters/numbers/dots/underscores/hyphens). |
| ADMIN_PASSWORD | First administrator bootstrap password, at least 8 characters and at most 1024 bytes without line breaks. Used only while the user table is empty. |
| POSTGRES_PASSWORD | Required independent database password; use at least 32 URL-safe characters because it is embedded in a PostgreSQL URL. |
| MOCK_CORS_ORIGINS | Exact comma-separated frontend origins. Empty allows same-origin browser calls; * alone allows any origin without browser credentials. |
| MOCK_CORS_HEADERS | Allowed preflight headers; defaults to Content-Type,Authorization,X-Api-Key. |

Generate independent strong administrator and database passwords before starting. Never commit `.env.production`. Internal mode requires administrator credentials and a configured origin and enforces the configured Host and management Origin boundaries. HTTP does not encrypt login credentials, session cookies, or request data; use your existing internal HTTPS infrastructure when transport confidentiality is required. Go does not terminate TLS in this setup.

```sh
make production-up
make production-down
```

These commands use the standalone `compose.production.yaml` and the separate `mocklab-production` project. The stack contains only Go and PostgreSQL. Only Go publishes a host port; PostgreSQL has no host port and stores data in a persistent volume. Stopping preserves the database volume. Do not merge this configuration with the development Compose file.

The Makefile clears overlapping development environment variables so `.env.production` supplies the managed installation settings. Changing a password in the environment does not rotate an existing PostgreSQL role; update the initialized role separately when rotating credentials.

## Using an existing internal HTTPS ingress

If internal infrastructure already serves `https://apimock.internal`, set `PUBLIC_BASE_URL=https://apimock.internal`. Forward paths unchanged to the Go server and preserve the public Host header. Keep Go bound to loopback when the proxy runs on the same host, or otherwise restrict its published interface to the ingress network. `APP_ENV=production` can be used when the configured origin is HTTPS.

Shared URLs will then use `https://apimock.internal/public/demo-store`. No Caddy service, ACME email, PUBLIC_DOMAIN variable, or application-managed certificate is needed. Forwarded headers are not trusted to construct shared URLs.

## Management and mock access

Management pages and APIs require session login. Open `/login` and sign in using the bootstrap admin account. Only admins have access to user management; regular users only see workspaces where they are members. Owners manage membership using Workspace settings → Manage members; Editors manage endpoints and Viewers can browse/test. Administrators retain recovery access to all workspaces. Migration `003_workspace_members.sql` assigns legacy workspaces to the earliest active administrator; grant existing regular users membership after upgrading. Public mock execution remains unauthenticated. Assets and the login page are public. Sign out revokes the current session. Username, role, password, and active-status changes revoke the affected user sessions. Bootstrap environment credentials do not overwrite existing accounts on restart.

`/public/...` mock endpoints do not require administrator credentials. Do not give those credentials to mock consumers. Configure request conditions for scenarios that should return 401/403 for mock tokens. CORS governs browser access; it is not caller authentication or a restriction on server-to-server clients without an Origin header.

The `/public` prefix identifies the callable mock API, not public internet exposure. Reachability depends on your internal network and host binding.

## Calling a saved endpoint

Create workspace `demo-store`, save `GET /products/:id`, then call:

```sh
curl http://apimock.internal:8080/public/demo-store/products/123
```

The UI has separate **Copy base URL** and **Copy endpoint** controls. Replace placeholders such as `:id` before calling the endpoint.

```js
const baseURL = 'http://apimock.internal:8080/public/demo-store';
const response = await fetch(`${baseURL}/products/123`, {
  credentials: 'omit',
  headers: { 'X-Api-Key': 'demo-token' }
});
const product = await response.json();
```

Add the calling frontend origin to `MOCK_CORS_ORIGINS` when it differs from the application origin. Browser preflights receive 204 for allowed methods and headers before flow execution. Ordinary OPTIONS calls without preflight headers still use configured OPTIONS endpoints. Allowed origins receive CORS headers on mock errors too. Credentialed cross-origin requests are disabled. Access-Control-* response headers are reserved for server configuration.

## Verification

```sh
curl -i http://apimock.internal:8080/healthz
curl -i http://apimock.internal:8080/api/workspaces
curl -i http://apimock.internal:8080/login
# Sign in through the browser, or POST JSON credentials to /api/auth/login,
# retain its cookie, and send the returned csrf_token as X-CSRF-Token for writes.
curl -i http://apimock.internal:8080/public/demo-store/products/123
```

Expected results are health 200, unauthenticated management 401, authenticated management 200 after signing in, and the configured mock response. Test hostname resolution and reachability from each intended client; those depend on your infrastructure.

**Breaking route change:** `/mock/{workspace}/...` has been replaced by `/public/{workspace}/...`; there is no compatibility alias. Update consumer base URLs. Endpoint paths and stored flow data do not need migration.

## Podman Compose commands

The same Compose files can be run with Podman Compose instead of Docker Compose.

For local development on the ports used during testing:

```sh
APP_PORT=18080 DB_PORT=55432 podman compose up --build -d
podman compose logs -f app
podman compose down
```

Open `http://localhost:18080/app`. Existing workspace mocks use `/public/{workspace-slug}` on that origin.

For a managed internal installation, first copy and complete `.env.production.example` as `.env.production`, then run:

```sh
podman compose --project-name mocklab-production --env-file .env.production -f compose.production.yaml up --build -d
podman compose --project-name mocklab-production --env-file .env.production -f compose.production.yaml logs -f app
podman compose --project-name mocklab-production --env-file .env.production -f compose.production.yaml down
```

Use the configured PUBLIC_BASE_URL to access the internal installation. Keep persistent volumes when stopping unless you intend to remove stored workspaces.

The identity migration adds account and session tables without changing workspace data. Fresh development installations may create their first administrator through the setup form. Internal/production installations must provide bootstrap credentials; browser setup is disabled there. Back up account tables along with workspace data.
