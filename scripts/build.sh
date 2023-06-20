#!/bin/bash

# 设定目标平台
declare -a system=("windows" "linux" "darwin")
declare -a cpu=("amd64")
# 设定脚本目录
current_dir=$(pwd)
# 初始化编译输出目录
build_dir=$(dirname "$current_dir")"/build"
mkdir -p "$build_dir"
for i in "${!system[@]}"
do
  rm -rf "$build_dir/${system[$i]}"
done
# 清空编译缓存
go clean -cache
# 开始编译
for i in "${!system[@]}"
do
  for j in "${!cpu[@]}"
  do
    # 编译对应cpu与系统的文件
    echo "building ${system[$i]} ${cpu[$j]}"
    export "GOARCH=${cpu[$j]}"
    export "GOOS=${system[$i]}"
    if [ -f "$current_dir/${system[$i]}/${cpu[$j]}-cpu.pprof" ]; then
      go build -trimpath -pgo="$current_dir/${system[$i]}/${cpu[$j]}-cpu.pprof" -ldflags "-s -w" -o "$build_dir/${system[$i]}/netsvr-${system[$i]}-${cpu[$j]}.bin" "$current_dir/../cmd/netsvr.go"
    else
      go build -trimpath -ldflags "-s -w" -o "$build_dir/${system[$i]}/netsvr-${system[$i]}-${cpu[$j]}.bin" "$current_dir/../cmd/netsvr.go"
    fi
    go build -trimpath -ldflags "-s -w" -o "$build_dir/${system[$i]}/business-${system[$i]}-${cpu[$j]}.bin" "$current_dir/../test/business/cmd/business.go"
    go build -trimpath -ldflags "-s -w" -o "$build_dir/${system[$i]}/stress-${system[$i]}-${cpu[$j]}.bin" "$current_dir/../test/stress/cmd/stress.go"
    # 拷贝配置文件
    mkdir -p "$build_dir/${system[$i]}/configs/"
    cp "$current_dir/../configs/netsvr.example.toml" "$build_dir/${system[$i]}/configs/netsvr.toml"
    cp "$current_dir/../test/business/configs/business.example.toml" "$build_dir/${system[$i]}/configs/business.toml"
    cp "$current_dir/../test/stress/configs/stress.example.toml" "$build_dir/${system[$i]}/configs/stress.toml"
    # 拷贝附带的脚本文件
    if [[ -d "$current_dir/${system[$i]}" ]]; then
      scripts="$current_dir/${system[$i]}/*"
      mkdir -p "$build_dir/${system[$i]}/scripts/"
       for file in $scripts; do
         if [[ "$file" == *"linux"* ]]; then
           continue 1
         fi
         if [ -f "$file" ]; then
           cp "$file" "$build_dir/${system[$i]}/scripts/"
         fi
       done
    fi
    # 压缩编译后的文件
    if command -v upx >/dev/null 2>&1; then
      upx -9 "$build_dir/${system[$i]}/netsvr-${system[$i]}-${cpu[$j]}.bin"
      upx -9 "$build_dir/${system[$i]}/business-${system[$i]}-${cpu[$j]}.bin"
      upx -9 "$build_dir/${system[$i]}/stress-${system[$i]}-${cpu[$j]}.bin"
    fi
    echo "build ${system[$i]} ${cpu[$j]} successfully"
  done
done
read -n 1 -s -r -p "按任意键继续..."
exit 0