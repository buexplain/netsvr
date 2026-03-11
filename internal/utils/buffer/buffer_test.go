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
