$BaseUrl = "http://localhost:8080"

function Test-Health {
    Write-Host "`n[1] Testing Health Check..." -ForegroundColor Cyan
    Invoke-RestMethod -Uri "$BaseUrl/health" -Method Get
}

function Test-OTP-Request {
    param($phone)
    Write-Host "`n[2] Requesting OTP for $phone..." -ForegroundColor Cyan
    $body = @{ phone = $phone } | ConvertTo-Json
    Invoke-RestMethod -Uri "$BaseUrl/v1/auth/otp/request" -Method Post -Body $body -ContentType "application/json"
}

function Test-Route-Search {
    param($query)
    Write-Host "`n[3] Searching Routes for '$query'..." -ForegroundColor Cyan
    Invoke-RestMethod -Uri "$BaseUrl/v1/routes/search?query=$query" -Method Get
}

# Example Usage Sequence
Write-Host "--- Shukisha API Test Suite ---" -ForegroundColor Yellow
Test-Health
Test-OTP-Request -phone "254700000000"
Test-Route-Search -query "Nairobi"

Write-Host "`nNote: Use the 'verify' endpoint with the hashed OTP from the DB to get a JWT token for protected routes." -ForegroundColor Gray
