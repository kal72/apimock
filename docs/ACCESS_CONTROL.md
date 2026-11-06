# Accounts and workspace access

Open `/app`; unauthenticated users are redirected to `/login`. On a fresh development installation without bootstrap credentials, this page asks you to create the first administrator. There are no default credentials and no public self-registration after setup. In internal/production deployments, set `ADMIN_USERNAME` and `ADMIN_PASSWORD` before first startup; they seed the first admin only when the user table is empty. Changing these environment values later does not reset database accounts. Use a 3–64 character lowercase username (letters, numbers, dots, underscores, hyphens) and a password of at least 8 characters (maximum 1024 bytes).

Administrators have a **User management** action in the toolbar account menu for creating users, editing names/usernames/roles, resetting passwords, deactivating accounts, and deleting accounts. Regular users cannot access these APIs, even by constructing requests manually. Workspace membership controls management access: Viewer can browse and test, Editor can additionally create/update/delete endpoints, and Owner can additionally edit/delete the workspace and manage members. Administrators retain recovery access to all workspaces. An admin cannot delete, demote, or deactivate their own account, and transactions protect the last active administrator. Open the account menu and choose **Change password** to change your own password. Password, username, role, or status changes revoke affected sessions. Deleting a user revokes sessions and removes memberships. Deletion or deactivation is blocked when the account is the last active owner of a workspace; assign another active owner first.

Sessions are opaque random tokens with a 12-hour absolute lifetime. PostgreSQL stores only token hashes; browser cookies use HttpOnly, SameSite=Strict, and Secure on HTTPS. Unsafe management requests also require a session-bound `X-CSRF-Token`; the UI fetches it from `/api/auth/me` and keeps it in memory. Passwords use salted PBKDF2-HMAC-SHA256 with 600,000 iterations through Go's standard library. Login/setup/password attempts are bounded per IP and username (20 per 15 minutes per server process); multi-instance distributed throttling is not implemented. HTTP on an internal network does not encrypt passwords or sessions; use the existing HTTPS infrastructure for transport confidentiality.

**Migration:** `002_identity.sql` automatically adds `app_users` and `app_sessions`; existing workspaces and endpoints are preserved. The running server replaces Basic authentication with session login, including in development. API clients must log in with JSON at `POST /api/auth/login`, retain the session cookie, and send the returned `csrf_token` as `X-CSRF-Token` for writes. `POST /api/auth/logout` revokes the session. Migration `003_workspace_members.sql` assigns existing workspaces to the earliest active administrator (ordered by creation time and ID). If the user table is empty, first-admin setup/bootstrap claims existing unowned workspaces. Existing public mock URLs and endpoint configurations are preserved. **Breaking change:** regular accounts lose the previous global management access until an owner/admin adds them to each workspace.

Password and session design references: [OWASP Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), and the [Go PBKDF2 standard-library documentation](https://go.dev/pkg/crypto/pbkdf2/).

## Workspace membership

Create a workspace to become its Owner. Open **Workspace settings** (the gear beside the selector), then **Manage members** to add an existing active account by username, change its role, or remove access. Owners may add another owner before leaving or downgrading themselves. At least one active owner must remain. Admin recovery access does not count as a membership or a replacement owner.

| Workspace role | Browse / Playground | Manage endpoints | Rename / delete workspace | Manage members |
| --- | --- | --- | --- | --- |
| Viewer | Yes | No | No | No |
| Editor | Yes | Yes | No | No |
| Owner | Yes | Yes | Yes | Yes |

Workspace lists and endpoint APIs enforce membership on the server. The UI hides editing controls for viewers. Members use existing accounts created by an administrator; email invitations are not included. Public `/public/{slug}` URLs intentionally remain callable without authentication; membership protects configuration, not execution of the mock API.

Membership API (session cookie and CSRF header required for writes): `GET /api/workspaces/{id}/members`, `PUT /api/workspaces/{id}/members/{username}` with `{"role":"viewer|editor|owner"}`, and `DELETE /api/workspaces/{id}/members/{username}`. Listing and modifying members require Owner access or administrator recovery access.


See [deployment instructions](DEPLOYMENT.md) for bootstrap configuration, internal URLs, and CORS.
