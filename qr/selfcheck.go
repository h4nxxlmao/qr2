package qr

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Decoded is the result of SelfCheck.
type Decoded struct {
	Data     []byte
	Text     string
	ECI      int
	FNC1     FNC1Mode
	Segments []SegmentInfo
}

func selfCheckError(code, msg string) *Error {
	return errf(code, "The matrix did not decode as a valid symbol.", "%s", msg)
}

// SelfCheck decodes a symbol produced by Encode or a compatible encoder. It
// verifies the format/version information, unmasks the modules, checks every
// Reed-Solomon block, and parses the data segments. It does not perform image
// detection or error correction; that is handled by the UI decoder.
func SelfCheck(sym *Symbol) (*Decoded, error) {
	if sym == nil || sym.Width != sym.Height || sym.Width < 21 || sym.Width > 177 || (sym.Width-17)%4 != 0 {
		return nil, selfCheckError("invalid_matrix", "matrix dimensions are not a QR Model 2 size")
	}
	version := (sym.Width - 17) / 4
	size := sym.Width

	// Build the function map. The ECC passed here only affects the placeholder
	// format bits, which are function modules regardless of value.
	fb := newMatrixBuilder(version)
	fb.drawFunctionPatterns(L)

	ecc, mask, ok := readFormat(sym.Modules, size)
	if !ok {
		return nil, selfCheckError("format", "format information could not be decoded")
	}
	if version >= 7 {
		if !readVersionOK(sym.Modules, size, version) {
			return nil, selfCheckError("version", "version information does not match the matrix size")
		}
	}

	unmasked := append([]byte(nil), sym.Modules...)
	applyMask(unmasked, fb.function, mask, size)

	raw, rerr := readCodewords(unmasked, fb.function, size, numRawCodewords(version))
	if rerr != nil {
		return nil, rerr
	}
	data, blocks, derr := deinterleaveBlocks(raw, version, ecc)
	if derr != nil {
		return nil, derr
	}
	layout := layoutFor(version, ecc)
	for _, block := range blocks {
		if !allZero(rsSyndromes(block, layout.ECPerBlock)) {
			return nil, selfCheckError("reed_solomon", "a Reed-Solomon block has a non-zero syndrome")
		}
	}

	dec, perr := parseData(data, version)
	if perr != nil {
		return nil, perr
	}
	if utf8.Valid(dec.Data) {
		dec.Text = string(dec.Data)
	}
	return dec, nil
}

func readFormat(modules []byte, size int) (ECC, int, bool) {
	first := 0
	get := func(x, y int) int { return int(modules[y*size+x] & 1) }
	for i := 0; i < 6; i++ {
		first |= get(8, i) << i
	}
	first |= get(8, 7) << 6
	first |= get(8, 8) << 7
	first |= get(7, 8) << 8
	for i := 9; i < 15; i++ {
		first |= get(14-i, 8) << i
	}

	second := 0
	for i := 0; i < 8; i++ {
		second |= get(size-1-i, 8) << i
	}
	for i := 8; i < 15; i++ {
		second |= get(8, size-15+i) << i
	}

	if e, m, ok := decodeFormatWord(first); ok {
		return e, m, true
	}
	if e, m, ok := decodeFormatWord(second); ok {
		return e, m, true
	}
	return 0, 0, false
}

func decodeFormatWord(raw int) (ECC, int, bool) {
	bestDist := 16
	bestECC, bestMask := L, 0
	for e := L; e <= H; e++ {
		for mask := 0; mask < 8; mask++ {
			d := bitsDifference(raw, formatBits(e, mask))
			if d < bestDist {
				bestDist = d
				bestECC, bestMask = e, mask
			}
		}
	}
	if bestDist <= 3 {
		return bestECC, bestMask, true
	}
	return 0, 0, false
}

func bitsDifference(a, b int) int {
	v := uint32(a ^ b)
	n := 0
	for v != 0 {
		n += int(v & 1)
		v >>= 1
	}
	return n
}

