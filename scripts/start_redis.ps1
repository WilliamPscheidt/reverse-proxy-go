$ErrorActionPreference = "Stop"

$dockerPath = "C:\Program Files\Docker\Docker\resources\bin"
if (Test-Path $dockerPath) {
    $env:PATH = "$dockerPath;" + $env:PATH
}

Write-Host "Checking existing Redis container..."
$existing = docker ps -a -q -f name=^/redis$
if ($existing) {
    Write-Host "Found existing container $existing. Starting it..."
    docker start redis
} else {
    Write-Host "Starting new Redis container..."
    docker run -d --name redis -p 6379:6379 --restart always redis:7-alpine
}

Start-Sleep -Seconds 2
docker ps -f name=redis
