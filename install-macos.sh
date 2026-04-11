#!/bin/bash

set -euo pipefail

APP_NAME="tango"

# ── Khi chạy qua "curl | sudo bash", BASH_SOURCE[0] sẽ rỗng.
# SCRIPT_DIR chỉ dùng để detect chạy từ repo clone → luôn fallback download nếu không có .git
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "")"

# ── Khi chạy qua "sudo bash", $USER = root, $SUDO_USER = user thật.
# Phải set REAL_USER/REAL_HOME trước để dùng trong BASE_DIR
REAL_USER="${SUDO_USER:-$USER}"
REAL_HOME=$(eval echo "~$REAL_USER")

# macOS: dùng ~/tango thay vì /opt/tango vì Docker Desktop chỉ share home theo mặc định.
# Override bằng env: TANGO_DIR=/opt/tango sudo bash install-macos.sh
BASE_DIR="${TANGO_DIR:-$REAL_HOME/tango}"
LETSENCRYPT_DIR="$BASE_DIR/letsencrypt"
ACME_FILE="$LETSENCRYPT_DIR/acme.json"
TRAEFIK_DIR="$BASE_DIR/traefik"
TRAEFIK_CONFIG_DIR="$TRAEFIK_DIR/config"
TRAEFIK_STATIC_CONFIG="$TRAEFIK_DIR/traefik.yml"
RESOURCE_MOUNT_ROOT="$BASE_DIR/data/resource-volumes"
RESOURCE_MOUNT_ROOT_APP="/platform/resource-volumes"
ENV_FILE="$BASE_DIR/.env"
CLI_BINARY_TARGET="/usr/local/bin/tango"
COMPOSE_FILE="$BASE_DIR/docker-compose.yml"
PROJECT_NAME="tango"
HEALTH_URL="http://localhost:8080/api/status"
CLI_RELEASE_TAG="${CLI_RELEASE_TAG:-cli-latest}"
CLI_DOWNLOAD_BASE_URL="${CLI_DOWNLOAD_BASE_URL:-https://github.com/time-groups/tango-cloud/releases/download/$CLI_RELEASE_TAG}"

EMAIL=""
DOMAIN=""
TLS_ENABLED="false"

# ── Temp dir với auto-cleanup khi script exit (dù thành công hay lỗi)
TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

# CLI config nằm trong home của user thật, không phải /root/.config
CLI_CONFIG_DIR="$REAL_HOME/.config/tango"
CLI_CONFIG_FILE="$CLI_CONFIG_DIR/daemon.json"

# ── Parse args ────────────────────────────────────────────────────────────────

usage() {
  echo "Usage: install-macos.sh [--email your@email.com] [--domain app.example.com] [--https]"
  echo ""
  echo "Quick install:"
  echo "  curl -fsSL https://raw.githubusercontent.com/time-groups/tango-cloud/main/install-macos.sh | sudo bash"
  echo ""
  echo "With options (pass args after -s --):"
  echo "  curl -fsSL https://raw.githubusercontent.com/time-groups/tango-cloud/main/install-macos.sh | sudo bash -s -- --domain app.example.com --email you@example.com --https"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --email)
      if [ -n "$2" ] && [[ ! "$2" =~ ^-- ]]; then
        EMAIL="$2"; shift 2
      else
        echo "Error: --email requires a value"; usage; exit 1
      fi
      ;;
    --domain)
      if [ -n "$2" ] && [[ ! "$2" =~ ^-- ]]; then
        DOMAIN="$2"; shift 2
      else
        echo "Error: --domain requires a value"; usage; exit 1
      fi
      ;;
    --https)
      TLS_ENABLED="true"; shift
      ;;
    -h|--help)
      usage; exit 0
      ;;
    *)
      echo "Unknown option: $1"; usage; exit 1
      ;;
  esac
done

# ── Helpers ───────────────────────────────────────────────────────────────────

generate_hex() {
  openssl rand -hex "$1"
}

