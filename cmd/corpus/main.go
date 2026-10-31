// Command corpus generates PNG images and a manifest for independent decoder
// verification. It is a development tool, not part of the WASM build.
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/h4nxxlmao/qr2/qr"
)

type manifestCase struct {
	File         string `json:"file"`
	Name         string `json:"name"`
	ExpectedB64  string `json:"expected_b64"`
	ExpectedText string `json:"expected_text,omitempty"`
	Version      int    `json:"version"`
	ECC          string `json:"ecc"`
	Mask         int    `json:"mask"`
	Width        int    `json:"width"`
}

func main() {
	out := flag.String("out", "testdata/corpus", "output directory")
	scale := flag.Int("scale", 8, "pixels per module")
	quiet := flag.Int("quiet", 4, "quiet zone in modules")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}

	var manifest []manifestCase
	add := func(name string, data []byte, text string, req qr.EncodeRequest) {
		syms, err := qr.Encode(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", name, err)
			return
		}
		sym := syms[0]
		dec, err := qr.SelfCheck(sym)
		if err != nil {
			fmt.Fprintf(os.Stderr, "self-check %s: %v\n", name, err)
			return
		}
		if text != "" {
			if string(dec.Data) != text {
				fatal(fmt.Errorf("%s: self-check payload mismatch", name))
			}
		} else if string(dec.Data) != string(data) {
			fatal(fmt.Errorf("%s: self-check binary mismatch", name))
		}
		file := safeName(name) + ".png"
		path := filepath.Join(*out, file)
		if err := writePNG(path, sym, *scale, *quiet); err != nil {
			fatal(err)
		}
		manifest = append(manifest, manifestCase{
			File:         file,
			Name:         name,
			ExpectedB64:  base64.StdEncoding.EncodeToString(dec.Data),
			ExpectedText: text,
			Version:      sym.Meta.Version,
			ECC:          sym.Meta.ECC,
			Mask:         sym.Meta.Mask,
			Width:        sym.Width,
		})
	}

	textCases := []struct {
		name string
		text string
	}{
		{"hello", "HELLO WORLD"},
		{"url", "https://qr.hanx.pro"},
		{"numeric", "01234567890123456789"},
		{"alphanumeric", "QR CODE 2026 $%*+-./:"},
		{"utf8", "Grüße, 日本語, café"},
		{"multiline", "line one\nline two\nline three"},
		{"long", strings.Repeat("The quick brown fox jumps over the lazy dog. ", 10)},
	}
	for _, tc := range textCases {
		for _, ecc := range []qr.ECC{qr.L, qr.M, qr.Q, qr.H} {
			add("auto_"+tc.name+"_"+ecc.String(), nil, tc.text, qr.EncodeRequest{
				Symbology: qr.QRModel2, Text: tc.text, IsText: true, ECC: ecc, Mask: -1, BoostECC: false,
			})
		}
	}

	// Forced versions with a short payload, all ECC levels.
	for v := 1; v <= 40; v++ {
		for _, ecc := range []qr.ECC{qr.L, qr.M, qr.Q, qr.H} {
			add(fmt.Sprintf("v%02d_%s", v, ecc.String()), nil, "VERSION CHECK", qr.EncodeRequest{
				Symbology: qr.QRModel2, Text: "VERSION CHECK", IsText: true,
				Version: v, ECC: ecc, Mask: -1,
			})
		}
	}

	// All masks.
	for m := 0; m < 8; m++ {
		add(fmt.Sprintf("mask%d", m), nil, "MASK TEST 1234567890", qr.EncodeRequest{
			Symbology: qr.QRModel2, Text: "MASK TEST 1234567890", IsText: true,
			Version: 2, ECC: qr.M, Mask: m,
		})
	}

	// Binary payload, no UTF-8 assumption.
	binary := []byte{0x00, 0x01, 0x02, 0x7F, 0x80, 0xFE, 0xFF, 0x10, 0x20, 0x30}
	add("binary", binary, "", qr.EncodeRequest{
		Symbology: qr.QRModel2, Data: binary, ECC: qr.Q, Mask: -1,
	})

	manifestPath := filepath.Join(*out, "manifest.json")
	f, err := os.Create(manifestPath)
	if err != nil {
		fatal(err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		fatal(err)
	}
	if err := f.Close(); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %d corpus images to %s\n", len(manifest), *out)
}

func writePNG(path string, sym *qr.Symbol, scale, quiet int) error {
	dim := (sym.Width + 2*quiet) * scale
	img := image.NewGray(image.Rect(0, 0, dim, dim))
	white := color.Gray{Y: 255}
	black := color.Gray{Y: 0}
	for y := 0; y < dim; y++ {
		for x := 0; x < dim; x++ {
			img.SetGray(x, y, white)
		}
	}
	for y := 0; y < sym.Height; y++ {
		for x := 0; x < sym.Width; x++ {
			if sym.Module(x, y) == 0 {
				continue
			}
			px := (x + quiet) * scale
			py := (y + quiet) * scale
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetGray(px+dx, py+dy, black)
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
