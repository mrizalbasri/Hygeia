[CmdletBinding()]
param(
    [ValidatePattern('^[a-z0-9][a-z0-9_.-]*$')]
    [string]$DockerHubUser,
    [ValidatePattern('^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$')]
    [string]$Tag = 'dev'
)

$ErrorActionPreference = 'Stop'
$backendPath = Join-Path (Split-Path $PSScriptRoot -Parent) 'backend'
$image = if ($DockerHubUser) { "${DockerHubUser}/hygeia-api:${Tag}" } else { "hygeia-api:${Tag}" }
docker info --format '{{.OSType}}' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Start the Docker Linux engine before building.' }
docker build --platform linux/amd64 --tag $image $backendPath
if ($LASTEXITCODE -ne 0) { throw 'Backend image build failed.' }
Write-Output "Built $image. Nothing was pushed to Docker Hub."