func readVersionOK(modules []byte, size, version int) bool {
	get := func(x, y int) int { return int(modules[y*size+x] & 1) }
	first, second := 0, 0
	for i := 0; i < 18; i++ {
		a := size - 11 + i%3
		b := i / 3
		first |= get(a, b) << i
		second |= get(b, a) << i
	}
	want := versionBits(version)
	return first == want && second == want
}

func readCodewords(modules []byte, function []bool, size, rawCodewords int) ([]byte, *Error) {
	out := make([]byte, rawCodewords)
	i := 0
	for right := size - 1; right > 0; right -= 2 {
		r := right
		if r <= 6 {
			r--
		}
		for vert := 0; vert < size; vert++ {
			for j := 0; j < 2; j++ {
				x := r - j
				upward := (r+1)&2 == 0
				y := vert
				if upward {
					y = size - 1 - vert
				}
				idx := y*size + x
				if !function[idx] {
					if i < rawCodewords*8 {
						if modules[idx]&1 != 0 {
							out[i>>3] |= 1 << uint(7-i&7)
						}
						i++
					}
				}
			}
		}
	}
	if i != rawCodewords*8 {
		return nil, selfCheckError("codewords", fmt.Sprintf("matrix yielded %d data bits, expected %d", i, rawCodewords*8))
	}
	return out, nil
}

type bitReader struct {
	data []byte
	n    int
	pos  int
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data, n: len(data) * 8}
}

func (r *bitReader) read(n int) (uint32, bool) {
	if n < 0 || n > 32 || r.pos+n > r.n {
		return 0, false
	}
	var v uint32
	for i := 0; i < n; i++ {
		byteIdx := (r.pos + i) >> 3
		bitIdx := 7 - (r.pos+i)&7
		v = v<<1 | uint32((r.data[byteIdx]>>uint(bitIdx))&1)
	}
	r.pos += n
	return v, true
}

func parseData(data []byte, version int) (*Decoded, *Error) {
	br := newBitReader(data)
	dec := &Decoded{}
	for br.pos+4 <= br.n {
		modeVal, ok := br.read(4)
		if !ok {
			break
		}
		if modeVal == 0 {
			break
		}
		switch modeVal {
		case 1:
			if err := parseNumeric(br, version, dec); err != nil {
				return nil, err
			}
		case 2:
			if err := parseAlnum(br, version, dec); err != nil {
				return nil, err
			}
		case 4:
			if err := parseByte(br, version, dec); err != nil {
				return nil, err
			}
		case 8:
			if err := parseKanji(br, version, dec); err != nil {
				return nil, err
			}
		case 7:
			value, err := readECI(br)
			if err != nil {
				return nil, err
			}
			dec.ECI = value
			dec.Segments = append(dec.Segments, SegmentInfo{Mode: "eci", Count: 0, Bits: 0})
		case 5:
			dec.FNC1 = FNC1First
			dec.Segments = append(dec.Segments, SegmentInfo{Mode: "fnc1-first", Count: 0, Bits: 4})
		case 9:
			if _, ok := br.read(8); !ok {
				return nil, selfCheckError("fnc1", "truncated FNC1 application indicator")
			}
			dec.FNC1 = FNC1Second
			dec.Segments = append(dec.Segments, SegmentInfo{Mode: "fnc1-second", Count: 0, Bits: 12})
		case 3:
			if _, ok := br.read(16); !ok {
				return nil, selfCheckError("structured_append", "truncated structured append header")
			}
		default:
			return nil, selfCheckError("mode", "unknown segment mode")
		}
	}
	return dec, nil
}

func parseNumeric(br *bitReader, version int, dec *Decoded) *Error {
	count, ok := br.read(charCountBits(modeNumeric, version))
	if !ok {
		return selfCheckError("numeric", "truncated numeric count")
	}
	for left := int(count); left > 0; {
		n := left
		if n > 3 {
			n = 3
		}
		v, ok := br.read(n*3 + 1)
		if !ok {
			return selfCheckError("numeric", "truncated numeric payload")
		}
		digits := make([]byte, n)
		for i := n - 1; i >= 0; i-- {
			digits[i] = byte('0' + v%10)
			v /= 10
		}
		dec.Data = append(dec.Data, digits...)
		left -= n
	}
	dec.Segments = append(dec.Segments, SegmentInfo{Mode: "numeric", Count: int(count), Bits: 0})
	return nil
}

