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

package buffer

import (
	"fmt"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"testing"
)

func TestBasic(t *testing.T) {
	bytes := byteslice.Get(16)
	b := New(bytes[:0])             //构建的保证元素个数是0
	_, _ = b.Write([]byte("hello")) //Write方法底层是append追加数据
	if string(b.Bytes()) != "hello" {
		t.Fatal("buffer is not equal", string(b.Bytes()))
	}
	fmt.Println(len(bytes), cap(bytes), string(bytes))
	b.Discard()
	byteslice.Put(bytes)
	bytes = byteslice.Get(16)
	fmt.Println(len(bytes), cap(bytes), string(bytes))
}
