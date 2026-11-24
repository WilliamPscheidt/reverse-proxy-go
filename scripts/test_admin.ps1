$ErrorActionPreference = "Stop"
Set-Location "c:\GOOGLE DRIVE\load balancer"

$proc = Start-Process -FilePath "bin/proxy.exe" -ArgumentList "-config=config/config.yaml" -PassThru
Start-Sleep -Seconds 1

try {
    $html = Invoke-WebRequest -Uri "http://127.0.0.1:8090/" -UseBasicParsing
    Write-Host "Admin Dashboard HTTP Status:" $html.StatusCode

    $status = Invoke-RestMethod -Uri "http://127.0.0.1:8090/api/status"
    Write-Host "Proxy Mode:" $status.mode
    Write-Host "L7 Algorithm:" $status.l7.algorithm
    Write-Host "L7 Healthy Nodes:" $status.l7.healthy_nodes "/" $status.l7.total_nodes
    Write-Host "L4 Healthy Nodes:" $status.l4.healthy_nodes "/" $status.l4.total_nodes

    $backends = Invoke-RestMethod -Uri "http://127.0.0.1:8090/api/backends"
    Write-Host "Total Configured Backends:" $backends.Count

    Write-Host "`n>>> Dynamically adding new backend http://127.0.0.1:8089 via Admin API..."
    $body = @{ type = "l7"; target = "http://127.0.0.1:8089"; weight = 2 } | ConvertTo-Json
    $add = Invoke-RestMethod -Uri "http://127.0.0.1:8090/api/backends" -Method Post -Body $body -ContentType "application/json"
    Write-Host "Added successfully:" $add.backend

    $backendsAfter = Invoke-RestMethod -Uri "http://127.0.0.1:8090/api/backends"
    Write-Host "Total Backends after addition:" $backendsAfter.Count

    Write-Host "`n>>> Dynamically switching algorithm to least_conn via Admin API..."
    $algoBody = @{ type = "both"; algorithm = "least_conn" } | ConvertTo-Json
    $algoResp = Invoke-RestMethod -Uri "http://127.0.0.1:8090/api/algorithm" -Method Post -Body $algoBody -ContentType "application/json"
    Write-Host "Algorithm updated to:" $algoResp.algorithm

    Write-Host "`nAdmin Dashboard and API verified successfully!" -ForegroundColor Green
} finally {
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
