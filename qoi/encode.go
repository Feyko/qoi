package qoi

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/exp/slices"
	"image"
	"image/color"
	"io"
	"runtime"
	"sync"
)

var stripCount = runtime.NumCPU() * 8

//Encode writes the Image m to w in PNG format. Any Image may be encoded, but images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	return NewEncoder(w, m).Encode()
}

type Encoder struct {
	out    *bufio.Writer
	img    *image.NRGBA
	Header Header
	window [windowLength]pixel
	strips []*strip
	//previousPixel, currentPixel  pixel
	//diffR, diffG, diffB, diffA   int8
	//minX, maxX, minY, maxY, x, y int
}

type strip struct {
	n, stripCount                int
	out                          bytes.Buffer
	img                          *image.NRGBA
	window                       [windowLength]pixel
	previousPixel, currentPixel  pixel
	diffR, diffG, diffB, diffA   int8
	minX, maxX, minY, maxY, x, y int
	done                         bool
}

type result struct {
	n int
	b []byte
}

func NewEncoder(out io.Writer, img image.Image) Encoder {
	if !isImageNRGBA(img) {
		img = convertImageToNRGBA(img)
	}
	return Encoder{out: bufio.NewWriter(out), img: img.(*image.NRGBA), strips: make([]*strip, stripCount)}
}

func isImageNRGBA(img image.Image) bool {
	return img.ColorModel() == color.NRGBAModel
}

func convertImageToNRGBA(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewNRGBA(bounds)
	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			newImg.Set(x, y, img.At(x, y))
		}
	}
	return newImg
}

func (enc Encoder) Encode() error {
	width := enc.img.Bounds().Size().X
	height := enc.img.Bounds().Size().Y
	header := Header{
		Magic:      QoiMagicBytes,
		Width:      uint32(width),
		Height:     uint32(height),
		Channels:   4,
		Colorspace: 1,
	}
	enc.Header = header
	err := enc.encodeHeader()
	if err != nil {
		return fmt.Errorf("could not encode the Header: %w", err)
	}
	err = enc.encodeBody()
	if err != nil {
		return fmt.Errorf("could not encode the body: %w", err)
	}
	return nil
}

func (enc *Encoder) encodeHeader() error {
	return enc.Header.write(enc.out)
}

func (enc *Encoder) encodeBody() error {
	for i := 0; i < stripCount; i++ {
		enc.strips[i] = enc.newStrip(i, stripCount)
	}

	results := make(chan result)
	r := make([][]byte, stripCount)
	var group sync.WaitGroup
	group.Add(stripCount)

	go func() {
		for {
			res := <-results
			r[res.n] = res.b
			group.Done()
		}
	}()

	for _, strp := range enc.strips {
		strp := strp
		go func() {
			var b []byte
			err := strp.encodeBody(&b)
			if err != nil {
				panic(err)
			}
			results <- result{strp.n, b}
		}()
	}
	group.Wait()
	for i := 0; i < stripCount; i++ {
		_, err := enc.out.Write(r[i])
		if err != nil {
			panic(err)
		}
	}

	_, err := enc.out.Write([]byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		return fmt.Errorf("could not write the end padding: %w", err)
	}
	err = enc.out.Flush()
	if err != nil {
		return fmt.Errorf("could not flush data: %w", err)
	}

	return nil
}

func (enc *Encoder) newStrip(n, stripCount int) *strip {
	strp := &strip{n: n, stripCount: stripCount, img: enc.img}
	strp.setupBounds()
	return strp
}

func (strp *strip) encodeBody(r *[]byte) error {

	strp.currentPixel = pixelFromColor(strp.img.At(strp.maxX, strp.minY-1))

	strp.setupPosition()

	strp.advancePixel()
	for !strp.done {
		err := strp.dispatchOP()
		if err != nil {
			return err
		}
	}
	*r = slices.Clone(strp.out.Bytes())
	return nil
}

func (strp *strip) setupBounds() {
	bounds := strp.img.Bounds()
	stripCount := strp.stripCount
	commonStripSize := bounds.Max.Y/stripCount + 1
	thisStripSize := commonStripSize

	if strp.n+1 == stripCount {
		thisStripSize += commonStripSize*stripCount - bounds.Max.Y
	}

	strp.minX = bounds.Min.X
	strp.maxX = bounds.Max.X - 1 // 'Max' is the size of the image, not the maximum index we can use
	strp.minY = commonStripSize * strp.n
	strp.maxY = strp.minY + thisStripSize
	if strp.minY != 0 {
		strp.minY += 1
	}
}

