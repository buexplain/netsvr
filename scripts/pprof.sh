#!/bin/bash
# 这个脚本只生成Linux下的amd64的cpu.pprof文件
# 如果出现错误：坏的解释器: 没有那个文件或目录，则需要执行：sed -i 's/\r$//' pprof.sh
# 初始化目录地址
base_dir=$(dirname "$(pwd)")
build_dir=${base_dir}"/build/linux"
save_dir=${base_dir}"/scripts/linux"
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
# 进入编译目录
cd "$build_dir"
# 启动网关进程
nohup "$build_dir/netsvr-linux-amd64.bin" -config "$build_dir/configs/netsvr.toml" 1>/dev/null 2>/dev/null &
# 启动business进程
sleep 2
nohup "$build_dir/business-linux-amd64.bin" -config "$build_dir/configs/business.toml" 1>/dev/null 2>/dev/null &
# 启动压测进进程
sleep 2
nohup "$build_dir/stress-linux-amd64.bin" -config "$build_dir/configs/stress.toml" 1>/tmp/stress.log 2>/dev/null &
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
      sleep 30
      echo "开始采集cpu数据"
      if [ ! -d ${save_dir} ]; then
        mkdir ${save_dir}
      fi
      curl -o ${save_dir}/amd64-cpu.pprof http://127.0.0.1:6062/debug/pprof/profile?seconds=60
      clearAll
      read -n 1 -s -r -p "按任意键继续..."
      exit 0
  fi
done