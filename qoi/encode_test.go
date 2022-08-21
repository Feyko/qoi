package qoi

import (
	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/require"
	"image"
	_ "image/png"
	"os"
	"testing"
)

func TestEncode(t *testing.T) {
	pngFile, err := os.Open("testimages/thonk.png")
	require.NoErrorf(t, err, "Could not read the PNG test image: %v", err)

	pngImg, _, err := image.Decode(pngFile)
	require.NoErrorf(t, err, "Could not decode the PNG test image: %v", err)

	qoiFile, err := os.OpenFile("testimages/thonk.qoi", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	require.NoErrorf(t, err, "Could not open the QOI output image: %v", err)

	err = NewEncoder(qoiFile, pngImg).Encode()
	require.NoErrorf(t, err, "Could not encode the test image: %v", err)

	qoiFile, err = os.Open("testimages/thonk.qoi")
	require.NoErrorf(t, err, "Could not open the encoded image: %v", err)

	qoiImg, err := Decode(qoiFile)
	require.NoErrorf(t, err, "Could not decode the encoded image: %v", err)

	imaging.Save(qoiImg, "testimages/thonk.qoi.png")
	require.EqualValuesf(t, pngImg, qoiImg, "The image was not encoded properly")
}
