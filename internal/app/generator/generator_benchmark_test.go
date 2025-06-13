package generator

import (
	"testing"
)

func BenchmarkGenerate(b *testing.B) {
	g := NewGenerator(8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate()
	}
}
