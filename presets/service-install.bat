@echo off
rem Skvoz - install as a Windows service (auto-start on boot) for YouTube + Discord.
net session >nul 2>&1 || (powershell -Command "Start-Process -FilePath '%~f0' -Verb RunAs" & exit /b)
cd /d "%~dp0"
skvoz.exe --service install --strategy fakedsplit --lists lists\list-youtube.txt,lists\list-discord.txt
echo.
echo Skvoz service installed and started. It will run automatically on boot.
pause
