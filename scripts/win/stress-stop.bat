@ECHO off

tasklist /nh|find /i "stress-win-amd64.exe" > nul

IF ERRORLEVEL 0 (
	taskkill /f /t /im "stress-win-amd64.exe"
)