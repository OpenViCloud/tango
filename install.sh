#!/bin/bash

set -e

APP_NAME="tango"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="${TANGO_DIR:-/opt/tango}"
LETSENCRYPT_DIR="$BASE_DIR/letsencrypt"
ACME_FILE="$LETSENCRYPT_DIR/acme.json"
TRAEFIK_DIR="$BASE_DIR/traefik"
TRAEFIK_CONFIG_DIR="$TRAEFIK_DIR/config"
TRAEFIK_STATIC_CONFIG="$TRAEFIK_DIR/traefik.yml"
RESOURCE_MOUNT_ROOT="$BASE_DIR/data/resource-volumes"
RESOURCE_MOUNT_ROOT_APP="/platform/resource-volumes"
ENV_FILE="$BASE_DIR/.env"
CLI_CONFIG_DIR="$HOME/.config/tango"
CLI_CONFIG_FILE="$CLI_CONFIG_DIR/daemon.json"
CLI_BINARY_TARGET="/usr/local/bin/tango"
COMPOSE_FILE="$BASE_DIR/docker-compose.yml"
PROJECT_NAME="tango"
HEALTH_URL="http://localhost:8080/api/status"
CLI_RELEASE_TAG="${CLI_RELEASE_TAG:-cli-latest}"
CLI_DOWNLOAD_BASE_URL="${CLI_DOWNLOAD_BASE_URL:-https://github.com/time-groups/tango-cloud/releases/download/$CLI_RELEASE_TAG}"

EMAIL=""
DOMAIN=""
TLS_ENABLED="false"

# ── Parse args ────────────────────────────────────────────────────────────────

usage() {
  echo "Usage: ./install-dev.sh [--email your@email.com] [--domain app.example.com] [--https]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --email)
      if [ -n "$2" ] && [[ ! "$2" =~ ^-- ]]; then
        EMAIL="$2"
        shift 2
      else
        echo "Error: --email requires a value"
        usage; exit 1
      fi
      ;;
    --domain)
      if [ -n "$2" ] && [[ ! "$2" =~ ^-- ]]; then
        DOMAIN="$2"
        shift 2
      else
        echo "Error: --domain requires a value"
        usage; exit 1
      fi
      ;;
    --https)
      TLS_ENABLED="true"
      shift
      ;;
    *)
      echo "Unknown option: $1"
      usage; exit 1
      ;;
  esac
done

generate_hex() {
  local bytes="$1"
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$bytes"
    return
  fi
  od -An -N"$bytes" -tx1 /dev/urandom | tr -d ' \n'
}

generate_alnum_32() {
  tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32
}

# ── Docker install ────────────────────────────────────────────────────────────

install_docker() {
  echo "=== DOCKER NOT FOUND. INSTALLING DOCKER ==="

  sudo apt-get update
  sudo apt-get install -y ca-certificates curl gnupg

  sudo install -m 0755 -d /etc/apt/keyrings
  if [ -f /etc/apt/keyrings/docker.gpg ]; then
    sudo rm -f /etc/apt/keyrings/docker.gpg
  fi

  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
    sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  sudo chmod a+r /etc/apt/keyrings/docker.gpg

  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

  sudo apt-get update
  sudo apt-get install -y \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin

  sudo systemctl enable docker
  sudo systemctl start docker
  sudo usermod -aG docker "$USER" || true
  sudo docker run --rm hello-world

  echo "Docker installed successfully!"
  echo "Docker version:"; sudo docker version
  echo "Docker Compose version:"; docker compose version || sudo docker compose version
  echo ""
  echo "NOTE: Run 'newgrp docker' or re-login to use Docker without sudo."
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      echo "amd64"
      ;;
    aarch64|arm64)
      echo "arm64"
      ;;
    *)
      echo "unsupported"
      ;;
  esac
}

