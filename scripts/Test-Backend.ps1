[CmdletBinding()]
param([switch]$UseLocalDatabase)

$ErrorActionPreference = 'Stop'
$backendPath = Join-Path (Split-Path $PSScriptRoot -Parent) 'backend'
$previousTestDatabase = $env:TEST_DATABASE_URL

try {
    if ($UseLocalDatabase) {
        $envPath = Join-Path $backendPath '.env'
        if (-not (Test-Path -LiteralPath $envPath)) {
            throw 'backend/.env is missing. Configure the local database first.'
        }
        $settings = @{}
        foreach ($line in [IO.File]::ReadAllLines($envPath)) {
            if ($line -match '^\s*([A-Z_]+)\s*=(.*)$') {
                $settings[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
            }
        }
        if ($settings['APP_ENV'] -notin @('development', 'test')) {
            throw '-UseLocalDatabase requires APP_ENV=development or test in backend/.env.'
        }
        if (-not $settings['DATABASE_URL']) {
            throw 'DATABASE_URL is missing from backend/.env.'
        }
        $env:TEST_DATABASE_URL = $settings['DATABASE_URL']
    }
    if (-not $env:TEST_DATABASE_URL) {
        throw 'Set TEST_DATABASE_URL or pass -UseLocalDatabase. Integration tests must not be silently skipped.'
    }
    Push-Location $backendPath
    try {
        go test ./cmd/... ./db/... ./internal/... -count=1
        if ($LASTEXITCODE -ne 0) { throw 'Backend tests failed.' }
        go vet ./cmd/... ./db/... ./internal/...
        if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
        Write-Host 'PASS backend tests (database integration enabled) and go vet.'
    } finally {
        Pop-Location
    }
} finally {
    $env:TEST_DATABASE_URL = $previousTestDatabase
}
