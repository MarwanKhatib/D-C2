# Install a built client on Windows and register a logon task.
# Run in an elevated PowerShell on a host you own.
#
# Usage:
#   .\scripts\install-windows.ps1 -Binary dist\dc2-windows-amd64.exe -EnvFile .env
param(
    [Parameter(Mandatory = $true)][string]$Binary,
    [string]$EnvFile = ".env",
    [string]$InstallDir = "$env:ProgramData\D-C2",
    [string]$TaskName = "D-C2",
    [switch]$NoTask
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $Binary)) { throw "binary not found: $Binary" }
if (-not (Test-Path $EnvFile)) { throw "env file not found: $EnvFile" }

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item -Force $Binary (Join-Path $InstallDir "dc2-client.exe")
Copy-Item -Force $EnvFile (Join-Path $InstallDir ".env")

# Restrict the config to Administrators and SYSTEM.
icacls (Join-Path $InstallDir ".env") /inheritance:r /grant:r "Administrators:F" "SYSTEM:F" | Out-Null

$exe = Join-Path $InstallDir "dc2-client.exe"

if (-not $NoTask) {
    $action = New-ScheduledTaskAction -Execute $exe -WorkingDirectory $InstallDir
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    $principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -RunLevel Highest
    Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Force | Out-Null
    Write-Host "registered scheduled task: $TaskName"
}

Write-Host "installed to $InstallDir"
Write-Host "run now: $exe"
