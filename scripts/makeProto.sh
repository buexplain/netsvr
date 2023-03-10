#!/bin/bash
mkdir -p ./../pkg/protocol
rm -rf ./../pkg/protocol/*

# https://protobuf.dev/reference/go/go-generated/

# shellcheck disable=SC2046
root_dir="$(dirname $(pwd))"
proto_path="$root_dir/api"
match="$proto_path/*.proto"
for file in $match; do
  proto="$(basename "$file")"
  echo "protoc --go_out=$root_dir --proto_path=$proto_path $proto"
  protoc --go_out="$root_dir" --proto_path="$proto_path" "$proto"
  # shellcheck disable=SC2181
  if [ $? != 0 ]; then
    # shellcheck disable=SC2162
    read
    exit $?
  fi
done
