$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

& (Join-Path $PSScriptRoot 'GenerateEnvironment.ps1')

function Copy-ExampleIfMissing([string]$Example, [string]$Target) {
    if (-not (Test-Path -LiteralPath $Target)) {
        Copy-Item -LiteralPath $Example -Destination $Target
        Write-Host "Created $Target from $Example"
    } else {
        Write-Host "Preserved existing $Target"
    }
}

Copy-ExampleIfMissing 'configs/master.yaml.example' 'configs/master.yaml'
Copy-ExampleIfMissing 'configs/agent.yaml.example' 'configs/agent.yaml'
New-Item -ItemType Directory -Force -Path 'uploads', 'logs' | Out-Null

Write-Host ''
Write-Host 'LANdapter setup files are ready.'
Write-Host "Next: run 'make up' to start PostgreSQL and Master in Docker, or use 'make quickstart' for first-time setup."
Write-Host "For an existing database, follow docs/en/INSTALL.md; Docker init scripts only run on a new volume."
Write-Host "For web Settings, run 'make copy-token' and paste the clipboard token once."
Write-Host 'The lndp TUI reads LANDAPTER_ADMIN_TOKEN from .env automatically.'
