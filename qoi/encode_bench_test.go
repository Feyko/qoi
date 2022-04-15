package qoi

import (
	"bytes"
	"github.com/disintegration/imaging"
	"testing"
)

func BenchmarkEncode(b *testing.B) {
	inputImg, err := imaging.Open("testimage2.qoi")
	if err != nil {
		b.Fatalf("Could not open the test image: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		Encode(&buf, inputImg)
	}
}
