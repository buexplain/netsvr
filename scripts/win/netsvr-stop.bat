@ECHO off

tasklist /nh|find /i "netsvr-win-amd64.exe" > nul

IF ERRORLEVEL 0 (
	taskkill /f /t /im "netsvr-win-amd64.exe"
)