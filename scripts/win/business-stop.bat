@ECHO off

tasklist /nh|find /i "business-win-amd64.exe" > nul

IF ERRORLEVEL 0 (
	taskkill /f /t /im "business-win-amd64.exe"
)