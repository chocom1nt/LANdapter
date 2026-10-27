#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
umask 077

if ! command -v openssl >/dev/null 2>&1; then
  echo "OpenSSL is required to generate secure secrets. Install it and retry." >&2
  exit 1
fi

new_secret() { openssl rand -hex 32; }

env_value() {
  local key="$1"
  [[ -f .env ]] || return 0
  awk -v key="$key" '
    /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
    {
      line=$0
      sub(/^[[:space:]]+/, "", line)
      if (index(line, key "=") == 1) {
        sub(/^[^=]*=/, "", line)
        print line
        exit
      }
    }
  ' .env
}

unquote_for_check() {
  local value="$1"
  value="${value%$'\r'}"
  if [[ ${#value} -ge 2 && ${value:0:1} == '"' && ${value: -1} == '"' ]]; then
    value="${value:1:${#value}-2}"
  elif [[ ${#value} -ge 2 && ${value:0:1} == "'" && ${value: -1} == "'" ]]; then
    value="${value:1:${#value}-2}"
  fi
  printf '%s' "$value"
}

set_env_value() {
  local key="$1" value="$2"
  local tmp
  tmp="$(mktemp .env.tmp.XXXXXX)"
  if [[ -f .env ]]; then
    awk -v key="$key" -v value="$value" '
      BEGIN { replaced=0 }
      /^[[:space:]]*#/ { print; next }
      {
        trimmed=$0
        sub(/^[[:space:]]+/, "", trimmed)
        if (index(trimmed, key "=") == 1) {
          if (!replaced) print key "=" value
          replaced=1
          next
        }
        print
      }
      END { if (!replaced) print key "=" value }
    ' .env > "$tmp"
  else
    printf '%s=%s\n' "$key" "$value" > "$tmp"
  fi
  chmod 600 "$tmp"
  mv -f -- "$tmp" .env
}

ensure_setting() {
  local key="$1" default_value="$2" kind="$3"
  local raw value placeholder=false
  raw="$(env_value "$key")"
  value="$(unquote_for_check "$raw")"

  if [[ "$value" == REPLACE_WITH_* || -z "$value" ]]; then
    placeholder=true
  fi

  case "$kind" in
    user)
      if [[ "$placeholder" == true ]]; then
        set_env_value "$key" "$default_value"
      fi
      return
      ;;
    password)
      if [[ "$placeholder" == true ]]; then
        set_env_value "$key" "$(new_secret)"
      fi
      return
      ;;
    token)
      if [[ "$placeholder" == true || ${#value} -lt 32 ]]; then
        set_env_value "$key" "$(new_secret)"
      fi
      return
      ;;
    *)
      echo "Internal setup error: unknown setting type." >&2
      exit 1
      ;;
  esac
}

# Repair missing or placeholder entries without rotating existing valid secrets.
ensure_setting POSTGRES_USER landapter user
ensure_setting POSTGRES_PASSWORD "" password
ensure_setting LANDAPTER_ADMIN_TOKEN "" token
chmod 600 .env

echo "Environment ready: .env contains POSTGRES_USER, POSTGRES_PASSWORD and LANDAPTER_ADMIN_TOKEN."
echo "Existing non-placeholder secrets were preserved; missing or placeholder secrets were generated."
echo "The Master and lndp load .env automatically when run from the project root."
