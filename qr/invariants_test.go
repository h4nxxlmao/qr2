// SPDX-License-Identifier: GPL-3.0-only

package qr

import "testing"

func TestFunctionModuleCount(t *testing.T) {
	for version := 1; version <= 40; version++ {
		b := newMatrixBuilder(version)
		b.drawFunctionPatterns(L)
		dataModules := 0
		for _, isFunction := range b.function {
			if !isFunction {
				dataModules++
			}
		}
		if want := numRawDataModules(version); dataModules != want {
			t.Errorf("v%d: %d data modules, want %d", version, dataModules, want)
		}
	}
}

func TestZigzagCoverage(t *testing.T) {
	for version := 1; version <= 40; version++ {
		b := newMatrixBuilder(version)
		b.drawFunctionPatterns(L)
		visited := 0
		for right := b.size - 1; right > 0; right -= 2 {
			r := right
			if r <= 6 {
				r--
			}
			for vert := 0; vert < b.size; vert++ {
				for j := 0; j < 2; j++ {
					x := r - j
					upward := (r+1)&2 == 0
					y := vert
					if upward {
						y = b.size - 1 - vert
					}
					if !b.function[y*b.size+x] {
						visited++
					}
				}
			}
		}
		if want := numRawDataModules(version); visited != want {
			t.Errorf("v%d: zigzag visited %d data modules, want %d", version, visited, want)
		}
	}
}

func TestSegmentBitAccounting(t *testing.T) {
	units, err := unitsForRequest(EncodeRequest{Text: "ORDER 12345678901234567890 abcdef", IsText: true})
	if err != nil {
		t.Fatal(err)
	}
	segs, bits, ok := optimizeSegments(units, 2)
	if !ok {
		t.Fatal("optimization failed")
	}
	sum := 0
	for _, seg := range segs {
		sum += seg.totalBits(2)
	}
	if sum != bits {
		t.Fatalf("segment total %d, optimizer cost %d", sum, bits)
	}
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segs))
	}
	if segs[0].mode != modeAlnum || segs[1].mode != modeNumeric || segs[2].mode != modeByte {
		t.Fatalf("unexpected mode sequence: %v %v %v", segs[0].mode, segs[1].mode, segs[2].mode)
	}
}
