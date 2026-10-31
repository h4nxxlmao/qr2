package qr

// Encode produces one or more QR Model 2 symbols. Structured append is
// reserved for a later phase and currently returns a clear unsupported error.
func Encode(req EncodeRequest) ([]*Symbol, error) {
	if req.Symbology != QRModel2 {
		return nil, errUnsupported("only QR Model 2 is implemented in this build")
	}
	if req.StructuredAppend {
		return nil, errUnsupported("structured append is not implemented yet")
	}
	if req.Version < 0 || req.Version > maxVersion {
		return nil, errInvalidRequest("version must be auto or 1 to 40")
	}
	if req.Mask < -1 || req.Mask > 7 {
		return nil, errInvalidRequest("mask must be auto or 0 to 7")
	}
	if req.ECC > H {
		return nil, errInvalidRequest("unknown error correction level")
	}

	units, uerr := unitsForRequest(req)
	if uerr != nil {
		return nil, uerr
	}
	prefix, perr := buildPrefix(req)
	if perr != nil {
		return nil, perr
	}
	prefixBits := prefix.Len()

	start, end := 1, maxVersion
	if req.Version != 0 {
		start, end = req.Version, req.Version
	}

	var segs []segment
	var segBits int
	var segOK bool
	chosen := 0
	lastGroup := -1
	for v := start; v <= end; v++ {
		group := versionGroup(v)
		if group != lastGroup {
			segs, segBits, segOK = optimizeSegments(units, v)
			lastGroup = group
		}
		if !segOK {
			continue
		}
		if prefixBits+segBits <= dataCapacityBits(v, req.ECC) {
			chosen = v
			break
		}
	}
	if chosen == 0 {
		cap := dataCapacityBits(40, req.ECC)
		if req.Version != 0 {
			cap = dataCapacityBits(req.Version, req.ECC)
		}
		return nil, errCapacity(prefixBits+segBits, cap)
	}

	effectiveECC := req.ECC
	if req.BoostECC {
		for e := int(req.ECC) + 1; e <= int(H); e++ {
			if prefixBits+segBits <= dataCapacityBits(chosen, ECC(e)) {
				effectiveECC = ECC(e)
			}
		}
	}

	capacity := dataCapacityBits(chosen, effectiveECC)
	bb := &bitBuffer{}
	bb.AppendBuffer(prefix)
	appendSegments(bb, segs, chosen)
	if bb.Len() != prefixBits+segBits {
		return nil, errf("internal", "This is an engine bug. Please report it.", "segment bit count mismatch: got %d, want %d", bb.Len(), prefixBits+segBits)
	}
	if bb.Len() > capacity {
		return nil, errCapacity(bb.Len(), capacity)
	}

	// Terminator, byte alignment, and alternating pad codewords.
	terminator := capacity - bb.Len()
	if terminator > 4 {
		terminator = 4
	}
	bb.AppendZeroes(terminator)
	for bb.Len()%8 != 0 {
		bb.AppendZeroes(1)
	}
	data := bb.Bytes()
	pad := [2]byte{0xEC, 0x11}
	for len(data) < numDataCodewords(chosen, effectiveECC) {
		data = append(data, pad[len(data)&1])
	}
	if len(data) != numDataCodewords(chosen, effectiveECC) {
		return nil, errf("internal", "This is an engine bug. Please report it.", "data codeword count mismatch")
	}

	all := interleaveBlocks(data, chosen, effectiveECC)
	modules, roles, mask, merr := buildMatrix(chosen, effectiveECC, all, req.Mask)
	if merr != nil {
		return nil, merr
	}

	segInfos := make([]SegmentInfo, len(segs))
	for i, seg := range segs {
		segInfos[i] = SegmentInfo{Mode: seg.mode.String(), Count: seg.count, Bits: seg.totalBits(chosen)}
	}

	meta := Metadata{
		Symbology:       QRModel2,
		Version:         chosen,
		ECC:             effectiveECC.String(),
		Mask:            mask,
		Segments:        segInfos,
		BitLength:       prefixBits + segBits,
		CapacityBits:    capacity,
		CapacityPercent: float64(prefixBits+segBits) / float64(capacity) * 100,
		ECCBoosted:      effectiveECC != req.ECC,
		FNC1:            fnc1Name(req.FNC1),
		ECI:             req.ECI,
	}
	size := chosen*4 + 17
	return []*Symbol{{
		Width:   size,
		Height:  size,
		Modules: modules,
		Roles:   roles,
		Meta:    meta,
	}}, nil
}

func versionGroup(version int) int {
	switch {
	case version >= 27:
		return 2
	case version >= 10:
		return 1
	default:
		return 0
	}
}

func buildPrefix(req EncodeRequest) (*bitBuffer, *Error) {
	bb := &bitBuffer{}
	switch req.FNC1 {
	case FNC1First:
		bb.Append(0x5, 4)
	case FNC1Second:
		bb.Append(0x9, 4)
		bb.Append(uint32(req.FNC1App), 8)
	case FNC1None:
	default:
		return nil, errInvalidRequest("unknown FNC1 mode")
	}
	if req.ECI != 0 {
		bb.Append(0x7, 4)
		if e := appendECI(bb, req.ECI); e != nil {
			return nil, e
		}
	}
	return bb, nil
}

func appendECI(bb *bitBuffer, value int) *Error {
	switch {
	case value < 0:
		return errInvalidRequest("ECI value must be non-negative")
	case value < 1<<7:
		bb.Append(uint32(value), 8)
	case value < 1<<14:
		bb.Append(0b10, 2)
		bb.Append(uint32(value), 14)
	case value < 1000000:
		bb.Append(0b110, 3)
		bb.Append(uint32(value), 21)
	default:
		return errInvalidRequest("ECI value is too large")
	}
	return nil
}

func fnc1Name(m FNC1Mode) string {
	switch m {
	case FNC1First:
		return "first"
	case FNC1Second:
		return "second"
	default:
		return ""
	}
}
