// Copyright (C) 2011, Ross Light

package pdf

import (
	"image"
	"image/color"
	"io"
)

const (
	deviceRGBColorSpace name = "DeviceRGB"
)

type imageStream struct {
	*stream
	Width            int
	Height           int
	BitsPerComponent int
	ColorSpace       name
}

type imageStreamInfo struct {
	Type             name
	Subtype          name
	Length           int
	Filter           name `pdf:",omitempty"`
	Width            int
	Height           int
	BitsPerComponent int
	ColorSpace       name
}

func newImageStream(filter name, w, h int) *imageStream {
	return &imageStream{
		stream:           newStream(filter),
		Width:            w,
		Height:           h,
		BitsPerComponent: 8,
		ColorSpace:       deviceRGBColorSpace,
	}
}

func (st *imageStream) marshalPDF(dst []byte) ([]byte, error) {
	return marshalStream(dst, imageStreamInfo{
		Type:             xobjectType,
		Subtype:          imageSubtype,
		Length:           st.Len(),
		Filter:           st.filter,
		Width:            st.Width,
		Height:           st.Height,
		BitsPerComponent: st.BitsPerComponent,
		ColorSpace:       st.ColorSpace,
	}, st.Bytes())
}

// encodeImageStream writes RGB data from an image in PDF format.
func encodeImageStream(w io.Writer, img image.Image) error {
	bd := img.Bounds()
	var buf [3]byte
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a != 0 {
				buf[0] = byte((r * 65535 / a) >> 8)
				buf[1] = byte((g * 65535 / a) >> 8)
				buf[2] = byte((b * 65535 / a) >> 8)
			} else {
				buf[0], buf[1], buf[2] = 0, 0, 0
			}
			if _, err := w.Write(buf[:]); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeRGBAStream(w io.Writer, img *image.RGBA) error {
	var rgb [3]uint8
	var a uint16
	for i := 0; i < len(img.Pix); i += 4 {
		a = uint16(img.Pix[i+3])
		if a != 0 {
			rgb[0] = uint8(uint16(img.Pix[i]) * 65535 / a)
			rgb[1] = uint8(uint16(img.Pix[i+1]) * 65535 / a)
			rgb[2] = uint8(uint16(img.Pix[i+2]) * 65535 / a)
		} else {
			rgb[0], rgb[1], rgb[2] = 0, 0, 0
		}
		if _, err := w.Write(rgb[:]); err != nil {
			return err
		}
	}
	return nil
}

func encodeNRGBAStream(w io.Writer, img *image.NRGBA) error {
	for i := 0; i < len(img.Pix); i += 4 {
		if _, err := w.Write(img.Pix[i : i+3]); err != nil {
			return err
		}
	}
	return nil
}

func encodeYCbCrStream(w io.Writer, img *image.YCbCr) error {
	var buf [3]byte
	var yy, cb, cr byte
	var i, j int
	dx, dy := img.Rect.Dx(), img.Rect.Dy()
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			i, j = x, y
			switch img.SubsampleRatio {
			case image.YCbCrSubsampleRatio420:
				j /= 2
				fallthrough
			case image.YCbCrSubsampleRatio422:
				i /= 2
			}
			yy = img.Y[y*img.YStride+x]
			cb = img.Cb[j*img.CStride+i]
			cr = img.Cr[j*img.CStride+i]

			buf[0], buf[1], buf[2] = color.YCbCrToRGB(yy, cb, cr)
			if _, err := w.Write(buf[:]); err != nil {
				return err
			}
		}
	}
	return nil
}
