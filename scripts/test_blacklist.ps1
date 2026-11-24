# 1. Block IP
$body = @{
    ip = "189.40.12.5"
    reason = "Bloqueado pelo teste"
} | ConvertTo-Json
$res = Invoke-RestMethod -Uri "http://localhost:8090/api/blacklist" -Method Post -Body $body -ContentType "application/json"
Write-Host "Block result: $($res.status) on IP $($res.entry.ip)"

# 2. Check analytics
$snap = Invoke-RestMethod -Uri "http://localhost:8090/api/analytics"
Write-Host "Analytics Blacklisted status: $($snap.geo_locations[0].ips[0].blacklisted)"

# 3. Request with blocked IP (must receive 403 Forbidden)
try {
    $r = Invoke-WebRequest -Uri "http://localhost:8080/" -Headers @{"X-Forwarded-For"="189.40.12.5"} -UseBasicParsing
    Write-Host "Unexpected status: $($r.StatusCode)"
} catch {
    Write-Host "Blocked successfully with HTTP Status: $($_.Exception.Response.StatusCode.value__)"
}

# 4. Request with unblocked IP (must receive 200 OK)
$ok = Invoke-WebRequest -Uri "http://localhost:8080/" -Headers @{"X-Forwarded-For"="189.40.12.99"} -UseBasicParsing
Write-Host "Allowed IP Status: $($ok.StatusCode)"

# 5. Unblock IP
$unblockBody = @{ ip = "189.40.12.5" } | ConvertTo-Json
$del = Invoke-RestMethod -Uri "http://localhost:8090/api/blacklist" -Method Delete -Body $unblockBody -ContentType "application/json"
Write-Host "Unblock result: $($del.status)"

# 6. Request again with 189.40.12.5 (must receive 200 OK)
$r2 = Invoke-WebRequest -Uri "http://localhost:8080/" -Headers @{"X-Forwarded-For"="189.40.12.5"} -UseBasicParsing
Write-Host "Unblocked request status: $($r2.StatusCode)"
