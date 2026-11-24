for ($i = 1; $i -le 15; $i++) {
    try {
        $res = Invoke-WebRequest -Uri "http://localhost:8080/" -Headers @{"X-Forwarded-For"="189.40.12.5"} -UseBasicParsing -TimeoutSec 1
        Write-Host "Req $i Status: $($res.StatusCode)"
    } catch {
        Write-Host "Req $i Status: 429 Blocked"
    }
}

for ($i = 1; $i -le 8; $i++) {
    try {
        $null = Invoke-WebRequest -Uri "http://localhost:8080/" -Headers @{"X-Forwarded-For"="54.210.12.8"} -UseBasicParsing -TimeoutSec 1
    } catch {}
}

for ($i = 1; $i -le 5; $i++) {
    try {
        $null = Invoke-WebRequest -Uri "http://localhost:8080/" -Headers @{"X-Forwarded-For"="3.120.45.1"} -UseBasicParsing -TimeoutSec 1
    } catch {}
}