generate_alnum_32() {
  # head -c đóng pipe sớm khiến tr exit != 0, cần tắt pipefail cục bộ
  local result
  result=$(set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
  echo "$result"
}

# ── Resolve CLI version từ GitHub Releases (dynamic) ─────────────────────────

resolve_cli_version() {
  if [[ -n "$CLI_RELEASE_TAG" && "$CLI_RELEASE_TAG" != "cli-latest" ]]; then
    echo "CLI version (override) : $CLI_RELEASE_TAG"
    return
  fi

  echo "→ Fetching latest CLI release tag from GitHub..."
  local fetched
  fetched=$(curl -fsSL "https://api.github.com/repos/time-groups/tango-cloud/releases?per_page=50" \
    | grep '"tag_name": "cli-' \
    | head -1 \
    | sed 's/.*"tag_name": "\([^"]*\)".*/\1/' || true)

  if [[ -n "$fetched" ]]; then
    CLI_RELEASE_TAG="$fetched"
    CLI_DOWNLOAD_BASE_URL="https://github.com/time-groups/tango-cloud/releases/download/$CLI_RELEASE_TAG"
    echo "CLI version (resolved) : $CLI_RELEASE_TAG"
  else
    echo "Warning: Could not resolve CLI version from GitHub API. Using fallback: $CLI_RELEASE_TAG"
  fi
}

# ── macOS guard ───────────────────────────────────────────────────────────────

check_macos() {
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "Error: This script is for macOS only."
    echo "For Linux, use:"
    echo "  curl -fsSL https://raw.githubusercontent.com/time-groups/tango-cloud/main/install.sh | sudo bash"
    exit 1
  fi
}

# ── Docker check ──────────────────────────────────────────────────────────────

check_docker() {
  echo "=== CHECK DOCKER ==="

  if ! command -v docker >/dev/null 2>&1; then
    echo ""
    echo "ERROR: Docker is not installed."
    echo ""
    echo "Please install Docker Desktop for Mac, then re-run this script:"
    echo "  Download : https://www.docker.com/products/docker-desktop/"
    echo "  Homebrew : brew install --cask docker"
    echo ""
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    # Trên macOS, Docker Desktop cài compose plugin vào thư mục user-space,
    # nhưng sudo/root không load được. Tự fix bằng cách symlink vào system-wide.
    local compose_src
    compose_src="$(find /Applications/Docker.app -name "docker-compose" 2>/dev/null | head -1 || true)"

    if [[ -n "$compose_src" ]]; then
      echo "→ Fixing Docker Compose plugin for root (symlink to system-wide)..."
      mkdir -p /usr/local/lib/docker/cli-plugins
      ln -sf "$compose_src" /usr/local/lib/docker/cli-plugins/docker-compose
    fi

    if ! docker compose version >/dev/null 2>&1; then
      echo ""
      echo "ERROR: 'docker compose' plugin not found."
      echo "Please update Docker Desktop to the latest version:"
      echo "  https://www.docker.com/products/docker-desktop/"
      echo ""
      exit 1
    fi
  fi

  if ! docker info >/dev/null 2>&1; then
    echo ""
    echo "ERROR: Docker daemon is not running."
    echo "Please open Docker Desktop and wait until it's ready, then re-run this script."
    echo "  open -a Docker"
    echo ""
    exit 1
  fi

  echo "Docker CLI     : $(docker --version)"
  echo "Docker Compose : $(docker compose version)"
  echo "Docker daemon  : running ✓"
}

# ── Arch detection ────────────────────────────────────────────────────────────

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)             echo "unsupported" ;;
  esac
}

# ── CLI install ───────────────────────────────────────────────────────────────

