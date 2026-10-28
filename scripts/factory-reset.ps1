[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $Root

# Capture credential variable names only. Values are never printed.
$CredentialEnvironmentVariables = @(
    Get-ChildItem Env: |
        Where-Object { $_.Name -match '^(?:LANDAPTER_)?(?:ADMIN|AGENT|CLIENT)_(?:TOKEN|UUID|ID)$' } |
        Select-Object -ExpandProperty Name
)

function Invoke-Compose {
    param([Parameter(Mandatory = $true)][string[]]$ComposeArgs)
    & docker compose @ComposeArgs
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose $($ComposeArgs -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Set-AdminTokenInEnvFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Token
    )

    $content = [System.IO.File]::ReadAllText($Path)
    $newline = if ($content.Contains("`r`n")) { "`r`n" } else { "`n" }
    $hadTrailingNewline = $content.EndsWith("`n")
    $lines = [System.Collections.Generic.List[string]]::new()
    foreach ($line in [regex]::Split($content, "\r?\n")) {
        $lines.Add($line)
    }
    if ($hadTrailingNewline -and $lines.Count -gt 0 -and $lines[$lines.Count - 1] -eq '') {
        $lines.RemoveAt($lines.Count - 1)
    }

    $adminTokenCount = 0
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\s*(?:LANDAPTER_)?(?:AGENT|CLIENT)_(?:ID|UUID|TOKEN)\s*=') {
            $lines[$i] = '# agent/client credential removed by factory reset'
        } elseif ($lines[$i] -match '^\s*LANDAPTER_ADMIN_TOKEN\s*=') {
            if ($adminTokenCount -eq 0) {
                $lines[$i] = "LANDAPTER_ADMIN_TOKEN=$Token"
            } else {
                $lines[$i] = '# duplicate LANDAPTER_ADMIN_TOKEN entry removed by factory reset'
            }
            $adminTokenCount++
        }
    }
    if ($adminTokenCount -eq 0) {
        $lines.Add("LANDAPTER_ADMIN_TOKEN=$Token")
    }

    $output = $lines -join $newline
    if ($hadTrailingNewline -or -not $output.EndsWith($newline)) {
        $output += $newline
    }
    $encoding = [System.Text.UTF8Encoding]::new($false)
    $tempPath = Join-Path (Split-Path -Parent $Path) ('landapter-env-reset-' + [guid]::NewGuid().ToString('N') + '.tmp')
    try {
        [System.IO.File]::WriteAllText($tempPath, $output, $encoding)
        [System.IO.File]::Replace($tempPath, $Path, $null)
    } finally {
        if (Test-Path $tempPath) { Remove-Item -LiteralPath $tempPath -Force -ErrorAction SilentlyContinue }
    }
}

$envPath = Join-Path $Root '.env'
$sqlPath = Join-Path $Root 'scripts/factory-reset.sql'
$agentExample = Join-Path $Root 'configs/agent.yaml.example'
$agentConfig = Join-Path $Root 'configs/agent.yaml'
$masterExample = Join-Path $Root 'configs/master.yaml.example'
$masterConfig = Join-Path $Root 'configs/master.yaml'
$uploadsPath = Join-Path $Root 'uploads'
$logsPath = Join-Path $Root 'logs'

if (-not (Test-Path $envPath -PathType Leaf)) {
    throw 'Missing .env. Refusing to reset without a project environment file.'
}
if (-not (Test-Path $sqlPath -PathType Leaf)) {
    throw 'Missing scripts/factory-reset.sql.'
}
if (-not (Test-Path $agentExample -PathType Leaf)) {
    throw 'Missing configs/agent.yaml.example; cannot safely reset the local agent config.'
}
if (-not (Test-Path $masterExample -PathType Leaf)) {
    throw 'Missing configs/master.yaml.example; cannot safely reset the Master config.'
}
foreach ($sensitivePath in @($envPath, $agentExample, $masterExample)) {
    $sensitiveItem = Get-Item $sensitivePath -Force
    if (($sensitiveItem.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "Refusing to read or modify linked path: $sensitivePath"
    }
}
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw 'Docker CLI was not found.'
}

