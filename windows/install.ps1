# Installs nothing system-wide. Checks Python and optionally builds the Zig engine.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

$py = Get-Command py -ErrorAction SilentlyContinue
if (-not $py) { $py = Get-Command python -ErrorAction SilentlyContinue }
if (-not $py) {
  Write-Error "Python 3 not found. Install from https://www.python.org/downloads/ (check 'Add python.exe to PATH')."
}
Write-Host "Python:" (& $py.Source --version)
Write-Host "Self-test..."
& $py.Source -m app.selftest
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Arcis Windows host is ready. Launch with windows\Arcis.bat"
