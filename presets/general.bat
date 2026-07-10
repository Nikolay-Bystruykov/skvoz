@echo off
rem Skvoz - unblock YouTube AND Discord together. Double-click to run.
net session >nul 2>&1 || (powershell -Command "Start-Process -FilePath '%~f0' -Verb RunAs" & exit /b)
cd /d "%~dp0"
skvoz.exe --strategy fakedsplit --lists lists\list-youtube.txt,lists\list-discord.txt
pause
