@ECHO off

chcp 65001

call ./stress-stop.bat

SET stress=./../stress-windows-amd64.bin -config ./../configs/stress.toml

start /i /min /WAIT /B %stress%

pause