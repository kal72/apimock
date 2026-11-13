#!/usr/bin/env bash
# ==============================================================================
# Mocklab - Podman Pull & Run Deployment Script for Manual VPS (No Compose)
# ==============================================================================
# This script is executed on the VPS. It:
# 1. Authenticates to GitHub Container Registry (ghcr.io)
# 2. Pulls the pre-built container image from GHCR
# 3. Stops and deletes the previous container
# 4. Starts the new container with Podman
# 5. Verifies application health at /healthz
# 6. Cleans up old dangling images
# ==============================================================================
set -euo pipefail

TARGET_ENV="${1:-dev}"
IMAGE_NAME="${2:-}"
APP_PORT="${3:-8080}"
APP_BIND="${4:-127.0.0.1}"
DEPLOY_PATH="${5:-/opt/mocklab/${TARGET_ENV}}"

CONTAINER_NAME="mocklab-${TARGET_ENV}"
ENV_FILE="${DEPLOY_PATH}/.env"
if [ "$TARGET_ENV" = "prod" ] && [ -f "${DEPLOY_PATH}/.env.production" ]; then
  ENV_FILE="${DEPLOY_PATH}/.env.production"
fi

echo "======================================================================"
echo " Mocklab Podman Deployment (GHCR Image)"
echo " Environment : ${TARGET_ENV}"
echo " Container   : ${CONTAINER_NAME}"
echo " Image       : ${IMAGE_NAME}"
echo " Port Binding: ${APP_BIND}:${APP_PORT} -> 8080"
echo " Deploy Path : ${DEPLOY_PATH}"
echo " Env File    : ${ENV_FILE}"
echo "======================================================================"

if ! command -v podman >/dev/null 2>&1; then
  echo "Error: 'podman' command is not installed on this VPS." >&2
  exit 1
fi

if [ -z "$IMAGE_NAME" ]; then
  echo "Error: Container image name must be provided as the 2nd argument." >&2
  exit 1
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: Environment file '${ENV_FILE}' not found." >&2
  exit 1
fi

# Step 1: Optional authentication with GitHub Container Registry
if [ -n "${GHCR_TOKEN:-}" ]; then
  echo "--> Authenticating with GitHub Container Registry (ghcr.io)..."
  echo "${GHCR_TOKEN}" | podman login ghcr.io -u "${GHCR_USER:-github-actions}" --password-stdin
fi

# Step 2: Pull the fresh image from GHCR
echo "--> Pulling latest image '${IMAGE_NAME}' from GHCR..."
podman pull "$IMAGE_NAME"

# Step 3: Stop and remove existing container
if podman container exists "$CONTAINER_NAME" 2>/dev/null; then
  echo "--> Stopping existing container '${CONTAINER_NAME}'..."
  podman stop "$CONTAINER_NAME" || true
  echo "--> Deleting existing container '${CONTAINER_NAME}'..."
  podman rm -f "$CONTAINER_NAME" || true
fi

# Step 4: Run the new container with Podman
echo "--> Running container '${CONTAINER_NAME}'..."
EXTRA_ARGS="${PODMAN_EXTRA_ARGS:-}"

# shellcheck disable=SC2086
podman run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p "${APP_BIND}:${APP_PORT}:8080" \
  --env-file "$ENV_FILE" \
  $EXTRA_ARGS \
  "$IMAGE_NAME"

# Step 5: Verify application health
echo "--> Verifying health check on ${APP_BIND}:${APP_PORT}/healthz..."
CHECK_HOST="127.0.0.1"
if [ "$APP_BIND" != "0.0.0.0" ] && [ "$APP_BIND" != "127.0.0.1" ]; then
  CHECK_HOST="$APP_BIND"
fi

HEALTH_SUCCESS=false
for i in $(seq 1 30); do
  if curl -s -f "http://${CHECK_HOST}:${APP_PORT}/healthz" >/dev/null 2>&1; then
    HEALTH_SUCCESS=true
    echo "--> Health check passed! Container responded with HTTP 200 (ok)."
    break
  fi
  echo "    Waiting for application startup (attempt ${i}/30)..."
  sleep 2
done

if [ "$HEALTH_SUCCESS" != "true" ]; then
  echo "Error: Health check timed out after 60 seconds." >&2
  echo "--> Container logs for diagnosis:" >&2
  podman logs --tail 50 "$CONTAINER_NAME" >&2 || true
  exit 1
fi

# Step 6: Prune dangling/unused images to save disk space
echo "--> Cleaning up dangling images..."
podman image prune -f || true

echo "--> Active Podman container status:"
podman ps --filter "name=${CONTAINER_NAME}"

echo "======================================================================"
echo " Successfully deployed ${CONTAINER_NAME}!"
echo " Image: ${IMAGE_NAME}"
echo "======================================================================"
