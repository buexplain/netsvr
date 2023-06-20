@ECHO off

chcp 65001

call ./netsvr-stop.bat

SET netsvr=./../netsvr-windows-amd64.bin -config ./../configs/netsvr.toml

start /i /min /WAIT /B %netsvr%