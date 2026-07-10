@echo off
rem Skvoz - stop and remove the Windows service.
net session >nul 2>&1 || (powershell -Command "Start-Process -FilePath '%~f0' -Verb RunAs" & exit /b)
cd /d "%~dp0"
skvoz.exe --service uninstall
echo.
echo Skvoz service removed.
pause