install_cli() {
  local arch cli_source cli_download_url
  arch="$(detect_arch)"
  if [ "$arch" = "unsupported" ]; then
    echo "Unsupported architecture: $(uname -m)"
    exit 1
  fi

  echo "=== INSTALL CLI ==="

  if [ -x "$CLI_BINARY_TARGET" ]; then
    echo "Using existing CLI binary: $CLI_BINARY_TARGET"
    return
  fi

  cli_source="$BASE_DIR/bin/tango-linux-$arch"
  cli_download_url="$CLI_DOWNLOAD_BASE_URL/tango-linux-$arch"

  mkdir -p "$BASE_DIR/bin"

  if [ -f "$cli_source" ]; then
    echo "Installing prebuilt CLI binary: $cli_source"
    sudo install -m 0755 "$cli_source" "$CLI_BINARY_TARGET"
    return
  fi

  echo "Downloading prebuilt CLI binary for linux/$arch"
  echo "CLI download URL: $cli_download_url"
  if curl -fsSL "$cli_download_url" -o "$cli_source"; then
    chmod +x "$cli_source"
    sudo install -m 0755 "$cli_source" "$CLI_BINARY_TARGET"
    return
  fi

  rm -f "$cli_source"
  echo "Prebuilt CLI download failed from: $cli_download_url"
  echo "If the release asset does not exist yet, run GitHub Actions -> Build -> Run workflow -> enable 'Build CLI'."
  echo "Expected release tag: $CLI_RELEASE_TAG"

  if command -v go >/dev/null 2>&1; then
    echo "Prebuilt binary not found. Building CLI from source for linux/$arch"
    GOOS=linux GOARCH="$arch" go build -o /tmp/tango-install ./cmd/cli
    sudo install -m 0755 /tmp/tango-install "$CLI_BINARY_TARGET"
    rm -f /tmp/tango-install
    return
  fi

  echo "Error: CLI binary download failed and Go is not installed."
  echo "Publish the CLI release asset first or install Go on the target machine."
  exit 1
}

write_cli_config() {
  echo "=== WRITE CLI CONFIG ==="
  mkdir -p "$CLI_CONFIG_DIR"
  cat > "$CLI_CONFIG_FILE" <<EOF
{
  "driver": "compose",
  "check_interval": "30s",
  "max_retries": 3,
  "retry_backoff": "10s",
  "retry_cooldown": "5m",
  "compose_file": "$COMPOSE_FILE",
  "project_name": "$PROJECT_NAME",
  "health_url": "$HEALTH_URL",
  "services": ["app", "traefik", "db", "buildkitd", "backup-runner"]
}
EOF
}

# ── Main ──────────────────────────────────────────────────────────────────────

echo "=== SETUP: $APP_NAME ==="
echo "Install directory: $BASE_DIR"

echo "=== SETUP BASE DIR ==="
mkdir -p "$BASE_DIR"

# Copy docker-compose.yml from repo if running from a cloned git repo,
# otherwise download from GitHub.
if [ -f "$SCRIPT_DIR/docker-compose.yml" ] && [ "$SCRIPT_DIR" != "$BASE_DIR" ] && [ -d "$SCRIPT_DIR/.git" ]; then
  echo "Copying docker-compose.yml from repo: $SCRIPT_DIR"
  cp "$SCRIPT_DIR/docker-compose.yml" "$COMPOSE_FILE"
elif [ ! -f "$COMPOSE_FILE" ]; then
  echo "Downloading docker-compose.yml from GitHub..."
  curl -fsSL "https://raw.githubusercontent.com/time-groups/tango-cloud/main/docker-compose.yml" \
    -o "$COMPOSE_FILE"
else
  echo "Using existing $COMPOSE_FILE"
fi

echo "=== CHECK DOCKER ==="
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  echo "Docker already installed"
else
  install_docker
fi

echo "=== LETSENCRYPT SETUP ==="
mkdir -p "$LETSENCRYPT_DIR"
if [ ! -f "$ACME_FILE" ]; then
  touch "$ACME_FILE"
fi
chmod 600 "$ACME_FILE"

echo "=== TRAEFIK CONFIG DIR ==="
mkdir -p "$TRAEFIK_CONFIG_DIR"

write_traefik_static_config() {
  local email="$1"
  cat > "$TRAEFIK_STATIC_CONFIG" <<TRAEFIK_EOF
api:
  dashboard: true
  insecure: false
providers:
  docker:
    exposedByDefault: false
  file:
    directory: /traefik/config
    watch: true
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
ping: {}
TRAEFIK_EOF

  if [ -n "$email" ]; then
    cat >> "$TRAEFIK_STATIC_CONFIG" <<ACME_EOF
certificatesResolvers:
  letsencrypt:
    acme:
      email: $email
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
ACME_EOF
  fi
}

echo "=== RESOURCE MOUNT ROOT ==="
mkdir -p "$RESOURCE_MOUNT_ROOT"

echo "=== DOCKER NETWORK ==="
if ! docker network inspect tango_net >/dev/null 2>&1; then
  docker network create tango_net
fi

echo "=== WRITE .env ==="

