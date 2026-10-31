// SPDX-License-Identifier: GPL-3.0-only

package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/h4nxxlmao/qr2/qr"
)

func sampleSymbol(t *testing.T) *qr.Symbol {
	t.Helper()
	syms, err := qr.Encode(qr.EncodeRequest{
		Symbology: qr.QRModel2,
		Text:      "https://qr.hanx.pro",
		IsText:    true,
		ECC:       qr.M,
		Mask:      -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return syms[0]
}

func TestRenderShapes(t *testing.T) {
	sym := sampleSymbol(t)
	for _, shape := range []string{"square", "rounded", "dots", "diamond"} {
		spec := Spec{
			Size:         512,
			QuietZone:    4,
			Foreground:   "#111111",
			Background:   "#ffffff",
			Shape:        shape,
			ModuleScale:  1,
			CornerRadius: 0.2,
		}
		svg, err := RenderSVG(sym, spec)
		if err != nil {
			t.Fatalf("%s: %v", shape, err)
		}
		s := string(svg)
		if !strings.HasPrefix(s, "<svg ") || !strings.HasSuffix(s, "</svg>") {
			t.Fatalf("%s: malformed svg", shape)
		}
		wantViewBox := `viewBox="0 0 33 33"` // v2 (25 modules) plus a 4-module quiet zone
		if !strings.Contains(s, wantViewBox) {
			t.Fatalf("%s: unexpected viewBox: %s", shape, s[:min(200, len(s))])
		}
		if strings.Count(s, "<rect") > 1 {
			t.Fatalf("%s: found per-module rects; only the background rect is allowed", shape)
		}
		if strings.Contains(s, "<text") {
			t.Fatalf("%s: found text element", shape)
		}
	}
}

func TestColorInjectionRejected(t *testing.T) {
	sym := sampleSymbol(t)
	spec := Spec{Size: 200, Foreground: `"><script>alert(1)</script>`, Background: "white"}
	svg, err := RenderSVG(sym, spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(svg)
	if strings.Contains(s, "<script") {
		t.Fatal("script injection reached the SVG")
	}
	if !strings.Contains(s, `fill="#111111"`) {
		t.Fatal("fallback foreground was not used")
	}
}

func TestTransparent(t *testing.T) {
	sym := sampleSymbol(t)
	svg, err := RenderSVG(sym, Spec{Size: 256, QuietZone: 4, Transparent: true})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(svg, []byte(`<rect`)) {
		t.Fatal("transparent output should not contain a background rect")
	}
}

func TestRoleColors(t *testing.T) {
	sym := sampleSymbol(t)
	spec := Spec{
		Size: 300, QuietZone: 4,
		Foreground: "#000000", Background: "#ffffff",
		FinderOuter: "#ff0000", FinderInner: "#00ff00", Timing: "#0000ff", Alignment: "#ff00ff",
	}
	svg, err := RenderSVG(sym, spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(svg)
	for _, want := range []string{"#ff0000", "#00ff00", "#0000ff", "#ff00ff"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing role color %s", want)
		}
	}
}

func TestLogo(t *testing.T) {
	sym := sampleSymbol(t)
	png := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Zl1sAAAAASUVORK5CYII="
	spec := Spec{
		Size: 512, QuietZone: 4, Foreground: "#181817", Background: "#f7f6f3",
		LogoDataURI: png, LogoSizePct: 0.2, LogoMarginPct: 0.1, LogoShape: "circle", LogoClear: true,
	}
	svg, err := RenderSVG(sym, spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(svg)
	for _, want := range []string{`<image href="data:image/png;base64,`, `<clipPath id="qrlogoClip">`, `<circle`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(s, "javascript:") {
		t.Fatal("unsafe logo URI reached the SVG")
	}
}

func TestBadLogoURI(t *testing.T) {
	sym := sampleSymbol(t)
	svg, err := RenderSVG(sym, Spec{
		Size: 200, QuietZone: 4, LogoDataURI: `data:text/html;base64,PHNjcmlwdD4=`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(svg), "<image") {
		t.Fatal("non-image data URI was embedded")
	}
}

func TestDeterministic(t *testing.T) {
	sym := sampleSymbol(t)
	spec := Spec{Size: 512, QuietZone: 4, Shape: "rounded", CornerRadius: 0.3}
	a, err := RenderSVG(sym, spec)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RenderSVG(sym, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("renderer is not deterministic")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
