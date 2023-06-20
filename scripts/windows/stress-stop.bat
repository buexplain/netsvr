@ECHO off

tasklist /nh|find /i "stress-windows-amd64.bin" > nul

IF ERRORLEVEL 0 (
	taskkill /f /t /im "stress-windows-amd64.bin"
)