install_cli() {
  local arch cli_source cli_download_url
  arch="$(detect_arch)"
  if [ "$arch" = "unsupported" ]; then
    echo "Unsupported architecture: $(uname -m)"; exit 1
  fi

  echo "=== INSTALL CLI ==="

  if [ -x "$CLI_BINARY_TARGET" ]; then
    echo "Using existing CLI binary: $CLI_BINARY_TARGET"
    return
  fi

  cli_source="$BASE_DIR/bin/tango-darwin-$arch"
  cli_download_url="$CLI_DOWNLOAD_BASE_URL/tango-darwin-$arch"

  mkdir -p "$BASE_DIR/bin"

  if [ -f "$cli_source" ]; then
    echo "Installing prebuilt CLI binary: $cli_source"
    install -m 0755 "$cli_source" "$CLI_BINARY_TARGET"
    xattr -d com.apple.quarantine "$CLI_BINARY_TARGET" 2>/dev/null || true
    return
  fi

  local tmp_download="$TMPDIR_WORK/tango-darwin-$arch"
  echo "Downloading CLI binary for darwin/$arch..."
  echo "  URL: $cli_download_url"
  if curl -fSL --progress-bar "$cli_download_url" -o "$tmp_download"; then
    # Verify file thực sự được download (không phải trang 404 HTML)
    if [[ ! -s "$tmp_download" ]]; then
      echo "❌ Downloaded file is empty."
      rm -f "$tmp_download"
    else
      chmod +x "$tmp_download"
      install -m 0755 "$tmp_download" "$CLI_BINARY_TARGET"
      xattr -d com.apple.quarantine "$CLI_BINARY_TARGET" 2>/dev/null || true
      # Cache vào BASE_DIR/bin để lần sau không cần download lại
      cp "$tmp_download" "$cli_source" 2>/dev/null || true
      return
    fi
  fi
  echo "❌ CLI binary download failed."
  echo "   URL tried   : $cli_download_url"
  echo "   Release tag : $CLI_RELEASE_TAG"
  echo "   Available   : https://github.com/time-groups/tango-cloud/releases"

  if command -v go >/dev/null 2>&1; then
    echo "Building CLI from source for darwin/$arch..."
    GOOS=darwin GOARCH="$arch" go build -o /tmp/tango-install ./cmd/cli
    install -m 0755 /tmp/tango-install "$CLI_BINARY_TARGET"
    rm -f /tmp/tango-install
    return
  fi

  echo "Error: CLI binary download failed and Go is not installed."
  echo "  Install Go via: brew install go"
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
  # Config phải thuộc user thật, không phải root
  chown -R "$REAL_USER" "$CLI_CONFIG_DIR"
}

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

# ── Main ──────────────────────────────────────────────────────────────────────

check_macos

echo "=== SETUP: $APP_NAME (macOS) ==="
echo "Install directory : $BASE_DIR"
echo "Running as        : $(whoami) (real user: $REAL_USER)"

echo "=== SETUP BASE DIR ==="
mkdir -p "$BASE_DIR"

# Nếu chạy từ repo clone (local), copy docker-compose.yml.
# Khi curl | bash: SCRIPT_DIR không có .git → luôn download từ GitHub.
if [ -n "$SCRIPT_DIR" ] \
  && [ -f "$SCRIPT_DIR/docker-compose.yml" ] \
  && [ "$SCRIPT_DIR" != "$BASE_DIR" ] \
  && [ -d "$SCRIPT_DIR/.git" ]; then
  echo "Copying docker-compose.yml from repo: $SCRIPT_DIR"
  cp "$SCRIPT_DIR/docker-compose.yml" "$COMPOSE_FILE"
elif [ ! -f "$COMPOSE_FILE" ]; then
  echo "Downloading docker-compose.yml from GitHub..."
  curl -fsSL "https://raw.githubusercontent.com/time-groups/tango-cloud/main/docker-compose.yml" \
    -o "$COMPOSE_FILE"
else
  echo "Using existing $COMPOSE_FILE"
fi

check_docker

echo "=== LETSENCRYPT SETUP ==="
mkdir -p "$LETSENCRYPT_DIR"
[ ! -f "$ACME_FILE" ] && touch "$ACME_FILE"
chmod 600 "$ACME_FILE"

echo "=== TRAEFIK CONFIG DIR ==="
mkdir -p "$TRAEFIK_CONFIG_DIR"

echo "=== RESOURCE MOUNT ROOT ==="
mkdir -p "$RESOURCE_MOUNT_ROOT"
chmod -R 777 "$RESOURCE_MOUNT_ROOT"

# Chown toàn bộ BASE_DIR về user thật (script chạy với sudo nên mọi file/thư mục mặc định owned bởi root)
chown -R "$REAL_USER" "$BASE_DIR"

