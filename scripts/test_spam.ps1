$ErrorActionPreference = "Continue"
Set-Location "c:\GOOGLE DRIVE\load balancer"

# Ensure mock backends are running
$b = Get-Process -Name backend -ErrorAction SilentlyContinue
if (-not $b) {
    Write-Host "Starting backend instances..."
    Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8081 -tcp-port=9081 -name=backend-1"
    Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8082 -tcp-port=9082 -name=backend-2"
    Start-Process -FilePath "bin/backend.exe" -ArgumentList "-http-port=8083 -tcp-port=9083 -name=backend-3"
}

# Ensure proxy is running
$p = Get-Process -Name proxy -ErrorAction SilentlyContinue
if (-not $p) {
    Write-Host "Starting proxy..."
    Start-Process -FilePath "bin/proxy.exe" -ArgumentList "-config=config/config.yaml"
    Start-Sleep -Seconds 1
}

Write-Host "Simulating 20 rapid requests to test Rate Limiting (Limit: 5 req/s, Burst: 10)..."
$successCount = 0
$blockedCount = 0

for ($i = 1; $i -le 20; $i++) {
    try {
        $resp = Invoke-WebRequest -Uri "http://127.0.0.1:8080/" -UseBasicParsing -TimeoutSec 2
        Write-Host "Request $i : HTTP $($resp.StatusCode) (OK)" -ForegroundColor Green
        $successCount++
    } catch {
        if ($_.Exception.Response) {
            $code = [int]$_.Exception.Response.StatusCode
            Write-Host "Request $i : HTTP $code (BLOCKED by Rate Limiter!)" -ForegroundColor Red
            if ($code -eq 429) {
                $blockedCount++
            }
        } else {
            Write-Host "Request $i : Error $($_.Exception.Message)"
        }
    }
}

Write-Host "`nSummary: Allowed=$successCount, Blocked(429)=$blockedCount"
Write-Host "Inspecting Redis keys:"
& "C:\Program Files\Docker\Docker\resources\bin\docker.exe" exec redis redis-cli keys "ratelimit:*"
