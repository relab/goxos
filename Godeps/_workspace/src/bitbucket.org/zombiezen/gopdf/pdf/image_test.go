package pdf

import (
	"code.google.com/p/go.image/bmp"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"testing"
)

const suzanneBytes = 512 * 512 * 3

func loadSuzanneRGBA() (*image.RGBA, error) {
	f, err := os.Open("testdata/suzanne.bmp")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := bmp.Decode(f)
	if err != nil {
		return nil, err
	}
	return img.(*image.RGBA), nil
}

func loadSuzanneNRGBA() (*image.NRGBA, error) {
	f, err := os.Open("testdata/suzanne.png")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return img.(*image.NRGBA), nil
}

func loadSuzanneYCbCr() (*image.YCbCr, error) {
	f, err := os.Open("testdata/suzanne.jpg")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return nil, err
	}
	return img.(*image.YCbCr), nil
}

func BenchmarkEncodeRGBAGeneric(b *testing.B) {
	b.StopTimer()
	img, _ := loadSuzanneRGBA()
	b.SetBytes(suzanneBytes)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		encodeImageStream(ioutil.Discard, img)
	}
}

func BenchmarkEncodeRGBA(b *testing.B) {
	b.StopTimer()
	img, _ := loadSuzanneRGBA()
	b.SetBytes(suzanneBytes)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		encodeRGBAStream(ioutil.Discard, img)
	}
}

func BenchmarkEncodeNRGBAGeneric(b *testing.B) {
	b.StopTimer()
	img, _ := loadSuzanneNRGBA()
	b.SetBytes(suzanneBytes)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		encodeImageStream(ioutil.Discard, img)
	}
}

func BenchmarkEncodeNRGBA(b *testing.B) {
	b.StopTimer()
	img, _ := loadSuzanneNRGBA()
	b.SetBytes(suzanneBytes)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		encodeNRGBAStream(ioutil.Discard, img)
	}
}

func BenchmarkEncodeYCbCrGeneric(b *testing.B) {
	b.StopTimer()
	img, _ := loadSuzanneYCbCr()
	b.SetBytes(suzanneBytes)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		encodeImageStream(ioutil.Discard, img)
	}
}

func BenchmarkEncodeYCbCr(b *testing.B) {
	b.StopTimer()
	img, _ := loadSuzanneYCbCr()
	b.SetBytes(suzanneBytes)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		encodeYCbCrStream(ioutil.Discard, img)
	}
}
