package qoi

import (
	"bytes"
	"os"
	"testing"
)

func BenchmarkDecode(b *testing.B) {
	data, err := os.ReadFile("testimage2.qoi")
	if err != nil {
		b.Fatalf("Could not open the test image: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		Decode(r)
	}
}
