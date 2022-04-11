package qoi

type pixel [4]byte

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
	p[0] += r
	p[1] += g
	p[2] += b
}

func (p pixel) Hash() byte {
	return (p.mulR() + p.mulG() + p.mulB() + p.mulA()) % 64
}
