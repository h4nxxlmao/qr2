// SPDX-License-Identifier: GPL-3.0-only

package qr

// matrixBuilder holds a QR module grid plus function-module and role maps.
type matrixBuilder struct {
	version  int
	size     int
	modules  []byte
	roles    []Role
	function []bool
}

func newMatrixBuilder(version int) *matrixBuilder {
	size := version*4 + 17
	return &matrixBuilder{
		version:  version,
		size:     size,
		modules:  make([]byte, size*size),
		roles:    make([]Role, size*size),
		function: make([]bool, size*size),
	}
}

func (m *matrixBuilder) idx(x, y int) int { return y*m.size + x }

func (m *matrixBuilder) inBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < m.size && y < m.size
}

func (m *matrixBuilder) setFunction(x, y int, dark bool, role Role) {
	if !m.inBounds(x, y) {
		return
	}
	i := m.idx(x, y)
	if dark {
		m.modules[i] = 1
	} else {
		m.modules[i] = 0
	}
	m.roles[i] = role
	m.function[i] = true
}

func (m *matrixBuilder) drawFunctionPatterns(ecc ECC) {
	// Timing patterns.
	for i := 0; i < m.size; i++ {
		m.setFunction(i, 6, i%2 == 0, RoleTiming)
		m.setFunction(6, i, i%2 == 0, RoleTiming)
	}

	// Finder patterns and separators.
	m.drawFinder(3, 3)
	m.drawFinder(m.size-4, 3)
	m.drawFinder(3, m.size-4)

	// Alignment patterns.
	pos := alignmentPatternPositions(m.version)
	last := len(pos) - 1
	for i := 0; i < len(pos); i++ {
		for j := 0; j < len(pos); j++ {
			if (i == 0 && (j == 0 || j == last)) || (i == last && j == 0) {
				continue
			}
			m.drawAlignment(pos[i], pos[j])
		}
	}

	m.drawFormatBits(0, ecc)
	m.drawVersionInfo()
}

func (m *matrixBuilder) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			d := max(abs(dx), abs(dy))
			dark := d != 2 && d != 4
			var role Role
			switch d {
			case 0, 1:
				role = RoleFinderInner
			case 2:
				role = RoleFinderWhite
			case 3:
				role = RoleFinderOuter
			default:
				role = RoleFinderWhite
			}
			m.setFunction(cx+dx, cy+dy, dark, role)
		}
	}
}

func (m *matrixBuilder) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			dark := max(abs(dx), abs(dy)) != 1
			m.setFunction(cx+dx, cy+dy, dark, RoleAlignment)
		}
	}
}

func formatBits(ecc ECC, mask int) int {
	data := ecc.FormatBits()<<3 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	return (data<<10 | rem) ^ 0x5412
}

func (m *matrixBuilder) drawFormatBits(mask int, ecc ECC) {
	bits := formatBits(ecc, mask)
	for i := 0; i < 6; i++ {
		m.setFunction(8, i, bitAt(uint32(bits), i), RoleFormat)
	}
	m.setFunction(8, 7, bitAt(uint32(bits), 6), RoleFormat)
	m.setFunction(8, 8, bitAt(uint32(bits), 7), RoleFormat)
	m.setFunction(7, 8, bitAt(uint32(bits), 8), RoleFormat)
	for i := 9; i < 15; i++ {
		m.setFunction(14-i, 8, bitAt(uint32(bits), i), RoleFormat)
	}

	for i := 0; i < 8; i++ {
		m.setFunction(m.size-1-i, 8, bitAt(uint32(bits), i), RoleFormat)
	}
	for i := 8; i < 15; i++ {
		m.setFunction(8, m.size-15+i, bitAt(uint32(bits), i), RoleFormat)
	}
	m.setFunction(8, m.size-8, true, RoleDark)
}

func versionBits(version int) int {
	rem := version
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
	}
	return version<<12 | rem
}

func (m *matrixBuilder) drawVersionInfo() {
	if m.version < 7 {
		return
	}
	bits := versionBits(m.version)
	for i := 0; i < 18; i++ {
		bit := bitAt(uint32(bits), i)
		a := m.size - 11 + i%3
		b := i / 3
		m.setFunction(a, b, bit, RoleVersion)
		m.setFunction(b, a, bit, RoleVersion)
	}
}

func (m *matrixBuilder) drawCodewords(data []byte) {
	i := 0
	for right := m.size - 1; right > 0; right -= 2 {
		r := right
		if r <= 6 {
			r--
		}
		for vert := 0; vert < m.size; vert++ {
			for j := 0; j < 2; j++ {
				x := r - j
				upward := (r+1)&2 == 0
				y := vert
				if upward {
					y = m.size - 1 - vert
				}
				if !m.function[m.idx(x, y)] && i < len(data)*8 {
					bit := (data[i>>3] >> uint(7-i&7)) & 1
					m.modules[m.idx(x, y)] = bit
					m.roles[m.idx(x, y)] = RoleData
					i++
				}
			}
		}
	}
}

