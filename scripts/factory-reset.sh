#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# A child Bash process cannot unset variables in its parent terminal. Capture
# only credential variable names (never their values) so we can warn afterward.
stale_credential_vars="$(compgen -e | grep -E '^(LANDAPTER_)?(ADMIN|AGENT|CLIENT)_(TOKEN|UUID|ID)$' | tr '\n' ' ' || true)"

fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || fail 'Docker CLI was not found.'
[[ -f .env ]] || fail 'Missing .env. Refusing to reset without a project environment file.'
[[ -f scripts/factory-reset.sql ]] || fail 'Missing scripts/factory-reset.sql.'
[[ -f configs/agent.yaml.example ]] || fail 'Missing configs/agent.yaml.example; cannot safely reset local agent config.'
[[ -f configs/master.yaml.example ]] || fail 'Missing configs/master.yaml.example; cannot safely reset Master config.'

# Do not follow linked paths while reading secrets or deleting runtime files.
[[ ! -L .env ]] || fail '.env is a symlink; refusing to edit it.'
[[ ! -L configs/agent.yaml.example ]] || fail 'configs/agent.yaml.example is a symlink; refusing to read it.'
[[ ! -L configs/master.yaml.example ]] || fail 'configs/master.yaml.example is a symlink; refusing to read it.'
[[ ! -L configs/master.yaml ]] || fail 'configs/master.yaml is a symlink; refusing to overwrite it.'
[[ ! -L uploads ]] || fail 'uploads/ is a symlink; refusing to delete through it.'
[[ ! -L logs ]] || fail 'logs/ is a symlink; refusing to delete through it.'
[[ ! -L configs/agent.yaml ]] || fail 'configs/agent.yaml is a symlink; refusing to overwrite it.'
[[ ! -e uploads || -d uploads ]] || fail 'uploads exists but is not a directory.'
[[ ! -e logs || -d logs ]] || fail 'logs exists but is not a directory.'
[[ ! -e configs/agent.yaml || -f configs/agent.yaml ]] || fail 'configs/agent.yaml exists but is not a file.'
[[ ! -e configs/master.yaml || -f configs/master.yaml ]] || fail 'configs/master.yaml exists but is not a file.'

echo
echo 'LANdapter FACTORY RESET'
echo 'This will:'
echo '  - truncate application data tables in the PostgreSQL public schema'
echo '  - rotate LANDAPTER_ADMIN_TOKEN in .env'
echo '  - reset configs/master.yaml and configs/agent.yaml from their .example files'
echo '  - delete files in uploads/ and logs/ and root-level *.uuid / *.pid files'
echo '  - recreate the Master container'
echo
echo 'Agent credentials stored in the database will be invalidated. Re-enroll agents after reset.'
read -r -p 'Type RESET-LANDAPTER to continue: ' confirmation
[[ "$confirmation" == 'RESET-LANDAPTER' ]] || fail 'Reset cancelled.'

docker compose config --quiet
db_name="$(docker compose exec -T postgres sh -c 'printf %s "$POSTGRES_DB"')" || fail 'Unable to read PostgreSQL database name. Ensure postgres is running.'
[[ "$db_name" == 'landapter' ]] || fail "Refusing to truncate unexpected database '${db_name}'. Expected 'landapter'."

new_token="$(openssl rand -hex 32)" || fail 'Could not generate a new admin token (openssl is required).'
[[ ${#new_token} -eq 64 ]] || fail 'Generated token had an unexpected length.'

# Atomic .env replacement without producing a backup copy of secrets.
umask 077
tmp_env="landapter-env-reset-$$-${RANDOM}${RANDOM}.tmp"
( set -o noclobber; : > "$tmp_env" ) || fail "Could not create temporary environment file: $tmp_env"
cleanup_tmp() {
  if [[ -n "${tmp_env:-}" && -f "$tmp_env" ]]; then
    rm -f -- "$tmp_env"
  fi
  return 0
}
trap cleanup_tmp EXIT
awk -v token="$new_token" '
  BEGIN { count = 0 }
  /^[[:space:]]*(LANDAPTER_)?(AGENT|CLIENT)_(ID|UUID|TOKEN)[[:space:]]*=/ { next }
  /^[[:space:]]*LANDAPTER_ADMIN_TOKEN[[:space:]]*=/ {
    if (count == 0) print "LANDAPTER_ADMIN_TOKEN=" token
    count++
    next
  }
  { print }
  END { if (count == 0) print "LANDAPTER_ADMIN_TOKEN=" token }
' .env > "$tmp_env"
chmod 600 "$tmp_env"
master_stopped=0
finish_reset() {
  status=$?
  trap - EXIT
  if [[ -n "${tmp_env:-}" && -f "$tmp_env" ]]; then rm -f -- "$tmp_env"; fi
  if [[ "$master_stopped" -eq 1 ]]; then
    echo 'Reset did not finish cleanly; attempting to start Master again.' >&2
    docker compose up -d master >/dev/null 2>&1 || true
  fi
  exit "$status"
}
trap finish_reset EXIT

docker compose stop master
master_stopped=1
unset LANDAPTER_ADMIN_TOKEN || true
mv -- "$tmp_env" .env
tmp_env=""

docker compose cp scripts/factory-reset.sql postgres:/tmp/landapter-factory-reset.sql
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f /tmp/landapter-factory-reset.sql'
docker compose exec -T postgres rm -f /tmp/landapter-factory-reset.sql || true

cp -- configs/agent.yaml.example configs/agent.yaml
cp -- configs/master.yaml.example configs/master.yaml
mkdir -p uploads logs
find uploads -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
find logs -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
find "$ROOT" -maxdepth 1 -type f \( -name '*.uuid' -o -name '*.pid' \) -delete

docker compose up -d --force-recreate master
master_stopped=0
trap - EXIT
cleanup_tmp

echo
echo 'Factory reset complete.'
echo 'A new admin token is stored in .env. Read it from that file; it was not printed by this script.'
echo 'Existing agent credentials were invalidated. Re-enroll each agent before reconnecting it.'
if [[ -n "$stale_credential_vars" ]]; then
  echo 'NOTE: Bash cannot clear variables in the calling terminal. Clear any stale credential variables there:'
  printf '  unset %s\n' "$stale_credential_vars"
fi
