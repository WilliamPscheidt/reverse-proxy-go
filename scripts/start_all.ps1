$ErrorActionPreference = "Continue"
Set-Location "c:\GOOGLE DRIVE\load balancer"

Get-Process -Name proxy, backend -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Milliseconds 500

Write-Host "Starting backends..."
$b1 = Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8081 -tcp-port=9081 -name=backend-1" -PassThru
$b2 = Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8082 -tcp-port=9082 -name=backend-2" -PassThru
$b3 = Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8083 -tcp-port=9083 -name=backend-3" -PassThru

Start-Sleep -Seconds 1
Write-Host "Starting proxy..."
$p = Start-Process -FilePath "bin/proxy.exe" -ArgumentList "-config=config/config.yaml" -PassThru

Write-Host "Services running:"
Write-Host "  Proxy (PID: $($p.Id)) on ports 8080 (L7), 9000 (L4), 8090 (Admin), 2112 (Metrics)"
Write-Host "  Backends on ports 8081, 8082, 8083"

Wait-Process -Id $p.Id
