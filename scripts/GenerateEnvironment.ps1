$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$EnvPath = Join-Path $Root '.env'

function New-Secret {
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $rng.GetBytes($bytes)
        return [BitConverter]::ToString($bytes).Replace('-', '').ToLowerInvariant()
    }
    finally {
        $rng.Dispose()
    }
}

function Get-EnvRawValue([string]$Key) {
    $pattern = '^\s*' + [regex]::Escape($Key) + '\s*='
    foreach ($line in $script:EnvLines) {
        if ($line -match $pattern) {
            $index = $line.IndexOf('=')
            if ($index -ge 0) { return $line.Substring($index + 1).Trim() }
        }
    }
    return ''
}

function Unquote-EnvValue([string]$Value) {
    $value = $Value.Trim()
    if ($value.Length -ge 2 -and $value.StartsWith('"') -and $value.EndsWith('"')) {
        return $value.Substring(1, $value.Length - 2)
    }
    if ($value.Length -ge 2 -and $value.StartsWith("'") -and $value.EndsWith("'")) {
        return $value.Substring(1, $value.Length - 2)
    }
    return $value
}

function Set-EnvRawValue([string]$Key, [string]$Value) {
    $pattern = '^\s*' + [regex]::Escape($Key) + '\s*='
    $newLines = [System.Collections.Generic.List[string]]::new()
    $replaced = $false
    foreach ($line in $script:EnvLines) {
        if ($line -match $pattern) {
            if (-not $replaced) {
                $newLines.Add("$Key=$Value")
                $replaced = $true
            }
            # Remove duplicate definitions so all tools see the same value.
            continue
        }
        $newLines.Add($line)
    }
    if (-not $replaced) { $newLines.Add("$Key=$Value") }
    $script:EnvLines = $newLines
}

if (Test-Path -LiteralPath $EnvPath) {
    $EnvLines = [System.Collections.Generic.List[string]]::new()
    foreach ($line in Get-Content -LiteralPath $EnvPath) { $EnvLines.Add([string]$line) }
} else {
    $EnvLines = [System.Collections.Generic.List[string]]::new()
}

$userRaw = Get-EnvRawValue 'POSTGRES_USER'
$user = Unquote-EnvValue $userRaw
if ([string]::IsNullOrWhiteSpace($user) -or $user.StartsWith('REPLACE_WITH_')) {
    $userRaw = 'landapter'
}
Set-EnvRawValue 'POSTGRES_USER' $userRaw

$passwordRaw = Get-EnvRawValue 'POSTGRES_PASSWORD'
$password = Unquote-EnvValue $passwordRaw
if ([string]::IsNullOrWhiteSpace($password) -or $password.StartsWith('REPLACE_WITH_')) {
    $passwordRaw = New-Secret
}
Set-EnvRawValue 'POSTGRES_PASSWORD' $passwordRaw

$adminTokenRaw = Get-EnvRawValue 'LANDAPTER_ADMIN_TOKEN'
$adminToken = Unquote-EnvValue $adminTokenRaw
if ([string]::IsNullOrWhiteSpace($adminToken) -or $adminToken.StartsWith('REPLACE_WITH_') -or $adminToken.Length -lt 32) {
    $adminTokenRaw = New-Secret
}
Set-EnvRawValue 'LANDAPTER_ADMIN_TOKEN' $adminTokenRaw

$tempPath = Join-Path $Root ('.env.tmp.' + [guid]::NewGuid().ToString('N'))
try {
    $linesArray = $EnvLines.ToArray()
    [System.IO.File]::WriteAllLines($tempPath, $linesArray, [System.Text.UTF8Encoding]::new($false))
    Move-Item -LiteralPath $tempPath -Destination $EnvPath -Force
}
finally {
    if (Test-Path -LiteralPath $tempPath) { Remove-Item -LiteralPath $tempPath -Force }
}

Write-Host "Environment ready: .env contains POSTGRES_USER, POSTGRES_PASSWORD and LANDAPTER_ADMIN_TOKEN."
Write-Host "Existing non-placeholder secrets were preserved; missing or placeholder secrets were generated."
Write-Host "The Master and lndp load .env automatically when run from the project root."
