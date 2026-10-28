#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -f .env ]]; then
  echo "Missing .env. Run 'make setup' first." >&2
  exit 1
fi

token="$(awk -F= '$1 == "LANDAPTER_ADMIN_TOKEN" { print substr($0, index($0, "=") + 1); exit }' .env)"
token="${token%$'\r'}"
if [[ ${#token} -ge 2 && ${token:0:1} == '"' && ${token: -1} == '"' ]]; then
  token="${token:1:${#token}-2}"
elif [[ ${#token} -ge 2 && ${token:0:1} == "'" && ${token: -1} == "'" ]]; then
  token="${token:1:${#token}-2}"
fi
if [[ ${#token} -lt 32 || "$token" == REPLACE_WITH_* ]]; then
  echo "LANDAPTER_ADMIN_TOKEN is missing or invalid. Run 'make setup' first." >&2
  exit 1
fi

if command -v clip.exe >/dev/null 2>&1; then
  printf '%s' "$token" | clip.exe
  echo "Admin token copied to the Windows clipboard."
elif command -v wl-copy >/dev/null 2>&1; then
  printf '%s' "$token" | wl-copy
  echo "Admin token copied to the clipboard."
elif command -v xclip >/dev/null 2>&1; then
  printf '%s' "$token" | xclip -selection clipboard
  echo "Admin token copied to the clipboard."
elif command -v pbcopy >/dev/null 2>&1; then
  printf '%s' "$token" | pbcopy
  echo "Admin token copied to the clipboard."
else
  echo "No clipboard utility found; printing the token because this command was explicitly requested:"
  printf '%s\n' "$token"
  echo "Treat it as a secret and do not paste it into logs or tickets." >&2
fi
unset token
