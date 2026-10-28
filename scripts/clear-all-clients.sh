#!/usr/bin/env bash
set -euo pipefail

cat <<'WARN'
WARNING: this permanently deletes ALL clients from the configured LANdapter database.
Related device inventories, snapshots and job results are removed through foreign-key cascades.
Agent token hashes are stored on client rows and are deleted too; old tokens will not re-enroll agents.
Existing in-memory WebSocket sessions are not closed by SQL alone; stop agents or restart Master for a clean slate.
This does NOT delete uploaded installer files, jobs without client results, policies, or schedules.
WARN
read -r -p 'Type DELETE ALL CLIENTS to continue: ' confirmation
if [[ "$confirmation" != 'DELETE ALL CLIENTS' ]]; then
  echo 'Cancelled; nothing was deleted.'
  exit 1
fi

docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "DELETE FROM clients;"'
