package qoi

var c int

type pixel struct {
	v    [4]byte
	hash byte
}

func newPixel(v [4]byte) (p pixel) {
	p = pixel{v: v}
	p.calculateHash()
	return
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

// the mulX methods allow for some compiler magic to minimally enhance performance. Don't ask me how it works. Also helps with profiling
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

func (p pixel) Hash() byte {
	return p.hash
}

func (p *pixel) calculateHash() {
	p.hash = (p.mulR() + p.mulG() + p.mulB() + p.mulA()) % 64
}
