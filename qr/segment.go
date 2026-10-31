// SPDX-License-Identifier: GPL-3.0-only

package qr

import (
	"strings"
	"unicode/utf8"
)

// Mode is a QR data mode.
type Mode uint8

const (
	modeNumeric Mode = iota
	modeAlnum
	modeByte
	modeKanji
	modeECI
)

func (m Mode) String() string {
	switch m {
	case modeNumeric:
		return "numeric"
	case modeAlnum:
		return "alphanumeric"
	case modeByte:
		return "byte"
	case modeKanji:
		return "kanji"
	case modeECI:
		return "eci"
	default:
		return "unknown"
	}
}

func (m Mode) indicator() uint32 {
	switch m {
	case modeNumeric:
		return 0x1
	case modeAlnum:
		return 0x2
	case modeByte:
		return 0x4
	case modeKanji:
		return 0x8
	case modeECI:
		return 0x7
	default:
		return 0
	}
}

const alnumChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

func alnumValue(r rune) (uint8, bool) {
	if r < 0 || r > 127 {
		return 0, false
	}
	i := strings.IndexByte(alnumChars, byte(r))
	if i < 0 {
		return 0, false
	}
	return uint8(i), true
}

func isNumericRune(r rune) bool { return r >= '0' && r <= '9' }

// unit is one input unit: a rune for text or a byte for binary data.
type unit struct {
	r     rune
	bytes []byte
	sjis  uint16
	kanji bool
}

func (u unit) numeric() bool { return isNumericRune(u.r) }
func (u unit) alnum() bool {
	_, ok := alnumValue(u.r)
	return ok
}

type segment struct {
	mode    Mode
	units   []unit
	count   int // characters for numeric/alnum/kanji, bytes for byte mode
	byteLen int
}

func (s segment) payloadBits(version int) int {
	switch s.mode {
	case modeNumeric:
		full := s.count / 3
		rem := s.count % 3
		bits := full*10 + rem*3 + boolToInt(rem > 0)
		return bits
	case modeAlnum:
		full := s.count / 2
		rem := s.count % 2
		return full*11 + rem*6
	case modeByte:
		return s.byteLen * 8
	case modeKanji:
		return s.count * 13
	default:
		return 0
	}
}

