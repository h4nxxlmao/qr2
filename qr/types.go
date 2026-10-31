// SPDX-License-Identifier: GPL-3.0-only

package qr

import "fmt"

// Symbology identifies the symbol family.
type Symbology uint8

const (
	QRModel2 Symbology = iota
	MicroQR
	RMQR
	QRModel1
)

func (s Symbology) String() string {
	switch s {
	case QRModel2:
		return "QR Model 2"
	case MicroQR:
		return "Micro QR"
	case RMQR:
		return "rMQR"
	case QRModel1:
		return "QR Model 1"
	default:
		return "unknown"
	}
}

// ECC is an error correction level. The order is also the format-info order.
type ECC uint8

const (
	L ECC = iota
	M
	Q
	H
)

func (e ECC) String() string {
	switch e {
	case L:
		return "L"
	case M:
		return "M"
	case Q:
		return "Q"
	case H:
		return "H"
	default:
		return "?"
	}
}

// FormatBits returns the two-bit ECC value used in QR format information.
func (e ECC) FormatBits() int {
	switch e {
	case L:
		return 1
	case M:
		return 0
	case Q:
		return 3
	case H:
		return 2
	default:
		return 0
	}
}

// FNC1Mode selects GS1 / AIM application identifier handling.
type FNC1Mode uint8

const (
	FNC1None FNC1Mode = iota
	FNC1First
	FNC1Second
)

// Charset controls how text is converted to bytes before encoding.
type Charset uint8

const (
	CharsetAuto Charset = iota
	CharsetUTF8
	CharsetLatin1
)

// EncodeRequest is the input to Encode. Data is authoritative when IsText is false.
type EncodeRequest struct {
	Symbology Symbology
	Data      []byte
	Text      string
	IsText    bool
	Charset   Charset
	ECI       int
	Version   int // 0 means auto; 1..40 for QR Model 2
	ECC       ECC
	Mask      int // -1 means auto; 0..7
	BoostECC  bool
	FNC1      FNC1Mode
	FNC1App   byte
	// StructuredAppend is reserved for the next engine phase.
	StructuredAppend bool
	MaxSymbols       int
}

// Role is a bitmask describing what a module belongs to. The renderer uses it
// to style data, eyes, timing, and alignment independently.
type Role uint8

const (
	RoleData Role = 0

	RoleFinderOuter Role = 1 << 0
	RoleFinderInner Role = 1 << 1
	RoleFinderWhite Role = 1 << 2
	RoleTiming      Role = 1 << 3
	RoleAlignment   Role = 1 << 4
	RoleFormat      Role = 1 << 5
	RoleVersion     Role = 1 << 6
	RoleDark        Role = 1 << 7
)

// Symbol is a complete module matrix plus its metadata.
type Symbol struct {
	Width   int
	Height  int
	Modules []byte // one byte per module, 1 = dark, 0 = light
	Roles   []Role // one role per module
	Meta    Metadata
}

// Module returns 1 for a dark module, 0 otherwise.
func (s *Symbol) Module(x, y int) byte {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return 0
	}
	return s.Modules[y*s.Width+x]
}

// RoleAt returns the role at a coordinate.
func (s *Symbol) RoleAt(x, y int) Role {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return RoleData
	}
	return s.Roles[y*s.Width+x]
}

// Metadata describes how the symbol was produced.
type Metadata struct {
	Symbology       Symbology     `json:"symbology"`
	Version         int           `json:"version"`
	ECC             string        `json:"ecc"`
	Mask            int           `json:"mask"`
	Segments        []SegmentInfo `json:"segments"`
	BitLength       int           `json:"bitLength"`
	CapacityBits    int           `json:"capacityBits"`
	CapacityPercent float64       `json:"capacityPercent"`
	ECCBoosted      bool          `json:"eccBoosted"`
	FNC1            string        `json:"fnc1,omitempty"`
	ECI             int           `json:"eci,omitempty"`
}

// SegmentInfo is a human-readable description of one encoded segment.
type SegmentInfo struct {
	Mode  string `json:"mode"`
	Count int    `json:"count"`
	Bits  int    `json:"bits"`
}

func (m Metadata) String() string {
	return fmt.Sprintf("%s v%d-%s mask=%d bits=%d/%d (%.1f%%)",
		m.Symbology, m.Version, m.ECC, m.Mask, m.BitLength, m.CapacityBits, m.CapacityPercent)
}
