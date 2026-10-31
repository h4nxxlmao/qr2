// SPDX-License-Identifier: GPL-3.0-only

package qr

import "fmt"

type bitBuffer struct {
	bits []bool
}

func (b *bitBuffer) Len() int { return len(b.bits) }

func (b *bitBuffer) AppendBit(v bool) {
	b.bits = append(b.bits, v)
}

// Append writes the n low-order bits of val, most significant first.
func (b *bitBuffer) Append(val uint32, n int) {
	if n < 0 || n > 32 {
		panic(fmt.Sprintf("qr: invalid bit count %d", n))
	}
	if n < 32 && val>>uint(n) != 0 {
		panic(fmt.Sprintf("qr: value %d does not fit in %d bits", val, n))
	}
	for i := n - 1; i >= 0; i-- {
		b.bits = append(b.bits, (val>>uint(i))&1 != 0)
	}
}

func (b *bitBuffer) AppendZeroes(n int) {
	for i := 0; i < n; i++ {
		b.bits = append(b.bits, false)
	}
}

func (b *bitBuffer) AppendBuffer(other *bitBuffer) {
	b.bits = append(b.bits, other.bits...)
}

func (b *bitBuffer) Bytes() []byte {
	out := make([]byte, (len(b.bits)+7)/8)
	for i, bit := range b.bits {
		if bit {
			out[i>>3] |= 1 << uint(7-i&7)
		}
	}
	return out
}

func bitAt(v uint32, i int) bool { return (v>>uint(i))&1 != 0 }
