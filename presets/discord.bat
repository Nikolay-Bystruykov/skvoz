@echo off
rem Skvoz - unblock Discord. Double-click to run (asks for administrator).
net session >nul 2>&1 || (powershell -Command "Start-Process -FilePath '%~f0' -Verb RunAs" & exit /b)
cd /d "%~dp0"
skvoz.exe --strategy fakedsplit --lists lists\list-discord.txt
pause
