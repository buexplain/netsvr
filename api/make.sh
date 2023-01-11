#!/bin/bash
mkdir -p ./../internal/protocol
rm -rf ./../internal/protocol/*

# https://developers.google.com/protocol-buffers/docs/reference/go-generated
for file in ./toWorker/*.proto; do
    protoc --go_out=./../internal "$file"
done;

for file in ./toServer/*.proto; do
    protoc --go_out=./../internal "$file"
done;