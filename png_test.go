// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qr

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"testing"
)

func TestPNG(t *testing.T) {
	c, err := Encode("hello, world", L)
	if err != nil {
		t.Fatal(err)
	}
	pngdat := c.PNG()
	if true {
		ioutil.WriteFile("x.png", pngdat, 0666)
	}
	m, err := png.Decode(bytes.NewBuffer(pngdat))
	if err != nil {
		t.Fatal(err)
	}

	testImage(t, c, m.(*image.Gray))
}

func TestImage(t *testing.T) {
	c, err := Encode("hello, world", L)
	if err != nil {
		t.Fatal(err)
	}

	img := c.Image()
	var buf bytes.Buffer
	png.Encode(&buf, img)
	if true {
		ioutil.WriteFile("y.png", buf.Bytes(), 0666)
	}
	m, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}

	testImage(t, c, m.(*image.Gray))
}

func testImage(t *testing.T, c *Code, gm *image.Gray) {
	t.Helper()

	scale := c.Scale
	siz := c.Size
	nbad := 0
	for y := 0; y < scale*(8+siz); y++ {
		for x := 0; x < scale*(8+siz); x++ {
			v := byte(255)
			if c.Black(x/scale-4, y/scale-4) {
				v = 0
			}
			if gv := gm.At(x, y).(color.Gray).Y; gv != v {
				t.Errorf("%d,%d = %d, want %d", x, y, gv, v)
				if nbad++; nbad >= 20 {
					t.Fatalf("too many bad pixels")
				}
			}
		}
	}
}

func BenchmarkPNG(b *testing.B) {
	c, err := Encode("0123456789012345678901234567890123456789", L)
	if err != nil {
		panic(err)
	}
	var bytes []byte
	for i := 0; i < b.N; i++ {
		bytes = c.PNG()
	}
	b.SetBytes(int64(len(bytes)))
}

func BenchmarkImagePNG(b *testing.B) {
	c, err := Encode("0123456789012345678901234567890123456789", L)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		png.Encode(&buf, c.Image())
	}
	b.SetBytes(int64(buf.Len()))
}
