/**
* Copyright 2023 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

// 本文件负责被测环境本身：构建并拉起网关与业务进程、探测就绪、测试结束回收。
// 端口等常量必须与 configs/*.toml 一致。
package integration

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

// 被测环境的监听地址，必须与 configs/*.toml 保持一致
const (
	customerAddr = "127.0.0.1:6160" // 网关的 customer websocket 服务
	workerAddr   = "127.0.0.1:6161" // 网关的 worker 服务（业务进程注册到这里）
	taskAddr     = "127.0.0.1:6162" // 网关的 task 服务（业务进程从这里发请求）
	clientAddr   = "127.0.0.1:6164" // 业务进程托管的手工测试页
	wsPath       = "/netsvr"
)

// wsURL 客户连接网关的地址
const wsURL = "ws://" + customerAddr + wsPath

// reuseEnvKey 置为 1 时复用已经在跑的集成测试环境，不再自行构建与启停
const reuseEnvKey = "NETSVR_IT_REUSE"

// stack 被测的网关 + 业务进程
type stack struct {
	// reused 为 true 时表示环境是外部已有的，本进程不负责启停
	reused bool
	// binDir 本次构建产物的临时目录
	binDir string
	// procs 本次拉起的进程，顺序与 names 一致
	procs []*exec.Cmd
	names []string
	// logs 各进程的输出，用于启动失败时排查
	logs []*bytes.Buffer
	// exited 任一进程退出时写入其名字
	exited chan string
}

// moduleRoot 返回仓库根目录（本文件位于 <root>/test/integration/）
func moduleRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func isPortOpen(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// startStack 准备被测环境：优先复用，其次自行构建并拉起
func startStack() (*stack, error) {
	st := &stack{exited: make(chan string, 4)}
	if os.Getenv(reuseEnvKey) == "1" {
		st.reused = true
		return st, nil
	}
	for _, addr := range []string{customerAddr, workerAddr, taskAddr} {
		if isPortOpen(addr) {
			return nil, fmt.Errorf(
				"端口 %s 已被占用：请先停止占用该端口的进程，或设置环境变量 %s=1 复用已运行的环境",
				addr, reuseEnvKey,
			)
		}
	}
	root := moduleRoot()
	dir, err := os.MkdirTemp("", "netsvr-integration-")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败：%v", err)
	}
	st.binDir = dir
	if err := st.build(root); err != nil {
		st.stop()
		return nil, err
	}
	if err := st.startGateway(root); err != nil {
		st.stop()
		return nil, err
	}
	// 业务进程注册到 worker 失败时会直接退出，所以必须等 worker/task 先就绪再拉业务进程
	if err := waitPort(workerAddr, 30*time.Second); err != nil {
		st.stop()
		return nil, fmt.Errorf("等待网关 worker 服务就绪失败：%v\n%s", err, st.dumpLogs())
	}
	if err := waitPort(taskAddr, 30*time.Second); err != nil {
		st.stop()
		return nil, fmt.Errorf("等待网关 task 服务就绪失败：%v\n%s", err, st.dumpLogs())
	}
	if err := st.startBusiness(root); err != nil {
		st.stop()
		return nil, err
	}
	if err := st.waitReady(60 * time.Second); err != nil {
		st.stop()
		return nil, err
	}
	return st, nil
}

// build 编译网关与业务进程
func (s *stack) build(root string) error {
	targets := []struct {
		pkg  string
		name string
	}{
		{"./cmd", "netsvr-integration"},
		{"./test/business/cmd", "business-integration"},
	}
	for _, target := range targets {
		out := filepath.Join(s.binDir, target.name+exeSuffix())
		cmd := exec.Command("go", "build", "-o", out, target.pkg)
		cmd.Dir = root
		var buf bytes.Buffer
		cmd.Stdout, cmd.Stderr = &buf, &buf
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("构建 %s 失败：%v\n%s", target.pkg, err, buf.String())
		}
	}
	return nil
}

func (s *stack) start(root, bin, config, name string) error {
	buf := &bytes.Buffer{}
	cmd := exec.Command(filepath.Join(s.binDir, bin+exeSuffix()), "-config", filepath.Join(root, "test", "integration", "configs", config))
	cmd.Dir = root
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动%s失败：%v", name, err)
	}
	s.procs = append(s.procs, cmd)
	s.names = append(s.names, name)
	s.logs = append(s.logs, buf)
	go func(cmd *exec.Cmd, name string) {
		_ = cmd.Wait()
		select {
		case s.exited <- name:
		default:
		}
	}(cmd, name)
	return nil
}

func (s *stack) startGateway(root string) error {
	return s.start(root, "netsvr-integration", "netsvr.toml", "网关")
}

func (s *stack) startBusiness(root string) error {
	return s.start(root, "business-integration", "business.toml", "业务进程")
}

// waitReady 用一次真实的「连接 -> 收到连接打开回包」来确认整条链路已经打通
func (s *stack) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case name := <-s.exited:
			return fmt.Errorf("%s 进程已退出，无法完成就绪探测\n%s", name, s.dumpLogs())
		default:
		}
		if err := probeHandshake(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("等待业务进程注册到网关超时（最后一次探测失败：%v）\n%s", lastErr, s.dumpLogs())
}

// probeHandshake 连一次网关并等待业务进程回一条消息
func probeHandshake() error {
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := conn.ReadMessage(); err != nil {
		return fmt.Errorf("未收到业务进程的连接打开回包：%v", err)
	}
	return nil
}

func waitPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortOpen(addr) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("等待端口 %s 监听超时", addr)
}

func (s *stack) dumpLogs() string {
	var buf bytes.Buffer
	for i, name := range s.names {
		buf.WriteString("===== " + name + " 输出 =====\n")
		buf.WriteString(s.logs[i].String())
		buf.WriteString("\n")
	}
	return buf.String()
}

// stop 回收本次拉起的进程；复用外部环境时不做任何事
func (s *stack) stop() {
	if s.reused {
		return
	}
	for _, cmd := range s.procs {
		if cmd.Process == nil {
			continue
		}
		_ = cmd.Process.Kill()
	}
	if s.binDir != "" {
		_ = os.RemoveAll(s.binDir)
	}
}
