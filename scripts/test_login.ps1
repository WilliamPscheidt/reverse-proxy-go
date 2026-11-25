Set-Location "c:\GOOGLE DRIVE\load balancer"

Write-Host "=== 1. Test GET / without auth ==="
$r1 = curl.exe -s -i http://localhost:8090/
Write-Host ($r1 | Select-Object -First 4 | Out-String)

Write-Host "=== 2. Test GET /login ==="
$r2 = curl.exe -s http://localhost:8090/login
Write-Host "Length:" $r2.Length
Write-Host "Found title:" ($r2.Contains("ProxyGO Enterprise Console"))

Write-Host "=== 3. Test POST /api/login (Bad Credentials) ==="
$r3 = curl.exe -s -i -X POST http://localhost:8090/api/login -H "Content-Type: application/json" -d '{\"username\":\"admin\",\"password\":\"errada\"}'
Write-Host ($r3 | Select-Object -First 6 | Out-String)

Write-Host "=== 4. Test POST /api/login (Valid Credentials) ==="
$r4 = curl.exe -s -i -c cookies.txt -X POST http://localhost:8090/api/login -H "Content-Type: application/json" -d '{\"username\":\"admin\",\"password\":\"admin123\"}'
Write-Host ($r4 | Out-String)

Write-Host "=== 5. Test GET / with Session Cookie ==="
$r5 = curl.exe -s -i -b cookies.txt http://localhost:8090/
Write-Host ($r5 | Select-Object -First 6 | Out-String)

Write-Host "=== 6. Test GET /api/me ==="
$r6 = curl.exe -s -b cookies.txt http://localhost:8090/api/me
Write-Host $r6

Write-Host "=== 7. Test POST /api/logout ==="
$r7 = curl.exe -s -i -b cookies.txt -c cookies.txt -X POST http://localhost:8090/api/logout -H "Content-Type: application/json" -d '{}'
Write-Host ($r7 | Out-String)

Write-Host "=== 8. Test GET / after Logout ==="
$r8 = curl.exe -s -i -b cookies.txt http://localhost:8090/
Write-Host ($r8 | Select-Object -First 4 | Out-String)

Remove-Item -Force cookies.txt -ErrorAction SilentlyContinue
