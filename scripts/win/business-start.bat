@ECHO off

chcp 65001

call ./business-stop.bat

SET business=./../business-win-amd64.exe -config ./../configs/business.toml

for /l %%i in (1,1,4) do (
   timeout /t 1
   start /i /min %business%
)