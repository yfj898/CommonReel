param([string]$LanIP = '', [string]$Listen = '127.0.0.1:8093')
$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $taskRoot
if (-not (Test-Path -LiteralPath '.env.local')) {
    Write-Host 'No .env.local: forms work, but subscription supply stays unknown until a free Watchmode key is configured.'
}
$taskArgs = @('run', './cmd/commonreel-v2', '-listen', $Listen)
if ($LanIP) { $taskArgs += @('-lan', $LanIP) }
$goCommand = Get-Command go -ErrorAction SilentlyContinue
if ($goCommand) { & $goCommand.Source @taskArgs }
elseif (Test-Path -LiteralPath 'C:\Program Files\Go\bin\go.exe') { & 'C:\Program Files\Go\bin\go.exe' @taskArgs }
else { throw 'Go is not installed or not on PATH.' }
exit $LASTEXITCODE
