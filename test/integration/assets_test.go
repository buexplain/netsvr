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

package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestClientPage 业务进程托管的手工测试页必须能被正常渲染。
// 页面里的命令是由 protocol.CmdName 注入的模板占位符，这里用来守住「页面与协议同步」：
// 一旦命令改名或漏注册，占位符就会残留，或页面缺少对应按钮。
func TestClientPage(t *testing.T) {
	resp, err := (&http.Client{Timeout: readTimeout}).Get("http://" + clientAddr + "/")
	if err != nil {
		t.Fatalf("访问手工测试页失败：%v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("手工测试页状态码不符合预期：%d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取手工测试页失败：%v", err)
	}
	page := string(body)
	if strings.Contains(page, "{!") {
		t.Fatalf("手工测试页仍有未替换的模板占位符")
	}
	// 批量广播是本套件新增覆盖的命令，页面上必须有对应入口
	if !strings.Contains(page, "broadcastBulk:") {
		t.Fatalf("手工测试页缺少 broadcastBulk 命令")
	}
}
