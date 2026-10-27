$ErrorActionPreference = 'Stop'
Write-Host ''
Write-Host 'WARNING: this permanently deletes ALL clients from the configured LANdapter database.' -ForegroundColor Red
Write-Host 'Related device inventories, snapshots and job results are removed through foreign-key cascades.' -ForegroundColor Yellow
Write-Host 'Agent token hashes are deleted too; old tokens will not re-enroll agents.' -ForegroundColor Yellow
Write-Host 'SQL does not close existing WebSocket sessions; stop agents or restart Master for a clean slate.'
Write-Host 'Uploaded installer files, jobs without client results, policies, and schedules are not deleted.'
$confirmation = Read-Host 'Type DELETE ALL CLIENTS to continue'
if ($confirmation -cne 'DELETE ALL CLIENTS') {
    Write-Host 'Cancelled; nothing was deleted.' -ForegroundColor Green
    exit 1
}

# Avoid passing SQL through nested PowerShell -> Docker -> sh quoting. In Windows
# PowerShell, that can strip the quotes around -c's SQL and send only "DELETE"
# to psql. Copy the reviewed SQL file into the running database container and
# execute it with psql -f instead.
$sqlSource = Join-Path $PSScriptRoot 'clear-all-clients.sql'
$sqlInContainer = '/tmp/landapter-clear-all-clients.sql'
if (-not (Test-Path -LiteralPath $sqlSource -PathType Leaf)) {
    Write-Error "SQL file not found: $sqlSource"
    exit 1
}

$dbUser = ((& docker compose exec -T postgres printenv POSTGRES_USER) | Out-String).Trim()
$commandExitCode = $LASTEXITCODE
if ($commandExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($dbUser)) {
    Write-Error 'Could not read POSTGRES_USER from the running postgres container. Is Docker Compose running?'
    exit 1
}
$dbName = ((& docker compose exec -T postgres printenv POSTGRES_DB) | Out-String).Trim()
$commandExitCode = $LASTEXITCODE
if ($commandExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($dbName)) {
    Write-Error 'Could not read POSTGRES_DB from the running postgres container. Is Docker Compose running?'
    exit 1
}

& docker compose cp $sqlSource "postgres:$sqlInContainer"
if ($LASTEXITCODE -ne 0) {
    Write-Error 'Could not copy the SQL cleanup script into the postgres container.'
    exit 1
}

$psqlExitCode = 1
try {
    # Pass each psql argument separately: no shell string and no embedded SQL
    # means PowerShell cannot split "DELETE FROM clients" into extra arguments.
    & docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U $dbUser -d $dbName -f $sqlInContainer
    $psqlExitCode = $LASTEXITCODE
}
finally {
    & docker compose exec -T postgres rm -f $sqlInContainer | Out-Null
}

if ($psqlExitCode -ne 0) {
    exit $psqlExitCode
}
Write-Host 'All client records were deleted successfully.' -ForegroundColor Green
