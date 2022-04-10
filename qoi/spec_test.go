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
		magic:      qoiMagicBytes,
		width:      400,
		height:     400,
		channels:   4,
		colorspace: 1,
	}
	expectedBytes := make([]byte, 0, headerLength)
	expectedBuf := bytes.NewBuffer(expectedBytes)
	err := binary.Write(expectedBuf, binary.BigEndian, header.magic)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.width)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.height)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.channels)
	require.NoError(t, err)
	err = binary.Write(expectedBuf, binary.BigEndian, header.colorspace)
	require.NoError(t, err)
	answerBuf := new(bytes.Buffer)
	err = header.write(answerBuf)
	require.NoError(t, err)
	assert.EqualValues(t, expectedBuf.Bytes(), answerBuf.Bytes())
}
