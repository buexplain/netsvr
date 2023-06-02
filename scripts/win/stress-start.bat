@ECHO off

chcp 65001

call ./stress-stop.bat

SET stress=./../stress-win-amd64.exe -config ./../configs/stress.toml

start /i /min %stress%