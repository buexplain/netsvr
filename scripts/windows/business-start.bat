@ECHO off

chcp 65001

call ./business-stop.bat

SET business=./../business-windows-amd64.bin -config ./../configs/business.toml

for /l %%i in (1,1,1) do (
   timeout /t 1
   start /i /min %business%
)
