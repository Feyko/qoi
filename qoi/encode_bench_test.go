package qoi

import (
	"bytes"
	"github.com/disintegration/imaging"
	"image"
	"testing"
)

func BenchmarkEncode(b *testing.B) {
	inputImg, err := imaging.Open("testimages/thonk.png")
	if err != nil {
		b.Fatalf("Could not open the test image: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		Encode(&buf, inputImg)
	}
}

func BenchmarkNRGGBAConv(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 2000, 2000))
	b.ResetTimer()
	var converted image.Image
	for i := 0; i < b.N; i++ {
		converted = convertImageToNRGBA(img)
	}
	converted = converted
}
