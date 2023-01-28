#!/bin/bash
mkdir -p ./../internal/protocol
rm -rf ./../internal/protocol/*

# https://developers.google.com/protocol-buffers/docs/reference/go-generated

for file in ./reCtx/*.proto; do
  echo "$file"
  protoc --go_out=./../../ --proto_path=./ "$file"
done

for file in ./toWorker/*.proto; do
  echo "$file"
  protoc --go_out=./../../ --proto_path=./ "$file"
done

for file in ./toServer/*.proto; do
  echo "$file"
  protoc --go_out=./../../ --proto_path=./ "$file"
done