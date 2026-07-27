// Copyright 2021 Dan Kortschak.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coding

import (
	"bytes"
	"testing"

	"github.com/seedhammer/kortschak-qr/gf256"
	"github.com/seedhammer/kortschak-qr/libqrencode"
)

func test(t *testing.T, v Version, l Level, text ...Encoding) bool {
	s := ""
	ty := libqrencode.EightBit
	switch x := text[0].(type) {
	case String:
		s = string(x)
	case Alpha:
		s = string(x)
		ty = libqrencode.Alphanumeric
	case Num:
		s = string(x)
		ty = libqrencode.Numeric
	}
	key, err := libqrencode.Encode(libqrencode.Version(v), libqrencode.Level(l), ty, s)
	if err != nil {
		t.Errorf("libqrencode.Encode(%v, %v, %d, %#q): %v", v, l, ty, s, err)
		return false
	}
	mask := (^key.Pixel[8][2]&1)<<2 | (key.Pixel[8][3]&1)<<1 | (^key.Pixel[8][4] & 1)
	p, err := NewPlan(v, l, Mask(mask))
	if err != nil {
		t.Errorf("NewPlan(%v, L, %d): %v", v, err, mask)
		return false
	}
	if len(p.Pixel) != len(key.Pixel) {
		t.Errorf("%v: NewPlan uses %dx%d, libqrencode uses %dx%d", v, len(p.Pixel), len(p.Pixel), len(key.Pixel), len(key.Pixel))
		return false
	}
	c, err := p.Encode(text...)
	if err != nil {
		t.Errorf("Encode: %v", err)
		return false
	}
	badpix := 0
Pixel:
	for y, prow := range p.Pixel {
		for x, pix := range prow {
			pix &^= Black
			if c.Black(x, y) {
				pix |= Black
			}

			keypix := key.Pixel[y][x]
			want := Pixel(0)
			switch {
			case keypix&libqrencode.Finder != 0:
				want = Position.Pixel()
			case keypix&libqrencode.Alignment != 0:
				want = Alignment.Pixel()
			case keypix&libqrencode.Timing != 0:
				want = Timing.Pixel()
			case keypix&libqrencode.Format != 0:
				want = Format.Pixel()
				want |= OffsetPixel(pix.Offset()) // sic
				want |= pix & Invert
			case keypix&libqrencode.PVersion != 0:
				want = PVersion.Pixel()
			case keypix&libqrencode.DataECC != 0:
				if pix.Role() == Check || pix.Role() == Extra {
					want = pix.Role().Pixel()
				} else {
					want = Data.Pixel()
				}
				want |= OffsetPixel(pix.Offset())
				want |= pix & Invert
			default:
				want = Unused.Pixel()
			}
			if keypix&libqrencode.Black != 0 {
				want |= Black
			}
			if pix != want {
				t.Errorf("%v/%v: Pixel[%d][%d] = %v, want %v %#x", v, mask, y, x, pix, want, keypix)
				if badpix++; badpix >= 100 {
					t.Errorf("stopping after %d bad pixels", badpix)
					break Pixel
				}
			}
		}
	}
	return badpix == 0
}

var input = []Encoding{
	String("hello"),
	Num("1"),
	Num("12"),
	Num("123"),
	Alpha("AB"),
	Alpha("ABC"),
}

func TestVersion(t *testing.T) {
	badvers := 0
Version:
	for v := Version(1); v <= 40; v++ {
		for l := L; l <= H; l++ {
			for _, in := range input {
				if !test(t, v, l, in) {
					if badvers++; badvers >= 10 {
						t.Errorf("stopping after %d bad versions", badvers)
						break Version
					}
				}
			}
		}
	}
}

func TestEncode(t *testing.T) {
	data := []byte{0x10, 0x20, 0x0c, 0x56, 0x61, 0x80, 0xec, 0x11, 0xec, 0x11, 0xec, 0x11, 0xec, 0x11, 0xec, 0x11}
	check := []byte{0xa5, 0x24, 0xd4, 0xc1, 0xed, 0x36, 0xc7, 0x87, 0x2c, 0x55}
	rs := gf256.NewRSEncoder(Field, len(check))
	out := make([]byte, len(check))
	rs.ECC(data, out)
	if !bytes.Equal(out, check) {
		t.Errorf("have %x want %x", out, check)
	}
}

