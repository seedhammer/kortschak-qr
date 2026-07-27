// Copyright 2021 Dan Kortschak.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package qr encodes QR codes.
*/
package qr

import (
	"errors"
	"image"
	"image/color"

	"github.com/seedhammer/kortschak-qr/coding"
)

// A Level denotes a QR error correction level.
// From least to most tolerant of errors, they are L, M, Q, H.
type Level int

const (
	L Level = iota // 20% redundant
	M              // 38% redundant
	Q              // 55% redundant
	H              // 65% redundant
)

// Encode returns an encoding of text at the given error correction level.
func Encode(text string, level Level) (*Code, error) {
	v, err := minVersion(text, level)
	if err != nil {
		return nil, err
	}
	cc, err := coding.Encode(v, coding.Level(level), pickEncoding(text))
	if err != nil {
		return nil, err
	}
	return &Code{cc.Bitmap, cc.Size, cc.Stride, 8}, nil
}

// MinSize returns the pixel size of the smallest code that holds text
// at the given error correction level, without building the code. It
// costs a capacity scan instead of Encode's full plan construction,
// for callers that lay out space for a code before committing to the
// expense of encoding it.
func MinSize(text string, level Level) (int, error) {
	v, err := minVersion(text, level)
	if err != nil {
		return 0, err
	}
	return 17 + int(v)*4, nil
}

// pickEncoding returns the smallest data encoding for text. We could
// split the string and use different encodings but that seems like
// overkill for now.
func pickEncoding(text string) coding.Encoding {
	switch {
	case coding.Num(text).Check() == nil:
		return coding.Num(text)
	case coding.Alpha(text).Check() == nil:
		return coding.Alpha(text)
	default:
		return coding.String(text)
	}
}

// minVersion returns the smallest version whose capacity holds text
// at the given error correction level.
func minVersion(text string, level Level) (coding.Version, error) {
	enc := pickEncoding(text)
	l := coding.Level(level)
	for v := coding.Version(coding.MinVersion); ; v++ {
		if v > coding.MaxVersion {
			return 0, errors.New("text too long to encode as QR")
		}
		if enc.Bits(v) <= v.DataBytes(l)*8 {
			return v, nil
		}
	}
}

// A Code is a square pixel grid.
// It implements image.Image and direct PNG encoding.
type Code struct {
	Bitmap []byte // 1 is black, 0 is white
	Size   int    // number of pixels on a side
	Stride int    // number of bytes per row
	Scale  int    // number of image pixels per QR pixel
}

// Black returns true if the pixel at (x,y) is black.
func (c *Code) Black(x, y int) bool {
	return 0 <= x && x < c.Size && 0 <= y && y < c.Size &&
		c.Bitmap[y*c.Stride+x/8]&(1<<uint(7-x&7)) != 0
}

// Image returns an Image displaying the code.
func (c *Code) Image() image.Image {
	return &codeImage{c}

}

// codeImage implements image.Image
type codeImage struct {
	*Code
}

var (
	whiteColor color.Color = color.Gray{0xFF}
	blackColor color.Color = color.Gray{0x00}
)

func (c *codeImage) Bounds() image.Rectangle {
	d := (c.Size + 8) * c.Scale
	return image.Rect(0, 0, d, d)
}

func (c *codeImage) At(x, y int) color.Color {
	if c.Black(x/c.Scale-4, y/c.Scale-4) {
		return blackColor
	}
	return whiteColor
}

func (c *codeImage) ColorModel() color.Model {
	return color.GrayModel
}
