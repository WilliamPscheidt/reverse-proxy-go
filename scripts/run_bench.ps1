# scripts/run_bench.ps1
$ErrorActionPreference = "Stop"

Set-Location "c:\GOOGLE DRIVE\load balancer"

Write-Host ">>> Compiling binaries..." -ForegroundColor Cyan
go build -o bin/proxy.exe ./cmd/proxy
go build -o bin/backend.exe ./cmd/backend
go build -o bin/bench.exe ./cmd/bench

Write-Host ">>> Starting 3 Upstream Backend instances..." -ForegroundColor Cyan
$b1 = Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8081 -tcp-port=9081 -name=backend-1" -PassThru
$b2 = Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8082 -tcp-port=9082 -name=backend-2" -PassThru
$b3 = Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8083 -tcp-port=9083 -name=backend-3" -PassThru

Write-Host ">>> Starting Reverse Proxy & Load Balancer..." -ForegroundColor Cyan
$proxy = Start-Process -FilePath "bin/proxy.exe" -ArgumentList "-config=config/config.yaml" -PassThru

Start-Sleep -Seconds 2

try {
    Write-Host ">>> Verifying L7 HTTP routing..." -ForegroundColor Green
    $r1 = Invoke-RestMethod -Uri "http://127.0.0.1:8080/" -Method Get
    $r2 = Invoke-RestMethod -Uri "http://127.0.0.1:8080/" -Method Get
    $r3 = Invoke-RestMethod -Uri "http://127.0.0.1:8080/" -Method Get
    Write-Host "Request 1 routed to: $($r1.server)"
    Write-Host "Request 2 routed to: $($r2.server)"
    Write-Host "Request 3 routed to: $($r3.server)"

    Write-Host "`n>>> Verifying Prometheus /metrics endpoint..." -ForegroundColor Green
    $metrics = Invoke-WebRequest -Uri "http://127.0.0.1:2112/metrics" -UseBasicParsing
    Write-Host "Metrics HTTP Status: $($metrics.StatusCode)"
    Write-Host "Sample metrics output:"
    ($metrics.Content -split "`n" | Select-String "loadbalancer_http" | Select-Object -First 6) | ForEach-Object { Write-Host "  $_" }

    Write-Host "`n>>> Running High-Concurrency Stress Benchmark (/ping)..." -ForegroundColor Yellow
    & "bin/bench.exe" -url "http://127.0.0.1:8080/ping" -c 120 -d 5s -warmup 1s

} finally {
    Write-Host "`n>>> Cleaning up processes..." -ForegroundColor Cyan
    Stop-Process -Id $proxy.Id -Force -ErrorAction SilentlyContinue
    Stop-Process -Id $b1.Id -Force -ErrorAction SilentlyContinue
    Stop-Process -Id $b2.Id -Force -ErrorAction SilentlyContinue
    Stop-Process -Id $b3.Id -Force -ErrorAction SilentlyContinue
    Write-Host ">>> Finished cleanly." -ForegroundColor Green
}
