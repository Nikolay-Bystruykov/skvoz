@echo off
rem Skvoz - unblock YouTube. Double-click to run (asks for administrator).
net session >nul 2>&1 || (powershell -Command "Start-Process -FilePath '%~f0' -Verb RunAs" & exit /b)
cd /d "%~dp0"
skvoz.exe --strategy fakedsplit --lists lists\list-youtube.txt
pause
