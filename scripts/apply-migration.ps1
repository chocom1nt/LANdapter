param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9]{3}_[A-Za-z0-9_-]+$')]
    [string]$Name
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$sqlPath = Join-Path (Join-Path $root 'migrations') ($Name + '.up.sql')
if (-not (Test-Path -LiteralPath $sqlPath)) {
    throw "Migration file not found: $sqlPath"
}
$sql = Get-Content -LiteralPath $sqlPath -Raw
$sql | docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
if ($LASTEXITCODE -ne 0) {
    throw "Migration $Name failed (exit code $LASTEXITCODE)."
}
