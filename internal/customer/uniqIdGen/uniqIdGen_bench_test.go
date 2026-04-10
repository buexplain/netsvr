package uniqIdGen

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New()
	}
}

// BenchmarkNewParallel 并行基准测试（模拟高并发场景）
func BenchmarkNewParallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New()
		}
	})
}