echo "=== DOCKER NETWORK ==="
if ! docker network inspect tango_net >/dev/null 2>&1; then
  docker network create tango_net
fi

echo "=== WRITE .env ==="

# Load existing .env (idempotent — giữ secrets cũ khi chạy lại)
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
  existing_email=$(grep "^ACME_EMAIL="            "$ENV_FILE" 2>/dev/null | cut -d'=' -f2  || true)
  existing_domain=$(grep "^APP_DOMAIN="           "$ENV_FILE" 2>/dev/null | cut -d'=' -f2  || true)
  existing_tls=$(grep "^APP_TLS_ENABLED="         "$ENV_FILE" 2>/dev/null | cut -d'=' -f2  || echo "false")
  existing_resource_mount_root=$(grep "^RESOURCE_MOUNT_ROOT="     "$ENV_FILE" 2>/dev/null | cut -d'=' -f2  || true)
  existing_resource_mount_root_app=$(grep "^RESOURCE_MOUNT_ROOT_APP=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2  || true)
  existing_jwt_secret=$(grep "^JWT_SECRET="            "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- || true)
  existing_data_encryption_key=$(grep "^DATA_ENCRYPTION_KEY=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- || true)
  existing_postgres_password=$(grep "^POSTGRES_PASSWORD="    "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- || true)
  existing_database_url=$(grep "^DATABASE_URL="         "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- || true)
fi

# Args override giá trị cũ; secrets giữ nguyên nếu đã tồn tại
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

# --https yêu cầu --email
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

chmod 600 "$ENV_FILE"
chown "$REAL_USER" "$ENV_FILE"

echo "Let's Encrypt : ${final_email:-DISABLED}"
echo "App domain    : ${final_domain:-not set (configure in Settings)}"
echo "App HTTPS     : $final_tls"
echo "Mount root    : $final_resource_mount_root"
echo "Secrets       : generated/preserved in $ENV_FILE"

echo "=== TRAEFIK STATIC CONFIG ==="
write_traefik_static_config "$final_email"
echo "Traefik config : $TRAEFIK_STATIC_CONFIG"

resolve_cli_version
install_cli
write_cli_config

echo "=== FIX DOCKER SOCKET PERMISSION ==="
fix_docker_socket() {
  local plist="/Library/LaunchDaemons/fix-docker-sock.plist"

  # Set permission ngay lập tức
  chmod 666 /var/run/docker.sock 2>/dev/null || true

  # Cài launchd daemon để tự fix lại mỗi khi Docker Desktop restart
  if [ ! -f "$plist" ]; then
    echo "→ Installing launchd daemon to persist docker socket permissions..."
    cat > "$plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>fix-docker-sock</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/chmod</string>
        <string>666</string>
        <string>/var/run/docker.sock</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>WatchPaths</key>
    <array>
        <string>/var/run/docker.sock</string>
    </array>
</dict>
</plist>
PLIST
    launchctl load "$plist" 2>/dev/null || true
    echo "Docker socket  : permission fixed + persisted via launchd ✓"
  else
    echo "Docker socket  : launchd daemon already installed ✓"
  fi
}
fix_docker_socket


docker compose -f "$COMPOSE_FILE" pull

echo "=== START SERVICES ==="
docker compose -f "$COMPOSE_FILE" up -d
echo "Services started."

# Docker tạo thêm subfolder trong resource-volumes lúc start → fix permission lại
chmod -R 777 "$RESOURCE_MOUNT_ROOT" 2>/dev/null || true
chown -R "$REAL_USER" "$BASE_DIR" 2>/dev/null || true

echo "=== INSTALL DAEMON SERVICE ==="
"$CLI_BINARY_TARGET" daemon install

echo "=== CHECK ORCHESTRATION STATUS ==="
"$CLI_BINARY_TARGET" status       || true
"$CLI_BINARY_TARGET" daemon status || true
"$CLI_BINARY_TARGET" service list  || true

echo ""
echo "=================================="
echo " $APP_NAME is installed! (macOS)"
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
  echo " App URL        : http://localhost:8080"
  echo "                  (set domain in Settings UI)"
fi
echo "=================================="