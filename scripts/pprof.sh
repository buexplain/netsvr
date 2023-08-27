#!/bin/bash
# 这个脚本只生成Linux下的amd64的cpu.pprof文件
# 初始化编译输出目录
base_dir=$(dirname "$(pwd)")"/build/linux"
# 初始化打开文件数据量
ulimit -n 100000
# 清理函数
clearAll() {
  if ps aux | grep '[n]etsvr-linux-amd64.bin'; then
    # shellcheck disable=SC2046
    kill -15 $(ps aux | grep '[n]etsvr-linux-amd64.bin' | awk '{print $2}')
  fi
  if ps aux | grep '[s]tress-linux-amd64.bin'; then
    # shellcheck disable=SC2046
    kill -15 $(ps aux | grep '[s]tress-linux-amd64.bin' | awk '{print $2}')
  fi
  if ps aux | grep '[b]usiness-linux-amd64.bin'; then
    # shellcheck disable=SC2046
    kill -15 $(ps aux | grep '[b]usiness-linux-amd64.bin' | awk '{print $2}')
  fi
  rm -fr ./nohup.out
  sleep 3
  rm -fr ./log
  rm -fr /tmp/stress.log
}
# 删除旧的文件
rm -fr ./linux/amd64-cpu.pprof
# 先杀进程
clearAll
# 启动网关进程
nohup "$base_dir/netsvr-linux-amd64.bin" -config "$base_dir/configs/netsvr.toml" 1>/dev/null 2>/dev/null &
# 启动business进程
sleep 2
nohup "$base_dir/business-linux-amd64.bin" -config "$base_dir/configs/business.toml" 1>/dev/null 2>/dev/null &
# 启动压测进进程
sleep 2
nohup "$base_dir/stress-linux-amd64.bin" -config "$base_dir/configs/stress.toml" 1>/tmp/stress.log 2>/dev/null &
# 最后监控日志，当所有压测的连接建立完毕后，再采集cpu数据
stress_log=""
while true; do
  sleep 3
  current_stress_log=$(tail -n 1 /tmp/stress.log)
  if [[ "$stress_log" != "$current_stress_log" ]]; then
      stress_log=$current_stress_log
      echo "$stress_log"
  fi
  if [[ "$stress_log" == *"established"* ]]; then
      sleep 6
      if [ ! -d ./linux ]; then
        mkdir ./linux
      fi
      curl -o ./linux/amd64-cpu.pprof http://127.0.0.1:6062/debug/pprof/profile?seconds=60
      clearAll
      read -n 1 -s -r -p "按任意键继续..."
      exit 0
  fi
done