#!/bin/bash
mkdir -p ./../internal/protocol
rm -rf ./../internal/protocol/*

for file in ./toWorker/*.proto; do
    protoc --go_out=./../internal "$file"
done;

for file in ./toServer/*.proto; do
    protoc --go_out=./../internal "$file"
done;