package qoi

type pixel []byte

func (p pixel) R() byte {
	return p[0]
}

func (p pixel) G() byte {
	return p[1]
}

func (p pixel) B() byte {
	return p[2]
}

func (p pixel) A() byte {
	return p[3]
}

func (p pixel) Components() (byte, byte, byte, byte) {
	return p.R(), p.G(), p.B(), p.A()
}

func (p pixel) Add(r, g, b byte) {
	p[0] += r
	p[1] += g
	p[2] += b
}

func (p pixel) Hash() byte {
	return (p.R()*3 + p.G()*5 + p.B()*7 + p.A()*11) % 64
}
