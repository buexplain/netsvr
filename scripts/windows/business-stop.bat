@ECHO off

tasklist /nh|find /i "business-windows-amd64.bin" > nul

IF ERRORLEVEL 0 (
	taskkill /f /t /im "business-windows-amd64.bin"
)