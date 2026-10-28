$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$sqlPath = Join-Path $root 'scripts/dev-clear-demo-agents.sql'
if (-not (Test-Path -LiteralPath $sqlPath)) {
    throw "Missing SQL file: $sqlPath"
}
Get-Content -LiteralPath $sqlPath -Raw | docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
if ($LASTEXITCODE -ne 0) {
    throw "Clearing demo clients failed (exit code $LASTEXITCODE)."
}
