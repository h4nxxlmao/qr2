// Command parity writes the native-Go expectations used by the WASM parity
// harness. It must stay in sync with the request shape in cmd/engine.
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/h4nxxlmao/qr2/qr"
	"github.com/h4nxxlmao/qr2/render"
)

type request struct {
	Text     string      `json:"text"`
	DataB64  string      `json:"dataB64"`
	IsText   bool        `json:"isText"`
	ECC      string      `json:"ecc"`
	Version  int         `json:"version"`
	Mask     int         `json:"mask"`
	BoostECC bool        `json:"boostECC"`
	ECI      int         `json:"eci"`
	FNC1     string      `json:"fnc1"`
	FNC1App  int         `json:"fnc1App"`
	Charset  string      `json:"charset"`
	Render   render.Spec `json:"render"`
}

type expectation struct {
	Version      int    `json:"version"`
	ECC          string `json:"ecc"`
	Mask         int    `json:"mask"`
	BitLength    int    `json:"bitLength"`
	CapacityBits int    `json:"capacityBits"`
	ModuleWidth  int    `json:"moduleWidth"`
	ModuleHeight int    `json:"moduleHeight"`
	SVGSize      int    `json:"svgSize"`
	SVGSHA256    string `json:"svgSha256"`
}

type parityCase struct {
	Name     string      `json:"name"`
	Request  request     `json:"request"`
	Expected expectation `json:"expected"`
}

func main() {
	out := flag.String("out", "testdata/parity.json", "output file")
	flag.Parse()

	base := render.Spec{
		Size: 640, QuietZone: 4, Foreground: "#181817", Background: "#f7f6f3",
		Shape: "square", ModuleScale: 1,
	}
	cases := []struct {
		name string
		req  request
	}{
		{"hello-m", request{Text: "HELLO WORLD", IsText: true, ECC: "M", Mask: -1, Render: base}},
		{"url-q", request{Text: "https://qr.hanx.pro", IsText: true, ECC: "Q", Mask: -1, Render: base}},
		{"rounded-dots", request{Text: "https://qr.hanx.pro", IsText: true, ECC: "M", Mask: -1, Render: withShape(base, "dots", 0.82, 0)}},
		{"long-h", request{Text: "The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog.", IsText: true, ECC: "H", Mask: -1, Render: base}},
		{"v40-h-mask5", request{Text: "VERSION CHECK", IsText: true, ECC: "H", Version: 40, Mask: 5, Render: base}},
		{"v1-mask2", request{Text: "01234567", IsText: true, ECC: "M", Version: 1, Mask: 2, Render: base}},
		{"binary", request{DataB64: base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02, 0x7f, 0x80, 0xfe, 0xff}), ECC: "Q", Mask: -1, Render: base}},
		{"transparent", request{Text: "TRANSPARENT", IsText: true, ECC: "M", Mask: -1, Render: withTransparent(base)}},
		{"roles", request{Text: "ROLE COLORS", IsText: true, ECC: "M", Mask: -1, Render: withRoles(base)}},
		{"rotation", request{Text: "ROTATED", IsText: true, ECC: "M", Mask: -1, Render: withRotation(base)}},
		{"fnc1", request{Text: "010001234560001210B1\x1d17261231", IsText: true, ECC: "M", Mask: -1, FNC1: "first", Render: base}},
		{"logo", request{Text: "LOGO TEST", IsText: true, ECC: "H", Mask: -1, Render: withLogo(base)}},
	}
	for m := 0; m < 8; m++ {
		cases = append(cases, struct {
			name string
			req  request
		}{fmt.Sprintf("mask-%d", m), request{Text: "MASK TEST 1234567890", IsText: true, ECC: "M", Version: 2, Mask: m, Render: base}})
	}

	outCases := make([]parityCase, 0, len(cases))
	for _, tc := range cases {
		var data []byte
		if tc.req.DataB64 != "" {
			var err error
			data, err = base64.StdEncoding.DecodeString(tc.req.DataB64)
			if err != nil {
				fail(fmt.Errorf("%s: %w", tc.name, err))
			}
		}
		syms, err := qr.Encode(qr.EncodeRequest{
			Symbology: qr.QRModel2,
			Data:      data,
			Text:      tc.req.Text,
			IsText:    tc.req.IsText,
			Charset:   parseCharset(tc.req.Charset),
			ECI:       tc.req.ECI,
			Version:   tc.req.Version,
			ECC:       parseECC(tc.req.ECC),
			Mask:      tc.req.Mask,
			BoostECC:  tc.req.BoostECC,
			FNC1:      parseFNC1(tc.req.FNC1),
			FNC1App:   byte(tc.req.FNC1App),
		})
		if err != nil {
			fail(fmt.Errorf("%s: %w", tc.name, err))
		}
		svg, err := render.RenderSVG(syms[0], tc.req.Render)
		if err != nil {
			fail(fmt.Errorf("%s: %w", tc.name, err))
		}
		sum := sha256.Sum256(svg)
		outCases = append(outCases, parityCase{
			Name:    tc.name,
			Request: tc.req,
			Expected: expectation{
				Version:      syms[0].Meta.Version,
				ECC:          syms[0].Meta.ECC,
				Mask:         syms[0].Meta.Mask,
				BitLength:    syms[0].Meta.BitLength,
				CapacityBits: syms[0].Meta.CapacityBits,
				ModuleWidth:  syms[0].Width,
				ModuleHeight: syms[0].Height,
				SVGSize:      len(svg),
				SVGSHA256:    hex.EncodeToString(sum[:]),
			},
		})
	}

	data, err := json.MarshalIndent(outCases, "", "  ")
	if err != nil {
		fail(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %d parity cases to %s\n", len(outCases), *out)
}

func withShape(spec render.Spec, shape string, scale, radius float64) render.Spec {
	spec.Shape = shape
	spec.ModuleScale = scale
	spec.CornerRadius = radius
	return spec
}

func withTransparent(spec render.Spec) render.Spec {
	spec.Transparent = true
	spec.Size = 512
	return spec
}

func withRoles(spec render.Spec) render.Spec {
	spec.FinderOuter = "#c2410c"
	spec.FinderInner = "#2f7d32"
	spec.Timing = "#1d4ed8"
	spec.Alignment = "#7c3aed"
	return spec
}

func withRotation(spec render.Spec) render.Spec {
	spec.Rotation = 15
	spec.Mirror = true
	return spec
}

func withLogo(spec render.Spec) render.Spec {
	spec.LogoDataURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Zl1sAAAAASUVORK5CYII="
	spec.LogoSizePct = 0.2
	spec.LogoMarginPct = 0.1
	spec.LogoShape = "circle"
	spec.LogoClear = true
	return spec
}

func parseECC(s string) qr.ECC {
	switch s {
	case "M":
		return qr.M
	case "Q":
		return qr.Q
	case "H":
		return qr.H
	default:
		return qr.L
	}
}

func parseFNC1(s string) qr.FNC1Mode {
	switch s {
	case "first":
		return qr.FNC1First
	case "second":
		return qr.FNC1Second
	default:
		return qr.FNC1None
	}
}

func parseCharset(s string) qr.Charset {
	if s == "latin1" {
		return qr.CharsetLatin1
	}
	return qr.CharsetUTF8
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
