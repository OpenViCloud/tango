#!/bin/bash

set -e

APP_NAME="tango"
BASE_DIR="$(pwd)"
LETSENCRYPT_DIR="$BASE_DIR/letsencrypt"
ACME_FILE="$LETSENCRYPT_DIR/acme.json"
TRAEFIK_CONFIG_DIR="$BASE_DIR/traefik/config"
ENV_FILE="$BASE_DIR/.env"

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

# ── Main ──────────────────────────────────────────────────────────────────────

echo "=== SETUP: $APP_NAME ==="

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

echo "=== WRITE .env ==="

# Load existing .env values if file exists
existing_email=""
existing_domain=""
existing_tls="false"
if [ -f "$ENV_FILE" ]; then
  existing_email=$(grep "^TRAEFIK_ACME_EMAIL=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
  existing_domain=$(grep "^APP_DOMAIN=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
  existing_tls=$(grep "^APP_TLS_ENABLED=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "false")
fi

# Resolve final values (args override existing)
final_email="${EMAIL:-$existing_email}"
final_domain="${DOMAIN:-$existing_domain}"
final_tls="${TLS_ENABLED:-$existing_tls}"

# --https requires --email (for Let's Encrypt)
if [ "$final_tls" = "true" ] && [ -z "$final_email" ]; then
  echo "Warning: --https requires --email for Let's Encrypt. HTTPS disabled."
  final_tls="false"
fi

cat > "$ENV_FILE" <<EOF
TRAEFIK_ACME_EMAIL=$final_email
APP_DOMAIN=$final_domain
APP_TLS_ENABLED=$final_tls
EOF

echo "Let's Encrypt : ${final_email:-DISABLED}"
echo "App domain    : ${final_domain:-not set (configure in Settings)}"
echo "App HTTPS     : $final_tls"

echo "=== PULL LATEST IMAGES ==="
docker compose pull

echo "=== START SERVICES ==="
docker compose up -d

echo ""
echo "=================================="
echo " $APP_NAME is up!"
echo " ACME file      : $ACME_FILE"
echo " Traefik config : $TRAEFIK_CONFIG_DIR"
echo " ENV file       : $ENV_FILE"
if [ -n "$final_domain" ]; then
  proto="http"
  [ "$final_tls" = "true" ] && proto="https"
  echo " App URL        : $proto://$final_domain"
else
  echo " App URL        : http://<server-ip>:8080"
  echo "                  (set domain in Settings UI)"
fi
echo "=================================="

# Usage:
# chmod +x install.sh
# ./install.sh --email admin@example.com --domain app.example.com --https