# Load existing .env values if file exists
existing_email=""
existing_domain=""
existing_tls="false"
existing_resource_mount_root=""
existing_resource_mount_root_app=""
existing_jwt_secret=""
existing_data_encryption_key=""
existing_postgres_password=""
existing_database_url=""
if [ -f "$ENV_FILE" ]; then
  existing_email=$(grep "^ACME_EMAIL=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
  existing_domain=$(grep "^APP_DOMAIN=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
  existing_tls=$(grep "^APP_TLS_ENABLED=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "false")
  existing_resource_mount_root=$(grep "^RESOURCE_MOUNT_ROOT=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
  existing_resource_mount_root_app=$(grep "^RESOURCE_MOUNT_ROOT_APP=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
  existing_jwt_secret=$(grep "^JWT_SECRET=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
  existing_data_encryption_key=$(grep "^DATA_ENCRYPTION_KEY=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
  existing_postgres_password=$(grep "^POSTGRES_PASSWORD=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
  existing_database_url=$(grep "^DATABASE_URL=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
fi

# Resolve final values (args override existing)
final_email="${EMAIL:-$existing_email}"
final_domain="${DOMAIN:-$existing_domain}"
final_tls="${TLS_ENABLED:-$existing_tls}"
final_resource_mount_root="${existing_resource_mount_root:-$RESOURCE_MOUNT_ROOT}"
final_resource_mount_root_app="${existing_resource_mount_root_app:-$RESOURCE_MOUNT_ROOT_APP}"
final_jwt_secret="${existing_jwt_secret:-$(generate_hex 32)}"
final_data_encryption_key="${existing_data_encryption_key:-$(generate_alnum_32)}"
final_postgres_password="${existing_postgres_password:-$(generate_hex 24)}"
default_database_url="postgres://postgres:${final_postgres_password}@db:5432/tango?sslmode=disable"
final_database_url="${existing_database_url:-$default_database_url}"

# --https requires --email (for Let's Encrypt)
if [ "$final_tls" = "true" ] && [ -z "$final_email" ]; then
  echo "Warning: --https requires --email for Let's Encrypt. HTTPS disabled."
  final_tls="false"
fi

cat > "$ENV_FILE" <<EOF
APP_DOMAIN=$final_domain
APP_TLS_ENABLED=$final_tls
ACME_EMAIL=$final_email
RESOURCE_MOUNT_ROOT=$final_resource_mount_root
RESOURCE_MOUNT_ROOT_APP=$final_resource_mount_root_app
POSTGRES_PASSWORD=$final_postgres_password
DATABASE_URL=$final_database_url
JWT_SECRET=$final_jwt_secret
DATA_ENCRYPTION_KEY=$final_data_encryption_key
EOF

sudo chown root:root "$ENV_FILE"
sudo chmod 600 "$ENV_FILE"

echo "Let's Encrypt : ${final_email:-DISABLED}"
echo "App domain    : ${final_domain:-not set (configure in Settings)}"
echo "App HTTPS     : $final_tls"
echo "Mount root    : $final_resource_mount_root"
echo "Database URL  : generated/preserved in $ENV_FILE"
echo "DB password   : generated/preserved in $ENV_FILE"
echo "JWT secret    : generated/preserved in $ENV_FILE"
echo "Data key      : generated/preserved in $ENV_FILE"

echo "=== TRAEFIK STATIC CONFIG ==="
write_traefik_static_config "$final_email"
echo "Traefik static config : $TRAEFIK_STATIC_CONFIG (acme=${final_email:-disabled})"

install_cli
write_cli_config

echo "=== PULL LATEST IMAGES ==="
docker compose -f "$COMPOSE_FILE" pull

echo "=== START SERVICES ==="
docker compose -f "$COMPOSE_FILE" up -d
echo "Services started."

echo "=== INSTALL DAEMON SERVICE ==="
sudo "$CLI_BINARY_TARGET" daemon install

echo "=== CHECK ORCHESTRATION STATUS ==="
"$CLI_BINARY_TARGET" status || true
"$CLI_BINARY_TARGET" daemon status || true
"$CLI_BINARY_TARGET" service list || true

echo ""
echo "=================================="
echo " $APP_NAME orchestration is installed!"
echo " CLI binary     : $CLI_BINARY_TARGET"
echo " CLI config     : $CLI_CONFIG_FILE"
echo " ACME file      : $ACME_FILE"
echo " Traefik config : $TRAEFIK_CONFIG_DIR"
echo " ENV file       : $ENV_FILE"
echo " Resource root  : $final_resource_mount_root"
if [ -n "$final_domain" ]; then
  proto="http"
  [ "$final_tls" = "true" ] && proto="https"
  echo " App URL        : $proto://$final_domain"
else
  echo " App URL        : http://<server-ip>:8080"
  echo "                  (set domain in Settings UI)"
fi
echo "=================================="
