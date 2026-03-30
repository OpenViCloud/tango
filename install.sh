#!/bin/bash

set -e

APP_NAME="tango"
BASE_DIR="$(pwd)"
LETSENCRYPT_DIR="$BASE_DIR/letsencrypt"
ACME_FILE="$LETSENCRYPT_DIR/acme.json"
ENV_FILE="$BASE_DIR/.env"

EMAIL=""

# ── Parse args ────────────────────────────────────────────────────────────────

while [[ $# -gt 0 ]]; do
  case "$1" in
    --email)
      if [ -n "$2" ] && [[ ! "$2" =~ ^-- ]]; then
        EMAIL="$2"
        shift 2
      else
        echo "Error: --email requires a value"
        echo "Usage: ./install-dev.sh [--email your-email@example.com]"
        exit 1
      fi
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: ./install-dev.sh [--email your-email@example.com]"
      exit 1
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

echo "=== WRITE .env ==="
if [ -n "$EMAIL" ]; then
  # --email passed: always write/update with new email
  cat > "$ENV_FILE" <<EOF
TRAEFIK_ACME_EMAIL=$EMAIL
EOF
  echo "Let's Encrypt: ENABLED (email: $EMAIL)"
elif [ -f "$ENV_FILE" ] && grep -q "TRAEFIK_ACME_EMAIL=." "$ENV_FILE"; then
  # no --email passed but existing non-empty value found: keep it
  existing_email=$(grep "TRAEFIK_ACME_EMAIL=" "$ENV_FILE" | cut -d'=' -f2)
  echo "Let's Encrypt: keeping existing email ($existing_email)"
else
  # no email anywhere: write empty (TLS disabled)
  cat > "$ENV_FILE" <<EOF
TRAEFIK_ACME_EMAIL=
EOF
  echo "Let's Encrypt: DISABLED (pass --email to enable)"
fi

echo "=== PULL LATEST IMAGES ==="
docker compose pull

echo "=== START SERVICES ==="
docker compose up -d

echo ""
echo "=================================="
echo " $APP_NAME is up!"
echo " ACME file : $ACME_FILE"
echo " ENV file  : $ENV_FILE"
echo "=================================="


# RUN
# chmod +x install-dev.sh
# ./install-dev.sh --email new@tango.cloud
