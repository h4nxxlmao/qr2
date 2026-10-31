# qr2

[![CI](https://github.com/h4nxxlmao/qr2/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/h4nxxlmao/qr2/actions/workflows/ci.yml)

qr2 is a pure Go QR Model 2 encoder, matrix builder, self-check decoder, and SVG renderer. It has no third-party QR library and no runtime dependency outside the Go standard library. It is the engine behind qr.hanx.pro.

The project is pre-1.0. The API may change before v1.

## What it does

- QR Model 2, versions 1 through 40, error correction L/M/Q/H
- Numeric, alphanumeric, and byte modes
- Optimal mixed-mode segment selection
- All eight masks with N1 through N4 penalty scoring
- Reed-Solomon over GF(256), BCH format and version fields
- A self-check decoder that verifies the matrix and every Reed-Solomon block
- A per-module role map for data, finder, timing, alignment, format, and version modules
- SVG output with merged paths, module shapes, per-role colors, logo cutout, rotation, and mirroring
- Go and WebAssembly entrypoints

## Install

```bash
go get github.com/h4nxxlmao/qr2/qr
go get github.com/h4nxxlmao/qr2/render
```

## Quick start

```go
package main

import (
	"fmt"
	"os"

	"github.com/h4nxxlmao/qr2/qr"
	"github.com/h4nxxlmao/qr2/render"
)

func main() {
	symbols, err := qr.Encode(qr.EncodeRequest{
		Symbology: qr.QRModel2,
		Text:      "https://example.com",
		IsText:    true,
		ECC:       qr.M,
		Mask:      -1,
		BoostECC:  true,
	})
	if err != nil {
		panic(err)
	}

	svg, err := render.RenderSVG(symbols[0], render.Spec{
		Size:         640,
		QuietZone:    4,
		Foreground:   "#181817",
		Background:   "#f7f6f3",
		Shape:        "square",
		ModuleScale:  1,
		CornerRadius: 0,
	})
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("qr.svg", svg, 0o644); err != nil {
		panic(err)
	}
	fmt.Println(symbols[0].Meta)
}
```

`Encode` returns a slice because structured append is reserved for a later phase. The current implementation returns one symbol.

## API

```go
func Encode(req EncodeRequest) ([]*Symbol, error)
func SelfCheck(sym *Symbol) (*Decoded, error)

func RenderSVG(sym *qr.Symbol, spec Spec) ([]byte, error)
```

`Symbol` contains:

- `Width`, `Height`, and one byte per module in `Modules`
- `Roles`, a matching per-module role grid
- `Meta`, with version, effective ECC, mask, segments, bit length, capacity bits, capacity percent, and ECC boost state

`SelfCheck` reads the matrix directly. It does not detect a QR code in an image, correct damaged pixels, or replace a camera scanner. Image-level verification is a separate concern.

`render.Spec` accepts size, quiet zone, foreground, background, transparency, module shape, module scale, corner radius, per-role colors, rotation, mirroring, and an optional logo plate.

## WebAssembly

Build the browser entrypoint with the standard Go toolchain:

```bash
GOOS=js GOARCH=wasm go build -o qr2.wasm ./cmd/engine
```

For a smaller artifact, TinyGo can build the same entrypoint:

```bash
tinygo build -target=wasm -o qr2.wasm ./cmd/engine
```

The entrypoint exposes `globalThis.qrEngine`:

```js
qrEngine.version // "0.1.0"
qrEngine.abi     // 1
qrEngine.render(JSON.stringify({
  text: "https://example.com",
  isText: true,
  ecc: "M",
  mask: -1,
  version: 0,
  render: { size: 640, quietZone: 4, shape: "square" }
}))
```

A successful call returns `{ meta: string, svg: Uint8Array, roles: Uint8Array }`. A failed call returns `{ error: { code, message, hint } }`. The page must load the `wasm_exec.js` file that matches the toolchain used to build the module.

## Verification

```bash
go test -race ./...
go vet ./...
test -z "$(gofmt -l .)"
GOOS=js GOARCH=wasm go build -o /tmp/qr2.wasm ./cmd/engine
```

The independent decoder check uses a generated PNG corpus and zxing-cpp:

```bash
go run ./cmd/corpus -out testdata/corpus
python -m pip install zxing-cpp==3.1.1 pillow==12.2.0
python tools/decode_corpus.py testdata/corpus
```

The corpus covers versions 1 through 40 at each ECC level where the test payload fits, all eight masks, automatic version selection, UTF-8, binary bytes, and long payloads. zxing-cpp and Pillow are verification-only dependencies. They are never linked into the engine or shipped.

The native/TinyGo parity vectors are generated with:

```bash
go run ./cmd/parity -out testdata/parity.json
```

## Compatibility

| Feature | Status |
|---|---|
| QR Model 2 versions 1 through 40, L/M/Q/H | Implemented |
| Numeric, alphanumeric, byte modes | Implemented |
| Optimal mixed-mode segmentation | Implemented |
| ECI prefix | Implemented |
| FNC1 first and second position | Implemented |
| Structured append | Not implemented |
| Kanji mode | Not implemented |
| Micro QR M1 through M4 | Not implemented |
| rMQR | Not implemented |
| QR Model 1 | Not implemented |
| Data Matrix, Aztec, PDF417, Code 128, EAN/UPC | Not implemented |
| SVG output and logo plate | Implemented |
| Image decoding | Not implemented |

Only `qr.QRModel2` is implemented. The other symbology enum values return an unsupported error.

## Repository layout

```text
qr/          encoder, matrix builder, masks, Reed-Solomon, BCH, self-check
render/      SVG renderer
cmd/engine/  WebAssembly JSON entrypoint
cmd/corpus/  PNG corpus generator for independent decoding
cmd/parity/  native and TinyGo parity vector generator
tools/       external corpus decoder
testdata/    committed parity vectors
```

The web application is a separate project. This repository contains only the engine and its verification tools.

## Security

The encoder and renderer do not perform network requests and do not write to disk. The WASM entrypoint exposes a JSON render function and keeps no request history. Payload text is never emitted as SVG markup.

Report vulnerabilities through the process in [SECURITY.md](./SECURITY.md).

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Before opening a pull request, run the tests, vet, formatting check, WASM build, and the independent decoder check.

## License

MIT. See [LICENSE](./LICENSE).

QR Code is a registered trademark of DENSO WAVE INCORPORATED. This project is an independent implementation and is not affiliated with or endorsed by DENSO WAVE.
