# CI/CD with Podman on Manual VPS (Dev & Prod)

Mocklab includes automated GitHub Actions pipelines for Continuous Integration (CI) and Continuous Deployment (CD) using **Podman directly** (without Docker/Podman Compose) on self-managed Linux VPS hosts for both `dev` and `prod` application environments.

## Architecture overview

```mermaid
flowchart TD
    subgraph GitHub ["GitHub Infrastructure"]
        DevBranch["Branch: develop"]
        MainBranch["Branch: main"]
        ManualDispatch["Manual Workflow Dispatch\n(Select branch + dev/prod)"]
        CIWorkflow["CI Workflow: .github/workflows/ci.yml\n- npm test\n- go test -race\n- go vet\n- podman build check"]
        BuildJob["CD: Build & Push Job\n- Podman builds multi-stage image\n- Pushes to GHCR: ghcr.io/owner/repo:tag"]
        GHCR[("GitHub Container Registry\n(ghcr.io)")]
    end

    subgraph VPS ["Manual VPS Host (Podman)"]
        subgraph DeployProcess ["SSH Deployment Flow"]
            UploadEnv["1. Upload environment configuration (.env)"]
            AuthGHCR["2. Authenticate to GHCR"]
            PullImage["3. podman pull ghcr.io/owner/repo:tag"]
            DeleteOld["4. podman stop && podman rm (delete old container)"]
            RunNew["5. podman run (run fresh container with .env)"]
            HealthCheck["6. Health check: /healthz HTTP 200"]
            Prune["7. podman image prune -f"]
        end

        subgraph Containers ["Running Podman Containers"]
            DevApp["mocklab-dev container (:8081)"]
            ProdApp["mocklab-prod container (:8080)"]
        end
    end

    DevBranch -->|Push| BuildJob
    MainBranch -->|Push| BuildJob
    ManualDispatch --> BuildJob
    DevBranch -->|PR / Push| CIWorkflow
    MainBranch -->|PR / Push| CIWorkflow

    BuildJob -->|Push image| GHCR
    BuildJob -->|Trigger deploy via SSH| UploadEnv
    UploadEnv --> AuthGHCR --> PullImage --> DeleteOld --> RunNew --> HealthCheck --> Prune
    RunNew -->|dev environment| DevApp
    RunNew -->|prod environment| ProdApp
    GHCR -.->|Pull pre-built image| PullImage
```

## GitHub Actions workflows

1. **`.github/workflows/ci.yml`**:
   - Triggers on pull requests and pushes to `main` and `develop`.
   - Runs frontend tests (`npm test`), template generation (`go tool templ generate`), asset build (`npm run build`), Go tests (`go test -race ./...`), and `go vet ./...`.
   - Validates standalone container builds with `podman build`.

2. **`.github/workflows/cd.yml`**:
   - Triggers automatically on push to `develop` (targets `dev`) and `main` (targets `prod`).
   - Supports manual execution anytime via `workflow_dispatch` with branch selection and environment selector (`dev` or `prod`).
   - **Builds container image once on GitHub Actions runners** and pushes to **GitHub Container Registry (`ghcr.io`)**.
   - **No source code sync to VPS**: The VPS host does not need repository files, `rsync` of code, Node, or Go compilers.
   - Deploys to VPS via SSH: uploads the `.env` file, logs into `ghcr.io`, pulls the pre-built image, deletes the previous container (`podman stop && podman rm`), starts the new container with `podman run`, and verifies `/healthz`.
   - **Zero Compose dependency**: No `compose.yaml` or `podman-compose` is required on the VPS.

---

## Configuring GitHub Actions

In your repository on GitHub: navigate to **Settings → Environments**, and create two environments:
1. `dev`
2. `prod` (optionally enable *Required reviewers* for manual deployment approval).

### 1. Environment Secrets

Configure these in **Settings → Environments → [dev / prod] → Environment secrets**:

| Secret Name | Required | Description | Example |
| --- | --- | --- | --- |
| `SSH_HOST` | Yes | VPS IP address or domain name | `203.0.113.10` or `vps.example.com` |
| `SSH_USER` | Yes | SSH user on VPS with Podman access | `deploy` or `root` |
| `SSH_PASSWORD` | Optional* | Password for SSH login on VPS (via `sshpass`) | `MyVpsPassword123!` |
| `SSH_PRIVATE_KEY` | Optional* | OpenSSH private key for key-based authentication | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `DATABASE_URL` | Yes | Full PostgreSQL connection URL | `postgres://user:pass@127.0.0.1:5432/mocklab?sslmode=disable` |
| `ADMIN_PASSWORD` | Yes | Initial bootstrap admin password | `SuperAdminPassword123!` |
| `SSH_PORT` | No | Custom SSH port (defaults to `22`) | `22` or `2222` |

*\*Note: Provide either `SSH_PASSWORD` (for standard password login) or `SSH_PRIVATE_KEY` (for SSH key-based login).*

> [!TIP]
> If your database is running on the VPS host itself and the container is in Podman rootless network:
> - You can connect via `127.0.0.1:5432` if using `--network host` in `PODMAN_EXTRA_ARGS`, or
> - Connect via the host gateway (e.g. `host.containers.internal:5432`), or
> - Connect via the host's private LAN IP.

### 2. Environment Variables

Configure these in **Settings → Environments → [dev / prod] → Environment variables**:

