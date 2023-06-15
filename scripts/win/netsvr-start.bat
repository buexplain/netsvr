@ECHO off

chcp 65001

call ./netsvr-stop.bat

SET netsvr=./../netsvr-win-amd64.exe -config ./../configs/netsvr.toml

start /i /min /WAIT /B %netsvr%