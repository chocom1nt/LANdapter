$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
$EnvPath = Join-Path $Root '.env'
if (-not (Test-Path -LiteralPath $EnvPath)) {
    throw "Missing .env. Run 'make setup' first."
}

$match = Select-String -LiteralPath $EnvPath -Pattern '^\s*LANDAPTER_ADMIN_TOKEN\s*=' | Select-Object -First 1
if (-not $match) {
    throw "LANDAPTER_ADMIN_TOKEN is missing. Run 'make setup' first."
}
$token = ($match.Line -split '=', 2)[1].Trim().Trim('"').Trim("'")
if ($token.Length -lt 32 -or $token.StartsWith('REPLACE_WITH_')) {
    throw "LANDAPTER_ADMIN_TOKEN is missing or invalid. Run 'make setup' first."
}
Set-Clipboard -Value $token
$token = $null
Write-Host 'Admin token copied to the clipboard. Paste it into the web Settings page.'
Write-Host 'Treat it as a secret; do not paste it into logs, chats, or tickets.'
