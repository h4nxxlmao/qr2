package qr

// rsGenerator returns the generator polynomial coefficients for degree error
// correction codewords, highest power first and excluding the leading 1.
func rsGenerator(degree int) []byte {
	if degree < 1 || degree > 255 {
		panic("qr: invalid Reed-Solomon degree")
	}
	result := make([]byte, degree)
	result[degree-1] = 1
	root := byte(1)
	for i := 0; i < degree; i++ {
		for j := 0; j < degree; j++ {
			result[j] = gfMul(result[j], root)
			if j+1 < degree {
				result[j] ^= result[j+1]
			}
		}
		root = gfMul(root, 2)
	}
	return result
}

// rsRemainder computes the error correction codewords for data.
func rsRemainder(data, generator []byte) []byte {
	result := make([]byte, len(generator))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i, coef := range generator {
			result[i] ^= gfMul(coef, factor)
		}
	}
	return result
}

// rsSyndromes evaluates a data+ECC block at alpha^0 .. alpha^(ecLen-1). The
// QR generator polynomial has those roots, so a valid codeword has all-zero
// syndromes.
func rsSyndromes(block []byte, ecLen int) []byte {
	synd := make([]byte, ecLen)
	for i := 0; i < ecLen; i++ {
		x := gfExp[i]
		var sum byte
		for _, c := range block {
			sum = gfMul(sum, x) ^ c
		}
		synd[i] = sum
	}
	return synd
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// interleaveBlocks appends ECC to each block and interleaves data and ECC
// codewords in the order required by ISO/IEC 18004 clause 7.6.
func interleaveBlocks(data []byte, version int, ecc ECC) []byte {
	layout := layoutFor(version, ecc)
	blocks := make([][]byte, layout.NumBlocks)
	eccs := make([][]byte, layout.NumBlocks)
	gen := rsGenerator(layout.ECPerBlock)
	offset := 0
	for i := 0; i < layout.NumBlocks; i++ {
		n := layout.ShortData
		if i >= layout.NumShort {
			n = layout.LongData
		}
		block := make([]byte, n)
		copy(block, data[offset:offset+n])
		offset += n
		blocks[i] = block
		eccs[i] = rsRemainder(block, gen)
	}
	if offset != len(data) {
		panic("qr: data length does not match block layout")
	}

	out := make([]byte, 0, layout.RawCodewords)
	for i := 0; i < layout.LongData; i++ {
		for b := 0; b < layout.NumBlocks; b++ {
			if i < len(blocks[b]) {
				out = append(out, blocks[b][i])
			}
		}
	}
	for i := 0; i < layout.ECPerBlock; i++ {
		for b := 0; b < layout.NumBlocks; b++ {
			out = append(out, eccs[b][i])
		}
	}
	if len(out) != layout.RawCodewords {
		panic("qr: interleaved codeword count mismatch")
	}
	return out
}

// deinterleaveBlocks is the inverse of interleaveBlocks, used by SelfCheck.
func deinterleaveBlocks(raw []byte, version int, ecc ECC) (data []byte, blocks [][]byte, err *Error) {
	layout := layoutFor(version, ecc)
	if len(raw) != layout.RawCodewords {
		return nil, nil, errf("internal", "This is an engine bug. Please report it.", "raw codeword count mismatch")
	}
	blocks = make([][]byte, layout.NumBlocks)
	dataLens := make([]int, layout.NumBlocks)
	for i := range blocks {
		n := layout.ShortData
		if i >= layout.NumShort {
			n = layout.LongData
		}
		dataLens[i] = n
		blocks[i] = make([]byte, 0, n+layout.ECPerBlock)
	}
	pos := 0
	for i := 0; i < layout.LongData; i++ {
		for b := 0; b < layout.NumBlocks; b++ {
			if i < dataLens[b] {
				blocks[b] = append(blocks[b], raw[pos])
				pos++
			}
		}
	}
	for i := 0; i < layout.ECPerBlock; i++ {
		for b := 0; b < layout.NumBlocks; b++ {
			blocks[b] = append(blocks[b], raw[pos])
			pos++
		}
	}
	if pos != len(raw) {
		return nil, nil, errf("internal", "This is an engine bug. Please report it.", "deinterleave did not consume all codewords")
	}
	for i, block := range blocks {
		data = append(data, block[:dataLens[i]]...)
	}
	return data, blocks, nil
}