func (strp *strip) setupPosition() {
	strp.x = strp.minX - 1 // Initialise one step back for the first update to land on the first pixel
	strp.y = strp.minY
}

func (strp *strip) advancePixel() {
	strp.updatePosition()
	pix := strp.img.At(strp.x, strp.y).(color.NRGBA)
	strp.previousPixel = strp.currentPixel
	strp.currentPixel = newPixel(pixelBytes{pix.R, pix.G, pix.B, pix.A})
}

func (strp *strip) updatePosition() {
	if strp.x == strp.maxX && strp.y == strp.maxY {
		strp.done = true
		return
	}
	if strp.x == strp.maxX {
		strp.y++
		strp.x = strp.minX
	} else {
		strp.x++
	}
	return
}

func (strp *strip) cacheCurrentPixel() {
	strp.window[strp.currentPixel.Hash()] = strp.currentPixel // We do not check for equality as copying a 4B array is faster than checking
}

func (strp *strip) dispatchOP() error {
	if strp.currentPixel == strp.previousPixel {
		return strp.op_RUN()
	}
	if strp.window[strp.currentPixel.hash] == strp.currentPixel {
		return strp.op_INDEX()
	}
	strp.cacheCurrentPixel()
	strp.calculateDiff()
	if strp.diffA != 0 {
		return strp.op_RGBA()
	}
	if strp.isCurrentPixelWithinDIFFSpec() {
		return strp.op_DIFF()
	}
	if strp.isCurrentPixelWithinLUMASpec() {
		return strp.op_LUMA()
	}

	return strp.op_RGB()
}

func (strp *strip) calculateDiff() {
	strp.diffR, strp.diffG, strp.diffB, strp.diffA = strp.currentPixel.Minus(strp.previousPixel)
}

func (strp *strip) isCurrentPixelWithinDIFFSpec() bool {
	return isValueWithinDIFFSpec(strp.diffR) && isValueWithinDIFFSpec(strp.diffG) && isValueWithinDIFFSpec(strp.diffB)
}

func (strp *strip) isCurrentPixelWithinLUMASpec() bool {
	return isValueWithinLUMASpec(strp.diffR-strp.diffG) && isGreenValueWithinLUMASpec(strp.diffG) && isValueWithinLUMASpec(strp.diffB-strp.diffG)
}

func (strp *strip) op_RGB() error {
	err := strp.out.WriteByte(QOI_OP_RGB)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	_, err = strp.out.Write(strp.currentPixel.v[:3])
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	strp.advancePixel()
	return nil
}

func (strp *strip) op_RGBA() error {
	err := strp.out.WriteByte(QOI_OP_RGBA)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	_, err = strp.out.Write(strp.currentPixel.v[:])
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	strp.advancePixel()
	return nil
}

func (strp *strip) op_INDEX() error {
	err := strp.out.WriteByte(QOI_OP_INDEX | strp.currentPixel.hash)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	strp.advancePixel()
	return nil
}

func (strp *strip) op_DIFF() error {
	r := byte(strp.diffR+diffBias) << 4
	g := byte(strp.diffG+diffBias) << 2
	b := byte(strp.diffB + diffBias)
	err := strp.out.WriteByte(QOI_OP_DIFF | r | g | b)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	strp.advancePixel()
	return nil
}

func (strp *strip) op_LUMA() error {
	directionRG := byte(strp.diffR - strp.diffG + lumaBias)
	directionBG := byte(strp.diffB - strp.diffG + lumaBias)
	err := strp.out.WriteByte(QOI_OP_LUMA | byte(strp.diffG+lumaGreenBias))
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	err = strp.out.WriteByte(directionRG<<4 | directionBG)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	strp.advancePixel()
	return nil
}

func (strp *strip) op_RUN() error {
	count := 1
	strp.advancePixel()
	for strp.currentPixel == strp.previousPixel && !strp.done {
		count++
		strp.advancePixel()
		if count == 62 {
			break
		}
	}
	err := strp.out.WriteByte(QOI_OP_RUN | byte(count) - runBias)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	return nil
}
