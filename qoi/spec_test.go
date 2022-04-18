package qoi

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHeaderWrite(t *testing.T) {
	header := Header{
		Magic:      QoiMagicBytes,
		Width:      400,
		Height:     400,
		Channels:   4,
		Colorspace: 1,
	}
	expectedBytes := make([]byte, 0, headerLength)
	expectedBuf := bytes.NewBuffer(expectedBytes)
	err := binary.Write(expectedBuf, binary.BigEndian, header.Magic)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.Width)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.Height)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.Channels)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.Colorspace)
	require.NoError(t, err)
	answerBuf := new(bytes.Buffer)
	err = header.write(answerBuf)
	require.NoError(t, err)
	assert.EqualValues(t, expectedBuf.Bytes(), answerBuf.Bytes())
}
