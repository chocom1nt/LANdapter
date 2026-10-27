#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

bash scripts/GenerateEnvironment.sh

copy_example() {
  local example="$1" target="$2"
  if [[ ! -e "$target" ]]; then
    cp "$example" "$target"
    echo "Created $target from $example"
  else
    echo "Preserved existing $target"
  fi
}

copy_example configs/master.yaml.example configs/master.yaml
copy_example configs/agent.yaml.example configs/agent.yaml
mkdir -p uploads logs

echo
echo "LANdapter setup files are ready."
echo "Next: run 'make up' to start PostgreSQL and Master in Docker, or use 'make quickstart' for first-time setup."
echo "For an existing database, follow the upgrade steps in docs/en/INSTALL.md; Docker init scripts only run on a new volume."
echo "For the web Settings page, run 'make copy-token' and paste the clipboard token once."
echo "The lndp TUI reads LANDAPTER_ADMIN_TOKEN from .env automatically."