Write-Host ''
Write-Host 'LANdapter FACTORY RESET' -ForegroundColor Yellow
Write-Host 'This will:'
Write-Host '  - truncate application data tables in the PostgreSQL public schema'
Write-Host '  - rotate LANDAPTER_ADMIN_TOKEN in .env'
Write-Host '  - reset configs/master.yaml and configs/agent.yaml from their .example files'
Write-Host '  - delete files in uploads/ and logs/ and root-level *.uuid / *.pid files'
Write-Host '  - recreate the Master container'
Write-Host ''
Write-Host 'Agent credentials stored in the database will be invalidated. Re-enroll agents after reset.' -ForegroundColor Yellow
$confirmation = Read-Host 'Type RESET-LANDAPTER to continue'
if ($confirmation -cne 'RESET-LANDAPTER') {
    Write-Host 'Reset cancelled.'
    exit 1
}

# Verify Compose and the target database before changing any secrets or data.
Invoke-Compose @('config', '--quiet')
$dbName = (& docker compose exec -T postgres sh -c 'printf %s "$POSTGRES_DB"').Trim()
if ($LASTEXITCODE -ne 0) { throw 'Unable to read PostgreSQL database name. Ensure the postgres service is running.' }
if ($dbName -ne 'landapter') { throw "Refusing to truncate unexpected database '$dbName'. Expected 'landapter'." }
# Fail closed if uploads/logs are junctions or symlinks to paths outside the repo.
foreach ($path in @($uploadsPath, $logsPath, $agentConfig, $masterConfig)) {
    if (Test-Path $path) {
        $item = Get-Item $path -Force
        if (($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -ne 0) {
            throw "Refusing to modify linked path: $path"
        }
        if ($path -in @($agentConfig, $masterConfig) -and $item.PSIsContainer) {
            throw "Expected a configuration file, but found a directory: $path"
        }
        if ($path -in @($uploadsPath, $logsPath) -and -not $item.PSIsContainer) {
            throw "Expected a directory, but found a file: $path"
        }
    }
}

$masterStopped = $false
try {
    $randomBytes = [byte[]]::new(32)
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($randomBytes) } finally { $rng.Dispose() }
    $newToken = ([System.BitConverter]::ToString($randomBytes)).Replace('-', '').ToLowerInvariant()

    Invoke-Compose @('stop', 'master')
    $masterStopped = $true

    # The old token is invalidated by this reset. Ensure Compose reads the updated .env.
    Remove-Item Env:LANDAPTER_ADMIN_TOKEN -ErrorAction SilentlyContinue
    Set-AdminTokenInEnvFile -Path $envPath -Token $newToken

    Invoke-Compose @('cp', 'scripts/factory-reset.sql', 'postgres:/tmp/landapter-factory-reset.sql')
    Invoke-Compose @('exec', '-T', 'postgres', 'sh', '-c', 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f /tmp/landapter-factory-reset.sql')
    & docker compose exec -T postgres rm -f /tmp/landapter-factory-reset.sql

    Copy-Item -LiteralPath $agentExample -Destination $agentConfig -Force
    Copy-Item -LiteralPath $masterExample -Destination $masterConfig -Force

    foreach ($dir in @($uploadsPath, $logsPath)) {
        if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir | Out-Null }
        Get-ChildItem -LiteralPath $dir -Force | Remove-Item -Recurse -Force
    }
    Get-ChildItem -LiteralPath $Root -File -Force | Where-Object { $_.Name -like '*.uuid' -or $_.Name -like '*.pid' } | Remove-Item -Force

    Invoke-Compose @('up', '-d', '--force-recreate', 'master')
    $masterStopped = $false
    Write-Host ''
    Write-Host 'Factory reset complete.' -ForegroundColor Green
    Write-Host 'A new admin token is stored in .env. Read it from that file; it was not printed by this script.'
    Write-Host 'Existing agent credentials were invalidated. Re-enroll each agent before reconnecting it.'
} catch {
    Write-Host "ERROR: $_" -ForegroundColor Red
    if ($masterStopped) {
        Write-Warning 'Attempting to start Master again; check its logs if it does not become healthy.'
        try { & docker compose up -d master } catch { Write-Warning 'Automatic restart failed.' }
    }
    exit 1
} finally {
    # Clear stale credential variables in this PowerShell process. This works in
    # PowerShell because environment variables have process scope; the Bash
    # script can only warn because a child process cannot mutate its parent.
    foreach ($name in $CredentialEnvironmentVariables) {
        Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
    }
}
