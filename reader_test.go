package qoi

import (
	"github.com/stretchr/testify/assert"
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
