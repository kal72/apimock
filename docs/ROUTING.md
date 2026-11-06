# HTTP routing and security boundaries

The HTTP adapter uses `github.com/go-chi/chi/v5` pinned to v5.5.0. Chi stays outside domain/use-case packages and serves ordinary net/http handlers, including templ rendering. Go remains the only application server.

## Route policies

| Group | Policy |
| --- | --- |
| `/healthz` | Readiness endpoint, no session; preserves the existing health Host exception. |
| `/login`, auth status/login/setup, `/assets/*` | Management origin/basic-fallback policy; no session required. Setup availability and login throttling remain enforced. |
| `/app`, `/docs/*`, configuration, Playground and own-account endpoints | Management origin policy, live session and CSRF on unsafe methods. |
| `/api/users...` | Protected group plus explicit administrator middleware. |
| Workspace endpoints | Protected group plus explicit Viewer / Editor / Owner requirements on each relevant route. |
| `/public/{slug}/*` | Independent mock CORS; no management session or CSRF. |
| Unknown/unsupported management routes | Authentication/origin checks still apply before 404/405; 405 includes the allowed methods. |

The application-level administrator is also an effective workspace owner for recovery. Workspace middleware denies missing workspace parameters and unknown policy roles instead of skipping authorization. Viewer access includes HEAD reads; writes remain Editor/Owner restricted. Member APIs and workspace rename/delete remain Owner restricted.

## Routing compatibility

- GET management/page/asset routes explicitly register HEAD; public mock methods retain independent endpoint matching.
- Wildcard mock paths use Chi's `*` capture, adapted to the existing request-path contract.
- Escaped path segments are decoded exactly once. An encoded slash stays inside its routing segment until parameter extraction; it cannot introduce a management route separator.
- Literal dot segments and repeated slashes use explicit canonical redirects with query preservation. Tests compare captures and redirects against the pinned Go ServeMux. Go 1.27 uses 307 for these redirects and trailing-slash redirects.
- Encoded literal static segments are normalized for matching without decoding encoded slashes into separators.
- A noncanonical request may receive a normalization redirect before the destination's session check. Following it still requires the destination's complete policy chain; normalization is not authorization.
- Public mock CORS is evaluated before canonical redirects so existing preflights keep their 204 behavior, even for noncanonical paths. A request-context marker prevents duplicate application inside the public group. Unknown HTTP methods on mock URLs return public errors rather than entering management login.
- Root mock URLs without a trailing slash redirect to their slash form. Old `/mock/...` aliases are not introduced.

No blanket CleanPath, RealIP, GetHead, profiler, or generic CORS middleware is installed. Login limits continue using the connection RemoteAddr, not untrusted forwarded headers. Existing server timeouts, request-body limits, cancellation, session cookies, CSRF checks and response validation remain in place. TLS termination and trusted-proxy IP handling remain deployment decisions.

Canonicalization is tested for ordinary REST methods. CONNECT does not undergo literal dot/slash cleanup, consistent with ServeMux; the mock configuration still supports its existing REST method allowlist.

## Verification

The Chi migration was checked with 67 JavaScript tests, Go race tests and vet, plus PostgreSQL integration covering login/CSRF, role and workspace isolation, session revocation and last-owner/admin safeguards. Router regression cases cover escaped/double-escaped parameters, encoded static segments, normalization redirects, root wildcard paths, HEAD/OPTIONS, Allow headers, explicit mock OPTIONS, CORS and authentication after redirects.

The Go 1.27.3 standalone build and Compose container build passed. Runtime checks confirmed healthy startup, login rendering, protected-page redirects and unauthenticated API rejection. The running container binary metadata identifies Go 1.27.3 and Chi v5.5.0. No UI interaction changes were introduced by this refactor; browser interaction QA was not repeated.

## Security scan baseline — 2026-11-06

`govulncheck` v1.9.0 reported nine symbol-reachable standard-library advisories with the existing Go 1.27.1 toolchain. It reported no reachable Chi finding. A symbol trace is not proof that every advisory is exploitable in the deployed network/protocol configuration; this was a dependency scan, not a penetration test.

| Advisory | Summary | Fixed Go patch on the 1.27 branch |
| --- | --- | --- |
| [GO-2026-6603](https://pkg.go.dev/vuln/GO-2026-6603) | HTTP/2 server memory exhaustion due to Trailer headers in net/http | 1.27.2 |
| [GO-2026-6607](https://pkg.go.dev/vuln/GO-2026-6607) | Reject malformed ECH outer extension references in crypto/tls | 1.27.2 |
| [GO-2026-6608](https://pkg.go.dev/vuln/GO-2026-6608) | Memory limit bypass when parsing MIME headers in net/textproto, mime/multipart | 1.27.2 |
| [GO-2026-6609](https://pkg.go.dev/vuln/GO-2026-6609) | Lack of limit on size of parsed Range headers in net/http | 1.27.2 |
| [GO-2026-6611](https://pkg.go.dev/vuln/GO-2026-6611) | Excessive CPU consumption from repeated initial window changes in net/http | 1.27.2 |
| [GO-2026-6612](https://pkg.go.dev/vuln/GO-2026-6612) | Double flow control refund on HTTP/2 server streams in net/http | 1.27.2 |
| [GO-2026-6613](https://pkg.go.dev/vuln/GO-2026-6613) | HTTP/1 server connection desynchronization after 2xx CONNECT response in net/http | 1.27.2 |
| [GO-2026-6617](https://pkg.go.dev/vuln/GO-2026-6617) | HTTP/2 server crash due to HPACK encoder race in net/http | 1.27.2 |
| [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809) | Distrust compromised intermediate certificates in crypto/x509 | 1.27.3 |

The project toolchain and container build image now use Go 1.27.3. A subsequent govulncheck v1.9.0 scan on 2026-11-06 reported `No vulnerabilities found.` The table above preserves the pre-upgrade baseline; the patch upgrade resolves those findings. No database migration or API configuration change is required.

Recheck after any toolchain/dependency update:

```sh
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

See [development](DEVELOPMENT.md), [account/access details](ACCESS_CONTROL.md), and [deployment](DEPLOYMENT.md) for the existing application controls.