func maskBit(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (x/3+y/2)%2 == 0
	case 5:
		return (x*y)%2+(x*y)%3 == 0
	case 6:
		return ((x*y)%2+(x*y)%3)%2 == 0
	case 7:
		return ((x+y)%2+(x*y)%3)%2 == 0
	default:
		return false
	}
}

func applyMask(modules []byte, function []bool, mask, size int) {
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := y*size + x
			if !function[i] && maskBit(mask, x, y) {
				modules[i] ^= 1
			}
		}
	}
}

func drawFormatBitsInto(modules []byte, size int, mask int, ecc ECC) {
	bits := formatBits(ecc, mask)
	set := func(x, y int, v bool) {
		if x < 0 || y < 0 || x >= size || y >= size {
			return
		}
		i := y*size + x
		if v {
			modules[i] = 1
		} else {
			modules[i] = 0
		}
	}
	for i := 0; i < 6; i++ {
		set(8, i, bitAt(uint32(bits), i))
	}
	set(8, 7, bitAt(uint32(bits), 6))
	set(8, 8, bitAt(uint32(bits), 7))
	set(7, 8, bitAt(uint32(bits), 8))
	for i := 9; i < 15; i++ {
		set(14-i, 8, bitAt(uint32(bits), i))
	}
	for i := 0; i < 8; i++ {
		set(size-1-i, 8, bitAt(uint32(bits), i))
	}
	for i := 8; i < 15; i++ {
		set(8, size-15+i, bitAt(uint32(bits), i))
	}
	set(8, size-8, true)
}

func penaltyScore(modules []byte, size int) int {
	score := 0

	// N1: runs of five or more same-colored modules in a row or column.
	for y := 0; y < size; y++ {
		run := 1
		for x := 1; x < size; x++ {
			if modules[y*size+x] == modules[y*size+x-1] {
				run++
			} else {
				if run >= 5 {
					score += 3 + run - 5
				}
				run = 1
			}
		}
		if run >= 5 {
			score += 3 + run - 5
		}
	}
	for x := 0; x < size; x++ {
		run := 1
		for y := 1; y < size; y++ {
			if modules[y*size+x] == modules[(y-1)*size+x] {
				run++
			} else {
				if run >= 5 {
					score += 3 + run - 5
				}
				run = 1
			}
		}
		if run >= 5 {
			score += 3 + run - 5
		}
	}

	// N2: 2x2 same-colored blocks.
	for y := 0; y < size-1; y++ {
		for x := 0; x < size-1; x++ {
			a := modules[y*size+x]
			if a == modules[y*size+x+1] && a == modules[(y+1)*size+x] && a == modules[(y+1)*size+x+1] {
				score += 3
			}
		}
	}

	// N3: finder-like 1:1:3:1:1 patterns with four light modules on a side.
	p1 := [11]byte{1, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0}
	p2 := [11]byte{0, 0, 0, 0, 1, 0, 1, 1, 1, 0, 1}
	checkLine := func(get func(int) byte) {
		for start := 0; start+10 < size; start++ {
			match1, match2 := true, true
			for k := 0; k < 11; k++ {
				v := get(start + k)
				if v != p1[k] {
					match1 = false
				}
				if v != p2[k] {
					match2 = false
				}
			}
			if match1 {
				score += 40
			}
			if match2 {
				score += 40
			}
		}
	}
	for y := 0; y < size; y++ {
		row := y
		checkLine(func(x int) byte { return modules[row*size+x] })
	}
	for x := 0; x < size; x++ {
		col := x
		checkLine(func(y int) byte { return modules[y*size+col] })
	}

	// N4: balance of dark and light modules.
	dark := 0
	for _, v := range modules {
		if v != 0 {
			dark++
		}
	}
	total := size * size
	deviation := abs(dark*100 - total*50)
	k := deviation / (total * 5)
	score += k * 10

	return score
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// buildMatrix creates the final matrix and chooses a mask unless one is forced.
func buildMatrix(version int, ecc ECC, codewords []byte, forcedMask int) ([]byte, []Role, int, *Error) {
	b := newMatrixBuilder(version)
	b.drawFunctionPatterns(ecc)
	b.drawCodewords(codewords)

	mask := forcedMask
	if mask < 0 {
		best := inf
		for m := 0; m < 8; m++ {
			candidate := append([]byte(nil), b.modules...)
			applyMask(candidate, b.function, m, b.size)
			drawFormatBitsInto(candidate, b.size, m, ecc)
			score := penaltyScore(candidate, b.size)
			if score < best {
				best = score
				mask = m
			}
		}
	}
	if mask < 0 || mask > 7 {
		return nil, nil, 0, errInvalidRequest("mask must be auto or 0 to 7")
	}
	applyMask(b.modules, b.function, mask, b.size)
	b.drawFormatBits(mask, ecc)
	return b.modules, b.roles, mask, nil
}