func (s segment) totalBits(version int) int {
	return 4 + charCountBits(s.mode, version) + s.payloadBits(version)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// unitsForRequest converts text or binary input into segmentation units.
func unitsForRequest(req EncodeRequest) ([]unit, *Error) {
	if !req.IsText {
		if len(req.Data) > 0 && req.Text != "" {
			return nil, errInvalidRequest("provide either Data or Text, not both")
		}
		units := make([]unit, len(req.Data))
		for i, b := range req.Data {
			units[i] = unit{r: rune(b), bytes: []byte{b}}
		}
		return units, nil
	}

	if !utf8.ValidString(req.Text) {
		return nil, errInvalidRequest("text is not valid UTF-8; send it as bytes instead")
	}

	switch req.Charset {
	case CharsetLatin1:
		return latin1Units(req.Text)
	case CharsetAuto, CharsetUTF8:
	default:
		return nil, errUnsupported("unknown charset")
	}

	if req.ECI == 3 {
		return latin1Units(req.Text)
	}
	if req.ECI != 0 && req.ECI != 26 {
		return nil, errUnsupported("only UTF-8, ISO-8859-1, and explicit ECI 26 are implemented")
	}

	var units []unit
	for _, r := range req.Text {
		u := unit{r: r, bytes: []byte(string(r))}
		if code, ok := toKanji(r); ok {
			u.kanji = true
			u.sjis = code
		}
		units = append(units, u)
	}
	return units, nil
}

func latin1Units(text string) ([]unit, *Error) {
	units := make([]unit, 0, len(text))
	for _, r := range text {
		if r > 0xFF {
			return nil, errInvalidRequest("character cannot be represented in ISO-8859-1")
		}
		units = append(units, unit{r: r, bytes: []byte{byte(r)}})
	}
	return units, nil
}

// toKanji is replaced with a generated JIS X 0208 table in a later phase.
// Keeping the hook here means the optimizer already understands the mode.
var toKanji = func(r rune) (uint16, bool) { return 0, false }

func canEncode(m Mode, u unit) bool {
	switch m {
	case modeNumeric:
		return u.numeric()
	case modeAlnum:
		return u.alnum()
	case modeByte:
		return true
	case modeKanji:
		return u.kanji
	default:
		return false
	}
}

func modePayloadInc(m Mode, mod int, u unit) int {
	switch m {
	case modeNumeric:
		switch mod {
		case 0:
			return 4
		case 1:
			return 3
		default:
			return 3
		}
	case modeAlnum:
		if mod == 0 {
			return 6
		}
		return 5
	case modeByte:
		return 8 * len(u.bytes)
	case modeKanji:
		return 13
	default:
		return 0
	}
}

func nextMod(m Mode, mod int) int {
	switch m {
	case modeNumeric:
		return (mod + 1) % 3
	case modeAlnum:
		return (mod + 1) % 2
	default:
		return 0
	}
}

type dpState struct {
	cost     int
	prevMode Mode
	prevMod  int
	started  bool
	ok       bool
}

const inf = int(^uint(0) >> 1)

// optimizeSegments finds the minimal-bit mixed-mode segmentation for a
// version. The cost includes every segment header. It returns ok=false when a
// segment's character count does not fit its count field.
func optimizeSegments(units []unit, version int) ([]segment, int, bool) {
	n := len(units)
	if n == 0 {
		return nil, 0, true
	}
	states := make([][4][3]dpState, n)
	for i := range states {
		for m := 0; m < 4; m++ {
			for r := 0; r < 3; r++ {
				states[i][m][r].cost = inf
			}
		}
	}

	for m := Mode(0); m < 4; m++ {
		if !canEncode(m, units[0]) {
			continue
		}
		mod := nextMod(m, 0)
		st := &states[0][m][mod]
		st.cost = 4 + charCountBits(m, version) + modePayloadInc(m, 0, units[0])
		st.prevMode = 255
		st.prevMod = 0
		st.started = true
		st.ok = true
	}

	for pos := 1; pos < n; pos++ {
		for m := Mode(0); m < 4; m++ {
			for mod := 0; mod < 3; mod++ {
				prev := states[pos-1][m][mod]
				if !prev.ok {
					continue
				}
				for next := Mode(0); next < 4; next++ {
					if !canEncode(next, units[pos]) {
						continue
					}
					started := next != m
					add := modePayloadInc(next, func() int {
						if started {
							return 0
						}
						return mod
					}(), units[pos])
					if started {
						add += 4 + charCountBits(next, version)
					}
					nextMod := nextModOf(next, mod, started)
					newCost := prev.cost + add
					cur := &states[pos][next][nextMod]
					if !cur.ok || newCost < cur.cost || (newCost == cur.cost && !started && cur.started) {
						cur.cost = newCost
						cur.prevMode = m
						cur.prevMod = mod
						cur.started = started
						cur.ok = true
					}
				}
			}
		}
	}

	bestMode, bestMod, bestCost := Mode(255), 0, inf
	for m := Mode(0); m < 4; m++ {
		for mod := 0; mod < 3; mod++ {
			st := states[n-1][m][mod]
			if st.ok && st.cost < bestCost {
				bestMode, bestMod, bestCost = m, mod, st.cost
			}
		}
	}
	if bestMode == 255 {
		return nil, 0, false
	}

	modes := make([]Mode, n)
	starts := make([]bool, n)
	cm, cr := bestMode, bestMod
	for i := n - 1; i >= 0; i-- {
		st := states[i][cm][cr]
		modes[i] = cm
		starts[i] = st.started
		if i == 0 {
			break
		}
		cm, cr = st.prevMode, st.prevMod
	}

	var segs []segment
	for i := 0; i < n; {
		if !starts[i] {
			return nil, 0, false
		}
		m := modes[i]
		seg := segment{mode: m}
		j := i
		for j < n && modes[j] == m {
			seg.units = append(seg.units, units[j])
			j++
		}
		switch m {
		case modeNumeric, modeAlnum, modeKanji:
			seg.count = len(seg.units)
		case modeByte:
			for _, u := range seg.units {
				seg.byteLen += len(u.bytes)
			}
			seg.count = seg.byteLen
		}
		ccbits := charCountBits(m, version)
		if seg.count >= 1<<uint(ccbits) {
			return nil, 0, false
		}
		segs = append(segs, seg)
		i = j
	}
	return segs, bestCost, true
}

func nextModOf(m Mode, oldMod int, started bool) int {
	if started {
		return nextMod(m, 0)
	}
	return nextMod(m, oldMod)
}

// appendSegments writes mode headers and payload bits.
func appendSegments(bb *bitBuffer, segs []segment, version int) {
	for _, seg := range segs {
		bb.Append(seg.mode.indicator(), 4)
		bb.Append(uint32(seg.count), charCountBits(seg.mode, version))
		switch seg.mode {
		case modeNumeric:
			for i := 0; i < len(seg.units); i += 3 {
				n := len(seg.units) - i
				if n > 3 {
					n = 3
				}
				var v uint32
				for j := 0; j < n; j++ {
					v = v*10 + uint32(seg.units[i+j].r-'0')
				}
				bb.Append(v, n*3+1)
			}
		case modeAlnum:
			for i := 0; i < len(seg.units); i += 2 {
				a, _ := alnumValue(seg.units[i].r)
				if i+1 < len(seg.units) {
					b, _ := alnumValue(seg.units[i+1].r)
					bb.Append(uint32(a)*45+uint32(b), 11)
				} else {
					bb.Append(uint32(a), 6)
				}
			}
		case modeByte:
			for _, u := range seg.units {
				for _, b := range u.bytes {
					bb.Append(uint32(b), 8)
				}
			}
		case modeKanji:
			for _, u := range seg.units {
				v := uint32(u.sjis)
				if v >= 0x8140 && v <= 0x9FFC {
					v -= 0x8140
				} else {
					v -= 0xC140
				}
				encoded := (v>>8)*0xC0 + (v & 0xFF)
				bb.Append(encoded, 13)
			}
		}
	}
}
