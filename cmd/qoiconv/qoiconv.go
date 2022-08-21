package main

import (
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"log"
	"os"
	"qoi/qoi"
	"strings"
)

const usage = `Usage: qoiconv <infile> <outfile>
Examples:
	qoiconv input.png output.qoi
	qoiconv input.qoi output.png`

func main() {
	if len(os.Args) != 3 {
		printUsage()
		return
	}

	inputFilename := os.Args[1]
	outputFilename := os.Args[2]

	inputImg := openImage(inputFilename)

	if !isQOIFilename(outputFilename) {
		writeGenericImage(inputImg, outputFilename)
		return
	}

	writeQOIImage(inputImg, outputFilename)
}

func printUsage() {
	fmt.Println(usage)
}

func openImage(filename string) image.Image {
	inputImg, err := imaging.Open(filename)
	checkForUnsupportedFormat(err)
	if err != nil {
		log.Fatalf("Could not open the input image: %v", err)
	}
	return inputImg
}

func checkForUnsupportedFormat(err error) {
	if errors.Is(err, imaging.ErrUnsupportedFormat) {
		fmt.Println("The only supported formats are png, jpeg, bmp, tiff & qoi")
		os.Exit(1)
	}
}

func isQOIFilename(filename string) bool {
	return strings.HasSuffix(filename, ".qoi")
}

func writeGenericImage(img image.Image, outputFilename string) {
	err := imaging.Save(img, outputFilename)
	checkForUnsupportedFormat(err)
	if err != nil {
		log.Fatalf("Could not save the output image: %v", err)
	}
}

func writeQOIImage(img image.Image, outputFilename string) {
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Could not open the output file: %v", err)
	}
	err = qoi.Encode(outputFile, img)
	if err != nil {
		log.Fatalf("Could not encode the image: %v", err)
	}
}
