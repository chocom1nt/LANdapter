if (Test-Path .env) {
    throw ".env file already exists. Don't overwrite it — add missing variables manually."
}

function New-Secret {
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $rng.GetBytes($bytes)
        return [BitConverter]::ToString($bytes).Replace("-", "").ToLowerInvariant()
    }
    finally {
        $rng.Dispose()
    }
}

$dbPassword = New-Secret
$adminToken = New-Secret

@"
POSTGRES_USER=landapter
POSTGRES_PASSWORD=$dbPassword
LANDAPTER_ADMIN_TOKEN=$adminToken
"@ | Set-Content -Encoding ascii .env

Write-Host ".env created. Secrets generated."