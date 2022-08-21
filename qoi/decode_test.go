package qoi

import (
	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"image"
	_ "image/png"
	"os"
	"testing"
)

func TestDecodeConfig(t *testing.T) {
	file, err := os.Open("testimages/noanswer.qoi")
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
	qoiFile, err := os.Open("testimages/noanswer.qoi")
	require.NoErrorf(t, err, "Could not read the QOI test image: %v", err)
	pngFile, err := os.Open("testimages/noanswer.png")
	require.NoErrorf(t, err, "Could not read the PNG test image: %v", err)
	qoiImg, _, err := image.Decode(qoiFile)
	require.NoErrorf(t, err, "Could not decode the QOI test image: %v", err)
	pngImg, _, err := image.Decode(pngFile)
	require.NoErrorf(t, err, "Could not decode the PNG test image: %v", err)
	assert.EqualValues(t, pngImg, qoiImg)
	imaging.Save(qoiImg, "qoi.png")
}
