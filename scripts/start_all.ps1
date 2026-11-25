$ErrorActionPreference = "Continue"
Set-Location "c:\GOOGLE DRIVE\load balancer"

Get-Process -Name proxy, backend -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Milliseconds 500

Write-Host "Starting backends..."
$wd = "c:\GOOGLE DRIVE\load balancer"
$b1 = Start-Process -FilePath "$wd\bin\backend.exe" -ArgumentList "-http-port=8081 -tcp-port=9081 -name=backend-1" -WorkingDirectory $wd -PassThru
$b2 = Start-Process -FilePath "$wd\bin\backend.exe" -ArgumentList "-http-port=8082 -tcp-port=9082 -name=backend-2" -WorkingDirectory $wd -PassThru
$b3 = Start-Process -FilePath "$wd\bin\backend.exe" -ArgumentList "-http-port=8083 -tcp-port=9083 -name=backend-3" -WorkingDirectory $wd -PassThru
$b4 = Start-Process -FilePath "$wd\bin\backend.exe" -ArgumentList "-http-port=8084 -tcp-port=9084 -name=backend-canary-v2" -WorkingDirectory $wd -PassThru
$b5 = Start-Process -FilePath "$wd\bin\backend.exe" -ArgumentList "-http-port=8085 -tcp-port=9085 -name=backend-site1" -WorkingDirectory $wd -PassThru

Start-Sleep -Seconds 1
Write-Host "Starting proxy..."
$p = Start-Process -FilePath "$wd\bin\proxy.exe" -ArgumentList "-config=config/config.yaml" -WorkingDirectory $wd -PassThru

Write-Host "Services running:"
Write-Host "  Proxy (PID: $($p.Id)) on ports 8080 (L7), 9000 (L4), 8090 (Admin), 2112 (Metrics)"
Write-Host "  Backends: 8081-8083 (Primary), 8084 (Canary v2), 8085 (Dedicated site1)"

Wait-Process -Id $p.Id