| Variable Name | Required | Dev Default | Prod Default | Description |
| --- | --- | --- | --- | --- |
| `APP_ENV` | No | `development` | `internal` | `development`, `internal`, or `production` |
| `PUBLIC_BASE_URL` | Yes | `http://dev-mocklab.example.com` | `https://mocklab.example.com` | Public origin of the application |
| `APP_PORT` | No | `8081` | `8080` | Host port mapped to the Go app |
| `APP_BIND` | No | `127.0.0.1` | `127.0.0.1` | Interface IP binding (`127.0.0.1` behind reverse proxy) |
| `DEPLOY_PATH` | No | `/opt/mocklab/dev` | `/opt/mocklab/prod` | Project directory on VPS |
| `ADMIN_USERNAME` | No | `admin` | `admin` | Bootstrap administrator username |
| `MOCK_CORS_ORIGINS` | No | `*` | `https://myfrontend.com` | Allowed CORS origins for mock calls |
| `MOCK_CORS_HEADERS` | No | `Content-Type,Authorization,X-Api-Key` | `Content-Type,Authorization,X-Api-Key` | Allowed CORS request headers |
| `PODMAN_EXTRA_ARGS` | No | `""` | `""` | Optional extra arguments for `podman run` |

---

## VPS Prerequisites & Setup

Run these steps once on your VPS host (Debian/Ubuntu/RHEL/Fedora). No compose tool is needed!

### 1. Install Podman

#### On Ubuntu / Debian:
```sh
sudo apt-get update
sudo apt-get install -y podman curl rsync
```

#### On Fedora / RHEL 9:
```sh
sudo dnf install -y podman curl rsync
```

Verify Podman:
```sh
podman --version
```

### 2. Configure Deployment User (Recommended)

Create a dedicated user `deploy` and enable lingering so containers continue running after SSH logout:

```sh
sudo useradd -m -s /bin/bash deploy
sudo loginctl enable-linger deploy

# Prepare directories
sudo mkdir -p /opt/mocklab/dev /opt/mocklab/prod
sudo chown -R deploy:deploy /opt/mocklab
```

### 3. SSH Authentication Options

#### Option A: Password Authentication (Recommended if VPS uses password only)
Simply add your VPS SSH user's password as the `SSH_PASSWORD` secret in GitHub Settings → Environments.
The deployment workflow will automatically install `sshpass` on the runner and connect securely using the password. No keypair generation or server configuration is required!

#### Option B: Key-based Authentication
If you prefer SSH keys:
1. Generate an SSH keypair:
   ```sh
   ssh-keygen -t ed25519 -C "github-actions-mocklab" -f ~/.ssh/mocklab_deploy
   ```
2. Save `~/.ssh/mocklab_deploy` (private key) in GitHub Secret `SSH_PRIVATE_KEY`.
3. Append `~/.ssh/mocklab_deploy.pub` (public key) to `/home/deploy/.ssh/authorized_keys` on your VPS:
   ```sh
   mkdir -p ~/.ssh && chmod 700 ~/.ssh
   cat >> ~/.ssh/authorized_keys << 'EOF'
   ssh-ed25519 AAAAC3... github-actions-mocklab
   EOF
   chmod 600 ~/.ssh/authorized_keys
   ```

---

## Reverse Proxy Configuration (Nginx / Caddy)

When binding `APP_BIND=127.0.0.1`, route traffic from your external web server or reverse proxy to the application port (`8080` for prod, `8081` for dev).

### Nginx Example (`/etc/nginx/sites-available/mocklab.conf`):

```nginx
server {
    listen 80;
    server_name mocklab.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## Image Versioning Strategy

| Environment | Image Tag Rule | Example Image Tag |
| --- | --- | --- |
| **`dev`** | Always tagged with `:dev` | `ghcr.io/username/apimock:dev` |
| **`prod`** | Tagged with custom version input (e.g. `v1.0`, `v1.2.3`), git release tag (`v*`), plus `:latest` | `ghcr.io/username/apimock:v1.0` |

All builds also generate an immutable commit-SHA tag (`ghcr.io/username/apimock:<git-sha>`).

---

## Triggering Deployments

### Automatic Deployment
- Push to `develop` branch → automatically builds and deploys `ghcr.io/owner/repo:dev` to `dev`.
- Push to `main` branch → automatically builds and deploys `ghcr.io/owner/repo:v1.0` (and `:latest`) to `prod`.
- Push to a release tag (e.g. `git tag v1.0 && git push origin v1.0`) → automatically builds and deploys `ghcr.io/owner/repo:v1.0` to `prod`.

### Manual Deployment via GitHub UI
1. Go to your repository on GitHub.
2. Click **Actions** → select **CD (Podman VPS)** in the left sidebar.
3. Click **Run workflow**.
4. Configure options:
   - **Use workflow from**: Choose the base branch from the GitHub dropdown.
   - **Target deployment environment**: Select `dev` or `prod`.
   - **Image version for prod**: If deploying to `prod`, enter your version tag (e.g. `v1.0`, `v2.0.1`). When `dev` is selected, this input is ignored and the image version is always `dev`.
   - **Branch or Git ref to deploy**: Optionally enter a specific branch name, tag, or commit. If left blank, it deploys the branch selected in the dropdown.
   - **Skip CI test checks**: Check only if you need emergency deployment without re-running unit tests.
5. Click **Run workflow** to initiate deployment.

---

## Troubleshooting & Maintenance on VPS

### View Container Logs

```sh
# Dev container
podman logs -f mocklab-dev

# Prod container
podman logs -f mocklab-prod
```

### Check Container Status

```sh
podman ps -a --filter "name=mocklab"
```

### Manual Restart via Script

```sh
cd /opt/mocklab/prod
./deploy/podman-deploy.sh prod ghcr.io/username/apimock:v1.0 8080 127.0.0.1 /opt/mocklab/prod
```
```
