# Build the contained Windows desktop binary.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  Write-Error "Go is required to build arcis.exe. https://go.dev/dl/"
}
go test ./internal/host/
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
New-Item -ItemType Directory -Force -Path dist | Out-Null
go build -ldflags "-s -w" -o dist\arcis.exe ./cmd/arcis
Write-Host "Built dist\arcis.exe — desktop window and local server are inside this file."