func TestPenalty(t *testing.T) {
	grid := &Code{
		// runs of 6, 8, 6    -> 14
		// 11 2x2 blocks      -> 33
		// 2 penalty patterns -> 80
		// 61% black          -> 20
		Bitmap: []byte{
			0xe0, // 11100000
			0xe7, // 11100111
			0x5c, // 01011100
			0x79, // 01111001
			0x5d, // 01011101
			0xfd, // 11111101
			0xe4, // 11100100
			0xe9, // 11101001
		},
		Size:   8,
		Stride: 1,
	}
	const (
		adjacentPenalty   = 14
		blockPenalty      = 33
		patternPenalty    = 80
		proportionPenalty = 20
	)

	if got := grid.adjacentPenalty(); got != adjacentPenalty {
		t.Errorf("have adjacent penalty %d want %d", got, adjacentPenalty)
	}
	if got := grid.blockPenalty(); got != blockPenalty {
		t.Errorf("have block penalty %d want %d", got, blockPenalty)
	}
	if got := grid.patternPenalty(); got != patternPenalty {
		t.Errorf("have pattern penalty %d want %d", got, patternPenalty)
	}
	if got := grid.proportionPenalty(); got != proportionPenalty {
		t.Errorf("have proportion penalty %d want %d", got, proportionPenalty)
	}
}

// TestLplanPlacement verifies the arithmetic bit placement against a
// reference that materializes the interleaved sequence the way lplan
// originally did, across every version and level.
func TestLplanPlacement(t *testing.T) {
	for v := Version(MinVersion); v <= MaxVersion; v++ {
		for _, l := range []Level{L, M, Q, H} {
			got, err := vplan(v)
			if err != nil {
				t.Fatal(err)
			}
			if err := fplan(l, 0, got); err != nil {
				t.Fatal(err)
			}
			if err := lplan(v, l, got); err != nil {
				t.Fatal(err)
			}

			want, err := vplan(v)
			if err != nil {
				t.Fatal(err)
			}
			if err := fplan(l, 0, want); err != nil {
				t.Fatal(err)
			}
			refLplan(t, v, l, want)

			for y := range want.Pixel {
				for x := range want.Pixel[y] {
					if got.Pixel[y][x] != want.Pixel[y][x] {
						t.Fatalf("version %d level %d: pixel (%d,%d) = %v, want %v",
							v, l, x, y, got.Pixel[y][x], want.Pixel[y][x])
					}
				}
			}
		}
	}
}

// refLplan is the pre-arithmetic lplan: materialize the data and
// checksum pixels, split into blocks, interleave, and sweep.
func refLplan(t *testing.T, v Version, l Level, p *Plan) {
	t.Helper()
	nblock := vtab[v].level[l].nblock
	ne := vtab[v].level[l].check
	nde := (vtab[v].bytes - ne*nblock) / nblock
	extra := (vtab[v].bytes - ne*nblock) % nblock
	dataBits := (nde*nblock + extra) * 8
	checkBits := ne * nblock * 8
	p.Level = l
	p.DataBytes = vtab[v].bytes - ne*nblock
	p.CheckBytes = ne * nblock
	p.Blocks = nblock

	data := make([]Pixel, dataBits)
	for i := range data {
		data[i] = Data.Pixel() | OffsetPixel(uint(i))
	}
	check := make([]Pixel, checkBits)
	for i := range check {
		check[i] = Check.Pixel() | OffsetPixel(uint(i+dataBits))
	}
	dataList := make([][]Pixel, nblock)
	checkList := make([][]Pixel, nblock)
	for i := 0; i < nblock; i++ {
		nd := nde
		if i >= nblock-extra {
			nd++
		}
		dataList[i], data = data[0:nd*8], data[nd*8:]
		checkList[i], check = check[0:ne*8], check[ne*8:]
	}
	bits := make([]Pixel, 0, dataBits+checkBits+7)
	for i := 0; i < nde+1; i++ {
		for _, b := range dataList {
			if i*8 < len(b) {
				bits = append(bits, b[i*8:(i+1)*8]...)
			}
		}
	}
	for i := 0; i < ne; i++ {
		for _, b := range checkList {
			if i*8 < len(b) {
				bits = append(bits, b[i*8:(i+1)*8]...)
			}
		}
	}
	for i := 0; i < 7; i++ {
		bits = append(bits, Extra.Pixel())
	}
	src := bits
	siz := len(p.Pixel)
	for x := siz; x > 0; {
		for y := siz - 1; y >= 0; y-- {
			if p.Pixel[y][x-1].Role() == 0 {
				p.Pixel[y][x-1], src = src[0], src[1:]
			}
			if p.Pixel[y][x-2].Role() == 0 {
				p.Pixel[y][x-2], src = src[0], src[1:]
			}
		}
		x -= 2
		if x == 7 {
			x--
		}
		for y := 0; y < siz; y++ {
			if p.Pixel[y][x-1].Role() == 0 {
				p.Pixel[y][x-1], src = src[0], src[1:]
			}
			if p.Pixel[y][x-2].Role() == 0 {
				p.Pixel[y][x-2], src = src[0], src[1:]
			}
		}
		x -= 2
	}
}