func parseAlnum(br *bitReader, version int, dec *Decoded) *Error {
	count, ok := br.read(charCountBits(modeAlnum, version))
	if !ok {
		return selfCheckError("alphanumeric", "truncated alphanumeric count")
	}
	for left := int(count); left > 0; {
		if left >= 2 {
			v, ok := br.read(11)
			if !ok {
				return selfCheckError("alphanumeric", "truncated alphanumeric payload")
			}
			if v/45 >= 45 || v%45 >= 45 {
				return selfCheckError("alphanumeric", "invalid alphanumeric value")
			}
			dec.Data = append(dec.Data, alnumChars[v/45], alnumChars[v%45])
			left -= 2
		} else {
			v, ok := br.read(6)
			if !ok {
				return selfCheckError("alphanumeric", "truncated alphanumeric payload")
			}
			if v >= 45 {
				return selfCheckError("alphanumeric", "invalid alphanumeric value")
			}
			dec.Data = append(dec.Data, alnumChars[v])
			left--
		}
	}
	dec.Segments = append(dec.Segments, SegmentInfo{Mode: "alphanumeric", Count: int(count), Bits: 0})
	return nil
}

func parseByte(br *bitReader, version int, dec *Decoded) *Error {
	count, ok := br.read(charCountBits(modeByte, version))
	if !ok {
		return selfCheckError("byte", "truncated byte count")
	}
	for i := 0; i < int(count); i++ {
		v, ok := br.read(8)
		if !ok {
			return selfCheckError("byte", "truncated byte payload")
		}
		dec.Data = append(dec.Data, byte(v))
	}
	dec.Segments = append(dec.Segments, SegmentInfo{Mode: "byte", Count: int(count), Bits: 0})
	return nil
}

func parseKanji(br *bitReader, version int, dec *Decoded) *Error {
	count, ok := br.read(charCountBits(modeKanji, version))
	if !ok {
		return selfCheckError("kanji", "truncated Kanji count")
	}
	for i := 0; i < int(count); i++ {
		v, ok := br.read(13)
		if !ok {
			return selfCheckError("kanji", "truncated Kanji payload")
		}
		sjis := (v/0xC0)<<8 | v%0xC0
		if sjis < 0x1F00 {
			sjis += 0x8140
		} else {
			sjis += 0xC140
		}
		r, ok := fromKanji(uint16(sjis))
		if !ok {
			return selfCheckError("kanji", "unknown Shift-JIS mapping")
		}
		dec.Data = append(dec.Data, []byte(string(r))...)
	}
	dec.Segments = append(dec.Segments, SegmentInfo{Mode: "kanji", Count: int(count), Bits: 0})
	return nil
}

func readECI(br *bitReader) (int, *Error) {
	first, ok := br.read(8)
	if !ok {
		return 0, selfCheckError("eci", "truncated ECI designator")
	}
	if first&0x80 == 0 {
		return int(first), nil
	}
	if first&0xC0 == 0x80 {
		second, ok := br.read(8)
		if !ok {
			return 0, selfCheckError("eci", "truncated ECI designator")
		}
		return int(first&0x3F)<<8 | int(second), nil
	}
	if first&0xE0 == 0xC0 {
		rest, ok := br.read(16)
		if !ok {
			return 0, selfCheckError("eci", "truncated ECI designator")
		}
		return int(first&0x1F)<<16 | int(rest), nil
	}
	return 0, selfCheckError("eci", "invalid ECI designator")
}

// fromKanji is replaced with a generated table in a later phase.
var fromKanji = func(code uint16) (rune, bool) { return 0, false }

// AlnumChars is exported for tests and the UI metadata.
const AlnumChars = alnumChars

// TrimPayload is a small helper used by tests and diagnostics.
func TrimPayload(s string) string { return strings.TrimRight(s, "\x00") }
