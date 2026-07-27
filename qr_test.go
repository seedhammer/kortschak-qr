// Copyright 2021 Dan Kortschak.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qr

import (
	"hash/crc32"
	"testing"
)

func TestMasks(t *testing.T) {
	tests := []struct {
		content string
		crc     uint32
	}{
		// Bitmap CRCs for the packed (size+7)/8 stride; the module
		// grids are unchanged from the padded-stride layout, verified
		// pixel-by-pixel via Black() when the stride was fixed.
		{"1", 0xfbccb55f},   // mask 0.
		{"db", 0x549b6418},  // mask 1.
		{"0", 0xd331dc1b},   // mask 2.
		{"7b9", 0xb1ecc198}, // mask 3.
		{"3", 0xd8e372e6},   // mask 4.
		{"16", 0x30fdb989},  // mask 5.
		{"35", 0x72f3c395},  // mask 6.
		{"c", 0xe22ac82e},   // mask 7.
	}
	for _, test := range tests {
		qr, err := Encode(test.content, L)
		if err != nil {
			t.Fatal(err)
		}
		crc := crc32.NewIEEE()
		if _, err := crc.Write(qr.Bitmap); err != nil {
			t.Error(err)
		}
		if got := crc.Sum32(); got != test.crc {
			t.Errorf("checksum for %q is %x, want %x", test.content, got, test.crc)
		}
	}
}

func TestMinSize(t *testing.T) {
	// MinSize must agree with the size Encode actually produces,
	// across version boundaries and encoding classes.
	texts := []string{
		"1234567890",
		"HELLO WORLD 123",
		"wsh(sortedmulti(2,xpub,xpub))",
	}
	for n := 8; n <= 2048; n *= 2 {
		texts = append(texts, string(make([]byte, n)))
	}
	for _, text := range texts {
		if text == "" {
			continue
		}
		for _, level := range []Level{L, M, Q, H} {
			want, err := Encode(text, level)
			if err != nil {
				continue
			}
			got, err := MinSize(text, level)
			if err != nil {
				t.Errorf("MinSize(%d bytes, %v): %v", len(text), level, err)
				continue
			}
			if got != want.Size {
				t.Errorf("MinSize(%d bytes, %v) = %d, Encode size %d", len(text), level, got, want.Size)
			}
		}
	}
}

func TestStride(t *testing.T) {
	code, err := Encode("STRIDE CHECK", L)
	if err != nil {
		t.Fatal(err)
	}
	if want := (code.Size + 7) / 8; code.Stride != want {
		t.Errorf("Stride = %d, want %d for size %d", code.Stride, want, code.Size)
	}
	if want := code.Stride * code.Size; len(code.Bitmap) != want {
		t.Errorf("len(Bitmap) = %d, want %d", len(code.Bitmap), want)
	}
}
