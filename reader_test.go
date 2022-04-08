package qoi

import (
	"github.com/stretchr/testify/assert"
	"image"
	_ "image/png"
	"os"
	"testing"
)

func TestDecodeConfig(t *testing.T) {
	file, err := os.Open("testimage.qoi")
	if err != nil {
		t.Fatalf("Could not read the test image: %v", err)
	}
	cfg, err := DecodeConfig(file)
	if err != nil {
		t.Fatalf("Could not decode the config: %v", err)
	}
	expectedWidth := 492
	expectedHeight := 445
	assert.Equal(t, expectedWidth, cfg.Width)
	assert.Equal(t, expectedHeight, cfg.Height)
}

func TestDecode(t *testing.T) {
	qoiFile, err := os.Open("testimage.qoi")
	assert.NoErrorf(t, err, "Could not read the QOI test image: %w", err)
	pngFile, err := os.Open("testimage.qoi")
	assert.NoErrorf(t, err, "Could not read the PNG test image: %w", err)
	qoiImg, _, err := image.Decode(qoiFile)
	assert.NoErrorf(t, err, "Could not decode the QOI test image: %w", err)
	pngImg, _, err := image.Decode(pngFile)
	assert.NoErrorf(t, err, "Could not decode the PNG test image: %w", err)
	assert.EqualValues(t, pngImg, qoiImg)
}
