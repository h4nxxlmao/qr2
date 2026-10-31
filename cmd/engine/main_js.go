//go:build js && wasm

// Command engine is the browser entry point. It exposes a small JSON API on
// globalThis.qrEngine and keeps the Go runtime alive for repeated calls.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/h4nxxlmao/qr2/qr"
	"github.com/h4nxxlmao/qr2/render"
)

const (
	engineVersion = "0.1.0"
	abiVersion    = 1
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

type metaOut struct {
	qr.Metadata
	ABI          int `json:"abi"`
	Engine       int `json:"engine"`
	SymbolCount  int `json:"symbolCount"`
	SVGSize      int `json:"svgSize"`
	ModuleWidth  int `json:"moduleWidth"`
	ModuleHeight int `json:"moduleHeight"`
}

func main() {
	api := map[string]any{
		"version": js.ValueOf(engineVersion),
		"abi":     js.ValueOf(abiVersion),
		"render":  js.FuncOf(renderCall),
	}
	js.Global().Set("qrEngine", js.ValueOf(api))
	select {}
}

func renderCall(this js.Value, args []js.Value) (ret any) {
	defer func() {
		if r := recover(); r != nil {
			ret = errorValue("panic", fmt.Sprintf("engine panic: %v", r), "This is a bug. Please report it with the payload that triggered it.")
		}
	}()
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return errorValue("invalid_request", "render expects a JSON string", "Check the worker request.")
	}
	var req request
	if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
		return errorValue("invalid_json", "request JSON could not be parsed", "Check the worker request.")
	}
	if req.Version < 0 || req.Version > 40 {
		return errorValue("invalid_version", "version must be auto or 1 to 40", "Use version 0 for automatic selection.")
	}
	if req.Mask < -1 || req.Mask > 7 {
		return errorValue("invalid_mask", "mask must be auto or 0 to 7", "Use -1 for automatic mask selection.")
	}
	ecc, ok := parseECC(req.ECC)
	if !ok {
		return errorValue("invalid_ecc", "unknown error correction level", "Choose L, M, Q, or H.")
	}
	fnc1 := parseFNC1(req.FNC1)
	charset := parseCharset(req.Charset)

	var data []byte
	if req.DataB64 != "" {
		b, err := base64.StdEncoding.DecodeString(req.DataB64)
		if err != nil {
			return errorValue("invalid_base64", "binary payload could not be decoded", "Send the payload as valid base64.")
		}
		data = b
	}

	syms, err := qr.Encode(qr.EncodeRequest{
		Symbology: qr.QRModel2,
		Data:      data,
		Text:      req.Text,
		IsText:    req.IsText,
		Charset:   charset,
		ECI:       req.ECI,
		Version:   req.Version,
		ECC:       ecc,
		Mask:      req.Mask,
		BoostECC:  req.BoostECC,
		FNC1:      fnc1,
		FNC1App:   byte(req.FNC1App),
	})
	if err != nil {
		return engineErrorValue(err)
	}
	if len(syms) == 0 {
		return errorValue("internal", "engine returned no symbols", "This is a bug. Please report it.")
	}
	sym := syms[0]
	svg, rerr := render.RenderSVG(sym, req.Render)
	if rerr != nil {
		return errorValue("render", rerr.Error(), "Check the style options.")
	}
	meta := metaOut{
		Metadata:     sym.Meta,
		ABI:          abiVersion,
		Engine:       1,
		SymbolCount:  len(syms),
		SVGSize:      len(svg),
		ModuleWidth:  sym.Width,
		ModuleHeight: sym.Height,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return errorValue("internal", "metadata serialization failed", "This is a bug. Please report it.")
	}
	roles := make([]byte, len(sym.Roles))
	for i, r := range sym.Roles {
		roles[i] = byte(r)
	}
	return map[string]any{
		"meta":  string(metaJSON),
		"svg":   bytesToJS(svg),
		"roles": bytesToJS(roles),
	}
}

func parseECC(s string) (qr.ECC, bool) {
	switch s {
	case "", "L", "l":
		return qr.L, true
	case "M", "m":
		return qr.M, true
	case "Q", "q":
		return qr.Q, true
	case "H", "h":
		return qr.H, true
	default:
		return qr.L, false
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
	switch s {
	case "latin1", "iso-8859-1":
		return qr.CharsetLatin1
	case "utf8", "utf-8", "":
		return qr.CharsetUTF8
	default:
		return qr.CharsetAuto
	}
}

func bytesToJS(b []byte) js.Value {
	u := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u, b)
	return u
}

func errorValue(code, message, hint string) map[string]any {
	return map[string]any{"error": map[string]any{
		"code": code, "message": message, "hint": hint,
	}}
}

func engineErrorValue(err error) map[string]any {
	var qe *qr.Error
	if e, ok := err.(*qr.Error); ok {
		qe = e
	}
	if qe != nil {
		return errorValue(qe.Code, qe.Message, qe.Hint)
	}
	return errorValue("engine", err.Error(), "Check the payload and encoding options.")
}
