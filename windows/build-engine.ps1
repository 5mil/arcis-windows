# Clone and build https://github.com/5mil/arcis into engine\arcis.exe
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$src = if ($env:ARCIS_SRC) { $env:ARCIS_SRC } else { Join-Path $root "third_party\arcis" }
$engine = Join-Path $root "engine"
New-Item -ItemType Directory -Force -Path $engine | Out-Null

if (-not (Test-Path (Join-Path $src "build.zig"))) {
  if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Error "git is required to clone 5mil/arcis, or set ARCIS_SRC to a local checkout."
  }
  git clone --depth 1 https://github.com/5mil/arcis.git $src
}
if (-not (Get-Command zig -ErrorAction SilentlyContinue)) {
  Write-Error "Zig 0.14+ is required. https://ziglang.org/download/"
}
Push-Location $src
zig build -Doptimize=ReleaseSafe -Dtarget=x86_64-windows-gnu
$built = Join-Path $src "zig-out\bin\arcis.exe"
if (-not (Test-Path $built)) { Write-Error "Build finished but arcis.exe was not produced." }
Copy-Item $built (Join-Path $engine "arcis.exe") -Force
Pop-Location
Write-Host "Installed $(Join-Path $engine 'arcis.exe')"
