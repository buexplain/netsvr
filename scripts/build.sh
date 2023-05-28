#!/bin/bash

# 清空编译缓存
go clean -cache

# 编译Windows版本
echo build Windows version
mkdir -p ./../build/win
rm -rf ./../build/win/*
mkdir -p ./../build/win/configs
export GOARCH=amd64
export GOOS=windows
go build -trimpath -ldflags "-s -w" -o ./../build/win/netsvr-win-amd64.exe ./../cmd/netsvr.go
go build -trimpath -ldflags "-s -w" -o ./../build/win/business-win-amd64.exe ./../test/business/cmd/business.go
go build -trimpath -ldflags "-s -w" -o ./../build/win/stress-win-amd64.exe ./../test/stress/cmd/stress.go
cp ./../configs/netsvr.example.toml ./../build/win/configs/netsvr.toml
cp ./../test/business/configs/business.example.toml ./../build/win/configs/business.toml
cp ./../test/stress/configs/stress.example.toml ./../build/win/configs/stress.toml

# 编译Linux版本
echo build Linux version
mkdir -p ./../build/linux
rm -rf ./../build/linux/*
mkdir -p ./../build/linux/configs
export GOARCH=amd64
export GOOS=linux
go build -trimpath -ldflags "-s -w" -o ./../build/linux/netsvr-linux-amd64.bin ./../cmd/netsvr.go
go build -trimpath -ldflags "-s -w" -o ./../build/linux/business-linux-amd64.bin ./../test/business/cmd/business.go
go build -trimpath -ldflags "-s -w" -o ./../build/linux/stress-linux-amd64.bin ./../test/stress/cmd/stress.go
cp ./../configs/netsvr.example.toml ./../build/linux/configs/netsvr.toml
cp ./../test/business/configs/business.example.toml ./../build/linux/configs/business.toml
cp ./../test/stress/configs/stress.example.toml ./../build/linux/configs/stress.toml

# 编译Darwin版本
echo build Darwin version
mkdir -p ./../build/darwin
rm -rf ./../build/darwin/*
mkdir -p ./../build/darwin/configs
export GOARCH=amd64
export GOOS=darwin
go build -trimpath -ldflags "-s -w" -o ./../build/darwin/netsvr-darwin-amd64.bin ./../cmd/netsvr.go
go build -trimpath -ldflags "-s -w" -o ./../build/darwin/business-darwin-amd64.bin ./../test/business/cmd/business.go
go build -trimpath -ldflags "-s -w" -o ./../build/darwin/stress-darwin-amd64.bin ./../test/stress/cmd/stress.go
cp ./../configs/netsvr.example.toml ./../build/darwin/configs/netsvr.toml
cp ./../test/business/configs/business.example.toml ./../build/darwin/configs/business.toml
cp ./../test/stress/configs/stress.example.toml ./../build/darwin/configs/stress.toml

echo build successfully