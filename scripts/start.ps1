$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath (Split-Path -Parent $PSScriptRoot)
if (-not (Test-Path -LiteralPath 'data\ml-1m\ratings.dat')) {
    Write-Host 'Read docs/DATA.md for the separate MovieLens research terms.'
    & go run ./cmd/fetch-data
    if ($LASTEXITCODE -ne 0) { throw 'Data download failed. See README for the native HTTPS download option.' }
}
& go run ./cmd/commonreel
if ($LASTEXITCODE -ne 0) { throw 'CommonReel stopped with an error.' }
