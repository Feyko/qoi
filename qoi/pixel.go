package qoi

import "image/color"

var c int

type pixelBytes [4]byte

type pixel struct {
	v    pixelBytes
	hash byte
}

func newPixel(v pixelBytes) (p pixel) {
	p = pixel{v: v}
	p.calculateHash()
	return
}

func pixelFromColor(c color.Color) pixel {
	r, g, b, a := c.RGBA()
	return newPixel([4]byte{byte(r), byte(g), byte(b), byte(a)})
}

func (p pixel) R() byte {
	return p.v[0]
}

func (p pixel) G() byte {
	return p.v[1]
}

func (p pixel) B() byte {
	return p.v[2]
}

func (p pixel) A() byte {
	return p.v[3]
}

// the mulX methods allow for some compiler Magic to minimally enhance performance. Don't ask me how it works. Also helps with profiling
func (p pixel) mulR() byte {
	return p.R() * 3
}

func (p pixel) mulG() byte {
	return p.G() * 5
}

func (p pixel) mulB() byte {
	return p.B() * 7
}

func (p pixel) mulA() byte {
	return p.A() * 11
}

func (p *pixel) Add(r, g, b byte) {
	p.v[0] += r
	p.v[1] += g
	p.v[2] += b
	p.calculateHash()
}

func (p pixel) Minus(p2 pixel) (r, g, b, a int8) {
	return int8(p.R()) - int8(p2.R()), int8(p.G()) - int8(p2.G()), int8(p.B()) - int8(p2.B()), int8(p.A()) - int8(p2.A())
}

func (p pixel) Hash() byte {
	return p.hash
}

func (p *pixel) calculateHash() {
	p.hash = (p.mulR() + p.mulG() + p.mulB() + p.mulA()) % windowLength
}
