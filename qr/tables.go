// SPDX-License-Identifier: GPL-3.0-only

package qr

const (
	minVersion = 1
	maxVersion = 40
)

// Index 0 is padding. Rows are L, M, Q, H. Values verified against the
// version/block structure test and the ISO 18004 Annex D totals.
var eccCodewordsPerBlock = [4][41]int{
	{-1, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{-1, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28},
	{-1, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{-1, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
}

var numErrorCorrectionBlocks = [4][41]int{
	{-1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},
	{-1, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},
	{-1, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},
	{-1, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81},
}

// numRawDataModules returns the number of data modules including remainder
// bits, before error correction. Formula from ISO/IEC 18004 clause 7.4.10.
func numRawDataModules(version int) int {
	result := (16*version+128)*version + 64
	if version >= 2 {
		numAlign := version/7 + 2
		result -= (25*numAlign-10)*numAlign - 55
		if version >= 7 {
			result -= 36
		}
	}
	return result
}

func numRawCodewords(version int) int { return numRawDataModules(version) / 8 }

// alignmentPatternPositions returns the ascending row/column centers.
func alignmentPatternPositions(version int) []int {
	if version == 1 {
		return nil
	}
	size := version*4 + 17
	numAlign := version/7 + 2
	step := (version*8 + numAlign*3 + 5) / (numAlign*4 - 4) * 2
	result := make([]int, 0, numAlign)
	for i := 0; i <= numAlign-2; i++ {
		result = append(result, size-7-i*step)
	}
	result = append(result, 6)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// blockLayout describes the RS block split for one version/ECC pair.
type blockLayout struct {
	NumBlocks    int
	ECPerBlock   int
	ShortData    int
	LongData     int
	NumShort     int
	RawCodewords int
}

func layoutFor(version int, ecc ECC) blockLayout {
	raw := numRawCodewords(version)
	numBlocks := numErrorCorrectionBlocks[ecc][version]
	ecPerBlock := eccCodewordsPerBlock[ecc][version]
	shortBlockLen := raw / numBlocks
	numShort := numBlocks - raw%numBlocks
	shortData := shortBlockLen - ecPerBlock
	return blockLayout{
		NumBlocks:    numBlocks,
		ECPerBlock:   ecPerBlock,
		ShortData:    shortData,
		LongData:     shortData + 1,
		NumShort:     numShort,
		RawCodewords: raw,
	}
}

func numDataCodewords(version int, ecc ECC) int {
	l := layoutFor(version, ecc)
	return l.NumShort*l.ShortData + (l.NumBlocks-l.NumShort)*l.LongData
}

func dataCapacityBits(version int, ecc ECC) int {
	return numDataCodewords(version, ecc) * 8
}

// charCountBits returns the character count field width for a mode and version.
func charCountBits(mode Mode, version int) int {
	group := 0
	switch {
	case version >= 27:
		group = 2
	case version >= 10:
		group = 1
	}
	switch mode {
	case modeNumeric:
		return [3]int{10, 12, 14}[group]
	case modeAlnum:
		return [3]int{9, 11, 13}[group]
	case modeByte:
		return [3]int{8, 16, 16}[group]
	case modeKanji:
		return [3]int{8, 10, 12}[group]
	case modeECI:
		return 0
	default:
		return 0
	}
}
