[CmdletBinding()]
param(
    [ValidateSet('Api', 'Migrate', 'Version')]
    [string]$Action = 'Version'
)

$ErrorActionPreference = 'Stop'
$backendPath = Join-Path (Split-Path $PSScriptRoot -Parent) 'backend'
$envPath = Join-Path $backendPath '.env.staging'
if (-not (Test-Path -LiteralPath $envPath)) {
    throw 'Configure backend/.env.staging first; see docs/NEON-STAGING-2026-10-29.md.'
}
$settings = @{}
foreach ($line in [IO.File]::ReadAllLines($envPath)) {
    if ($line -match '^\s*([A-Z_]+)\s*=(.*)$') {
        $settings[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
    }
}
$connectionKey = if ($Action -eq 'Api') { 'DATABASE_URL' } else { 'DATABASE_URL_UNPOOLED' }
if (-not $settings[$connectionKey]) { throw "$connectionKey is required in .env.staging." }
$connection = [uri]$settings[$connectionKey]
if ($connection.Scheme -notin @('postgres', 'postgresql') -or
    $connection.Host -notlike '*.neon.tech' -or
    $connection.AbsolutePath -ne '/hygeia_staging' -or
    $connection.Query -notmatch '(^\?|&)sslmode=verify-full(&|$)') {
    throw 'Expected the hygeia_staging Neon database with sslmode=verify-full.'
}
if ($Action -ne 'Api' -and $connection.Host -like '*-pooler.*') {
    throw 'Migrations require a direct connection, without -pooler in the hostname.'
}
$settings['DATABASE_URL'] = $settings[$connectionKey]
$previous = @{}
try {
    foreach ($key in $settings.Keys) {
        $previous[$key] = [Environment]::GetEnvironmentVariable($key, 'Process')
        [Environment]::SetEnvironmentVariable($key, $settings[$key], 'Process')
    }
    Push-Location $backendPath
    try {
        switch ($Action) {
            'Api' { go run ./cmd/api }
            'Migrate' { go run ./cmd/migrate up }
            'Version' { go run ./cmd/migrate version }
        }
        if ($LASTEXITCODE -ne 0) { throw "Staging $Action failed." }
    } finally {
        Pop-Location
    }
} finally {
    foreach ($key in $previous.Keys) {
        [Environment]::SetEnvironmentVariable($key, $previous[$key], 'Process')
    }
}
