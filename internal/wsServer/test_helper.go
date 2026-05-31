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

package wsServer

// SetCompressionForTest 测试辅助函数：设置连接的压缩标志
// 仅用于测试目的，不应在生产代码中使用
func SetCompressionForTest(conn *Conn, compress bool) {
	conn.setCompression(compress)
}
