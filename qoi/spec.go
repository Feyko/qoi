package qoi

const (
	quoi_OP_RGB   byte = 0b11111110
	quoi_OP_RGBA  byte = 0b11111111
	quoi_OP_INDEX byte = 0b00
	quoi_OP_DIFF  byte = 0b01
	quoi_OP_LUMA  byte = 0b10
	quoi_OP_RUN   byte = 0b11

	quoi_2B_MASK byte = 0b11
)

func getOP(b byte) byte {
	masked := b & quoi_2B_MASK
	switch masked {
	case quoi_OP_INDEX, quoi_OP_DIFF, quoi_OP_LUMA, quoi_OP_RUN:
		return masked
	default:
		return b
	}
}

const headerLength = 4 + 4 + 4 + 1 + 1

const qoiMagic = "qoif"
