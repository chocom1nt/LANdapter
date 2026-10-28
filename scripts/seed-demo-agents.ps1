$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$sqlPath = Join-Path $root 'scripts/dev-seed-agents.sql'
if (-not (Test-Path -LiteralPath $sqlPath)) {
    throw "Missing SQL file: $sqlPath"
}
# Ask Docker to run psql inside the PostgreSQL service and feed the SQL on stdin.
Get-Content -LiteralPath $sqlPath -Raw | docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
if ($LASTEXITCODE -ne 0) {
    throw "Seeding demo clients failed (exit code $LASTEXITCODE)."
}
