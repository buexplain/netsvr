@ECHO off

tasklist /nh|find /i "netsvr-windows-amd64.bin" > nul

IF ERRORLEVEL 0 (
	taskkill /f /t /im "netsvr-windows-amd64.bin"
)