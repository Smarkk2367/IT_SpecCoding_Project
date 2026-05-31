# 1. Login
$loginBody = @{
    email = "marketer@example.com"
    password = "password"
} | ConvertTo-Json

Write-Host "Logging in..."
$loginRes = Invoke-RestMethod -Uri "http://localhost:3000/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
$token = $loginRes.token
Write-Host "Token received."

$headers = @{
    Authorization = "Bearer $token"
}

# 2. Create client
$clientBody = @{
    name = "Acme Corp $((Get-Date).Ticks)"
} | ConvertTo-Json

Write-Host "Creating client..."
$clientRes = Invoke-RestMethod -Uri "http://localhost:3000/api/clients" -Method Post -Body $clientBody -ContentType "application/json" -Headers $headers
$clientId = $clientRes.id
Write-Host "Client created. ID: $clientId"

# 3. Create link
$linkBody = @{
    original_url = "https://google.com"
    campaign_name = "Autumn Sale"
    client_id = $clientId
} | ConvertTo-Json

Write-Host "Creating link..."
$linkRes = Invoke-RestMethod -Uri "http://localhost:3000/api/links" -Method Post -Body $linkBody -ContentType "application/json" -Headers $headers
$linkId = $linkRes.id
$shortCode = $linkRes.short_code
Write-Host "Link created. Short code: $shortCode"

# 4. Simulate clicks
Write-Host "Simulating clicks..."

# Click 1: PL Desktop Google
$headers1 = @{
    "X-Forwarded-For" = "89.64.0.1"
    "Referer" = "https://google.com"
}
Write-Host "Simulating click 1 (PL, Desktop, Google)..."
$res1 = Invoke-WebRequest -Uri "http://localhost:3000/$shortCode" -Headers $headers1 -UserAgent "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" -MaximumRedirection 0 -ErrorAction SilentlyContinue
Write-Host "Click 1 Status: $($res1.StatusCode)"

# Click 2: PL Mobile Facebook
$headers2 = @{
    "X-Forwarded-For" = "89.64.0.2"
    "Referer" = "https://facebook.com"
}
Write-Host "Simulating click 2 (PL, Mobile, Facebook)..."
$res2 = Invoke-WebRequest -Uri "http://localhost:3000/$shortCode" -Headers $headers2 -UserAgent "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36" -MaximumRedirection 0 -ErrorAction SilentlyContinue
Write-Host "Click 2 Status: $($res2.StatusCode)"

# Click 3: US Desktop Direct
$headers3 = @{
    "X-Forwarded-For" = "8.8.8.8"
}
Write-Host "Simulating click 3 (US, Desktop, Direct)..."
$res3 = Invoke-WebRequest -Uri "http://localhost:3000/$shortCode" -Headers $headers3 -UserAgent "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" -MaximumRedirection 0 -ErrorAction SilentlyContinue
Write-Host "Click 3 Status: $($res3.StatusCode)"

# Wait for worker processing
Write-Host "Waiting 4 seconds for click consumer processing..."
Start-Sleep -Seconds 4

# 5. Request report
$reportBody = @{
    date_from = "2026-05-29T00:00:00Z"
    date_to = "2026-06-02T00:00:00Z"
    client_id = $clientId
} | ConvertTo-Json

Write-Host "Requesting report..."
$reportRes = Invoke-RestMethod -Uri "http://localhost:3000/api/reports" -Method Post -Body $reportBody -ContentType "application/json" -Headers $headers
$reportId = $reportRes.report_id
Write-Host "Report requested. ID: $reportId, Status: $($reportRes.status)"

# 6. Poll report status
Write-Host "Polling report status..."
$maxAttempts = 20
$attempt = 1
$status = "pending"

while ($attempt -le $maxAttempts) {
    $statusRes = Invoke-RestMethod -Uri "http://localhost:3000/api/reports/$reportId" -Method Get -Headers $headers
    $status = $statusRes.status
    Write-Host "Attempt $($attempt): Status = $status"
    if ($status -eq "done" -or $status -eq "failed") {
        break
    }
    Start-Sleep -Seconds 2
    $attempt++
}

if ($status -ne "done") {
    if ($statusRes.error_message) {
        Write-Error "Report generation failed with error: $($statusRes.error_message)"
    } else {
        Write-Error "Report generation timed out or failed. Current status: $status"
    }
    exit 1
}

# 7. Download PDF
Write-Host "Downloading PDF..."
$pdfPath = "scratch/report_$reportId.pdf"
Invoke-WebRequest -Uri "http://localhost:3000/api/reports/$reportId/download" -OutFile $pdfPath -Headers $headers
Write-Host "PDF downloaded to: $pdfPath"

$fileInfo = Get-Item $pdfPath
Write-Host "File Size: $($fileInfo.Length) bytes"
if ($fileInfo.Length -gt 1000) {
    Write-Host "SUCCESS: Report PDF generated and downloaded successfully!"
} else {
    Write-Error "ERROR: Downloaded file is too small (likely not a valid PDF)."
}
