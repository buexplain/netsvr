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

// Package integration 是网关的端到端集成测试。
//
// 它把 test/business/assets/client.html 这套「人工点按钮 + 肉眼看输出」的验收方式自动化：
// 测试进程直接以 websocket 客户端身份连上网关（和手工测试页一样），按同一套
// {"cmd":<命令>, "data":"<参数>"} 协议发命令，再对业务进程回包做断言。
//
// 命令集合、参数形状、回包形状都对齐 client.html 与 test/business 的实现，
// 因此这层用例覆盖的是「客户 -> 网关 -> 业务进程 -> 网关 -> 客户」的完整链路，
// 而不是网关内部的单元逻辑（内部逻辑由 internal/** 下的单测覆盖）。
//
// 文件组织：
//
//	环境   main_test.go      TestMain：启动/回收被测环境
//	       stack_test.go     构建并拉起网关与业务进程、探测就绪
//	辅助   client_test.go    测试客户端（连接、收发、接收断言）
//	       protocol_test.go  命令与回包的报文结构、各命令回包 data 的形状
//	       assert_test.go    通用断言
//	       fixture_test.go   用例共用的测试数据与准备动作
//	用例   conn_test.go      连接、uniqId/customerId 查询、在线检查、统计、限流
//	       sign_test.go      登录 / 退出 / 伪造登录
//	       send_test.go      单播、批量单播、组播、广播
//	       topic_test.go     主题订阅、退订、删除、发布与主题查询
//	       offline_test.go   三种强制下线
//	       assets_test.go    手工测试页渲染
//
// 运行方式：
//
//	go test ./test/integration/...
//
// 用例默认自行构建并拉起一套独立的网关 + 业务进程（监听 6160/6161/6162/6164，
// 与开发默认端口 6060/6061/6062 错开），跑完自动回收。
// 若已经手工起好了同一套环境，可用 NETSVR_IT_REUSE=1 复用，避免重复构建。
package integration

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	st, err := startStack()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "集成测试环境启动失败：")
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	code := m.Run()
	if code != 0 && !st.reused {
		_, _ = fmt.Fprintln(os.Stderr, "集成测试失败，被测进程输出如下：")
		_, _ = fmt.Fprintln(os.Stderr, st.dumpLogs())
	}
	st.stop()
	os.Exit(code)
}
