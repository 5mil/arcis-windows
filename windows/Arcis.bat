@echo off
setlocal
cd /d "%~dp0.."
where py >nul 2>&1 && (
  py -3 -m app.desktop
  goto :eof
)
where python >nul 2>&1 && (
  python -m app.desktop
  goto :eof
)
echo Python 3 is required. Install it from https://www.python.org/downloads/ and re-run.
exit /b 1
