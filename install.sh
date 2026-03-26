#!/bin/sh
# install.sh — cài CLI cho Linux/macOS
# Usage: curl -fsSL https://yourdomain.com/install.sh | bash

set -e

VERSION="0.1.0"
BINARY="demo"
BASE_URL="https://github.com/yourname/tango/releases/download/v${VERSION}"
CONFIG_DIR="${HOME}/.config/demo"
CONFIG_FILE="${CONFIG_DIR}/config.json"
SYSTEM_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="${HOME}/.local/bin"

resolve_install_dir() {
  if [ -d "$SYSTEM_INSTALL_DIR" ] && [ -w "$SYSTEM_INSTALL_DIR" ]; then
    INSTALL_DIR="$SYSTEM_INSTALL_DIR"
    INSTALL_MODE="system"
    return 0
  fi

  if [ ! -d "$SYSTEM_INSTALL_DIR" ] && [ -w "$(dirname "$SYSTEM_INSTALL_DIR")" ]; then
    mkdir -p "$SYSTEM_INSTALL_DIR"
    INSTALL_DIR="$SYSTEM_INSTALL_DIR"
    INSTALL_MODE="system"
    return 0
  fi

  mkdir -p "$USER_INSTALL_DIR"
  chmod 700 "$USER_INSTALL_DIR"
  INSTALL_DIR="$USER_INSTALL_DIR"
  INSTALL_MODE="user"
}

load_existing_llm_key() {
  if [ ! -f "$CONFIG_FILE" ]; then
    return 0
  fi

  sed -n 's/.*"llm_config_encryption_key"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$CONFIG_FILE" | head -n 1
}

generate_llm_key() {
  LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32
}

prompt_llm_key() {
  existing_key=$(load_existing_llm_key || true)

  if [ -n "$existing_key" ]; then
    printf 'Found existing LLM_CONFIG_ENCRYPTION_KEY in %s\n' "$CONFIG_FILE"
    printf 'Press Enter to keep it, type "random" to generate a new one, or enter a new 32-character key.\n'
    printf '> '
    IFS= read -r input
    if [ -z "$input" ]; then
      LLM_KEY="$existing_key"
      return 0
    fi
  else
    printf 'LLM_CONFIG_ENCRYPTION_KEY is required for encrypting provider API keys stored in DB.\n'
    printf 'Enter a 32-character key, or press Enter to generate a random one.\n'
    printf '> '
    IFS= read -r input
  fi

  if [ "$input" = "random" ] || [ -z "$input" ]; then
    LLM_KEY=$(generate_llm_key)
    printf 'Generated key: %s\n' "$LLM_KEY"
  else
    LLM_KEY="$input"
  fi

  if [ "${#LLM_KEY}" -ne 32 ]; then
    echo "❌ LLM_CONFIG_ENCRYPTION_KEY must be exactly 32 characters."
    exit 1
  fi
}

write_llm_key_to_config() {
  umask 077
  mkdir -p "$CONFIG_DIR"
  chmod 700 "$CONFIG_DIR"

  tmp_file="${CONFIG_FILE}.tmp"
  escaped_key=$(printf '%s' "$LLM_KEY" | sed 's/[\/&]/\\&/g')

  if [ ! -f "$CONFIG_FILE" ] || [ ! -s "$CONFIG_FILE" ]; then
    cat >"$CONFIG_FILE" <<EOF
{
  "llm_config_encryption_key": "$LLM_KEY"
}
EOF
    chmod 600 "$CONFIG_FILE"
    return 0
  fi

  if grep -q '"llm_config_encryption_key"[[:space:]]*:' "$CONFIG_FILE"; then
    sed 's/"llm_config_encryption_key"[[:space:]]*:[[:space:]]*"[^"]*"/"llm_config_encryption_key": "'"$escaped_key"'"/' "$CONFIG_FILE" >"$tmp_file"
    mv "$tmp_file" "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"
    return 0
  fi

  awk -v key="$LLM_KEY" '
    {
      lines[NR] = $0
    }
    END {
      if (NR == 0) {
        print "{"
        print "  \"llm_config_encryption_key\": \"" key "\""
        print "}"
        next
      }

      insert_at = NR
      for (i = NR; i >= 1; i--) {
        if (lines[i] ~ /^[[:space:]]*}[[:space:]]*$/) {
          insert_at = i
          break
        }
      }

      for (i = 1; i < insert_at; i++) {
        line = lines[i]
        if (i == insert_at - 1 && line ~ /"[[:space:]]*$/ && line !~ /,[[:space:]]*$/) {
          line = line ","
        }
        print line
      }
      print "  \"llm_config_encryption_key\": \"" key "\""
      print lines[insert_at]
      for (i = insert_at + 1; i <= NR; i++) {
        print lines[i]
      }
    }
  ' "$CONFIG_FILE" >"$tmp_file"

  mv "$tmp_file" "$CONFIG_FILE"
  chmod 600 "$CONFIG_FILE"
}

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "❌ Unsupported arch: $ARCH"; exit 1 ;;
esac

FILENAME="${BINARY}_${OS}_${ARCH}"
URL="${BASE_URL}/${FILENAME}"

prompt_llm_key
write_llm_key_to_config
resolve_install_dir

echo "⬇️  Downloading demo CLI v${VERSION} for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o "/tmp/${BINARY}"
chmod +x "/tmp/${BINARY}"
mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod 755 "${INSTALL_DIR}/${BINARY}"

echo "✅ Installed! LLM_CONFIG_ENCRYPTION_KEY saved to ${CONFIG_FILE}"
if [ "$INSTALL_MODE" = "user" ]; then
  echo "Installed binary to ${INSTALL_DIR}/${BINARY}"
  case ":$PATH:" in
    *":${USER_INSTALL_DIR}:"*) ;;
    *)
      echo "Add this to your shell profile if needed:"
      echo "  export PATH=\"${USER_INSTALL_DIR}:\$PATH\""
      ;;
  esac
fi
echo "Try: ${BINARY} onboard"
