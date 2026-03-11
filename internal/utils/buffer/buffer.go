package buffer

// Buffer is a simple wrapper around []byte
// 请自行保证buf的len和cap
type Buffer struct {
	b []byte
}

func New(buf []byte) *Buffer {
	if buf == nil {
		panic("buffer can not be nil")
	}
	return &Buffer{
		b: buf,
	}
}

func (b *Buffer) Set(buf []byte) {
	if buf == nil {
		panic("buffer can not be nil")
	}
	b.b = buf
}

func (b *Buffer) Len() int {
	return len(b.b)
}

func (b *Buffer) Bytes() []byte {
	return b.b
}

func (b *Buffer) Discard() {
	b.b = nil
}

// Write implements io.Writer
// append追加数据
func (b *Buffer) Write(p []byte) (int, error) {
	b.b = append(b.b, p...)
	return len(p), nil
}
