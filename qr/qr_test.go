// SPDX-License-Identifier: GPL-3.0-only

package qr

import (
	"bytes"
	"testing"
)

func TestTables(t *testing.T) {
	knownData := map[int][4]int{
		1:  {19, 16, 13, 9},
		2:  {34, 28, 22, 16},
		10: {274, 216, 154, 122},
		40: {2956, 2334, 1666, 1276},
	}
	for v := 1; v <= 40; v++ {
		for e := L; e <= H; e++ {
			l := layoutFor(v, e)
			total := l.NumShort*l.ShortData + (l.NumBlocks-l.NumShort)*l.LongData + l.ECPerBlock*l.NumBlocks
			if total != l.RawCodewords {
				t.Fatalf("v%d-%s: block totals %d, raw %d", v, e, total, l.RawCodewords)
			}
			if want, ok := knownData[v]; ok {
				if got := numDataCodewords(v, e); got != want[e] {
					t.Fatalf("v%d-%s: data codewords %d, want %d", v, e, got, want[e])
				}
			}
		}
	}
}

func TestAlignmentPositions(t *testing.T) {
	cases := map[int][]int{
		1:  nil,
		2:  {6, 18},
		7:  {6, 22, 38},
		14: {6, 26, 46, 66},
		32: {6, 34, 60, 86, 112, 138},
		40: {6, 30, 58, 86, 114, 142, 170},
	}
	for v, want := range cases {
		got := alignmentPatternPositions(v)
		if len(got) != len(want) {
			t.Fatalf("v%d positions %v, want %v", v, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("v%d positions %v, want %v", v, got, want)
			}
		}
	}
}

func TestReedSolomonKnown(t *testing.T) {
	// Known QR example: version 1-M data codewords 32 91 11 120 209 114 220 77
	// 67 64 236 17 236 17 236 17 with 10 EC codewords.
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	gen := rsGenerator(10)
	ecc := rsRemainder(data, gen)
	want := []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}
	if !bytes.Equal(ecc, want) {
		t.Fatalf("ecc %v, want %v", ecc, want)
	}
	block := append(append([]byte{}, data...), ecc...)
	if !allZero(rsSyndromes(block, 10)) {
		t.Fatalf("syndromes not zero: %v", rsSyndromes(block, 10))
	}
	block[3] ^= 0xFF
	if allZero(rsSyndromes(block, 10)) {
		t.Fatal("expected non-zero syndromes after corruption")
	}
}

func TestRoundTrip(t *testing.T) {
	inputs := []string{
		"",
		"HELLO WORLD",
		"https://qr.hanx.pro",
		"0123456789",
		"12345678901234567890123456789012345678901234567890",
		"THE QUICK BROWN FOX JUMPS OVER THE LAZY DOG 0123456789 $%*+-./:",
		"multi\nline\ntext with UTF-8: Grüße, 日本語, 👋",
	}
	for _, text := range inputs {
		for _, ecc := range []ECC{L, M, Q, H} {
			req := EncodeRequest{Symbology: QRModel2, Text: text, IsText: true, ECC: ecc, Mask: -1, BoostECC: true}
			syms, err := Encode(req)
			if err != nil {
				t.Fatalf("%q ecc %s: %v", text, ecc, err)
			}
			dec, err := SelfCheck(syms[0])
			if err != nil {
				t.Fatalf("%q ecc %s selfcheck: %v", text, ecc, err)
			}
			if !bytes.Equal(dec.Data, []byte(text)) {
				t.Fatalf("%q ecc %s: decoded %q", text, ecc, dec.Data)
			}
		}
	}
}

func TestAllVersionsAndMasks(t *testing.T) {
	data := []byte("Version sweep 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for v := 1; v <= 40; v++ {
		for _, ecc := range []ECC{L, M, Q, H} {
			for mask := 0; mask < 8; mask++ {
				req := EncodeRequest{
					Symbology: QRModel2, Data: data,
					Version: v, ECC: ecc, Mask: mask,
				}
				syms, err := Encode(req)
				if err != nil {
					continue // data may not fit this version/ECC
				}
				dec, err := SelfCheck(syms[0])
				if err != nil {
					t.Fatalf("v%d-%s mask %d: %v", v, ecc, mask, err)
				}
				if !bytes.Equal(dec.Data, data) {
					t.Fatalf("v%d-%s mask %d: payload mismatch", v, ecc, mask)
				}
			}
		}
	}
}

func TestCapacityError(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 3000)
	_, err := Encode(EncodeRequest{Symbology: QRModel2, Data: data, ECC: H, Mask: -1})
	if err == nil {
		t.Fatal("expected capacity error")
	}
	if e, ok := err.(*Error); !ok || e.Code != "capacity" {
		t.Fatalf("expected capacity error, got %v", err)
	}
}

func TestMixModes(t *testing.T) {
	req := EncodeRequest{
		Symbology: QRModel2,
		Text:      "ORDER 12345678901234567890 abcdef",
		IsText:    true,
		ECC:       L,
		Mask:      -1,
	}
	syms, err := Encode(req)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := SelfCheck(syms[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec.Data, []byte(req.Text)) {
		t.Fatalf("decoded %q", dec.Data)
	}
	if len(dec.Segments) < 2 {
		t.Fatalf("expected mixed segments, got %+v", dec.Segments)
	}
}

func TestMaskOverride(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		syms, err := Encode(EncodeRequest{
			Symbology: QRModel2, Text: "MASK OVERRIDE", IsText: true, Version: 1, ECC: M, Mask: mask,
		})
		if err != nil {
			t.Fatal(err)
		}
		if syms[0].Meta.Mask != mask {
			t.Fatalf("mask %d reported as %d", mask, syms[0].Meta.Mask)
		}
	}
}
