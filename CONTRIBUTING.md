# Contributing

Thanks for helping improve qr2.

## Before opening a pull request

Run these from the repository root:

```bash
go test -race ./...
go vet ./...
test -z "$(gofmt -l .)"
GOOS=js GOARCH=wasm go build -o /tmp/qr2.wasm ./cmd/engine
```

If you change tables, masks, format information, version information, Reed-Solomon code, segmentation, or SVG geometry, also run the independent decoder check:

```bash
go run ./cmd/corpus -out testdata/corpus
python -m pip install zxing-cpp==3.1.1 pillow==12.2.0
python tools/decode_corpus.py testdata/corpus
```

## What belongs here

This repository is the engine only: the encoder, matrix builder, self-check decoder, SVG renderer, and engine verification tools. UI work belongs in the web project.

Keep the implementation independent. Do not add a third-party QR encoder or link one into the engine.

Do not commit generated WASM, corpus images, or test binaries.

## Pull requests

Keep changes focused. Explain the behavior being changed and the compatibility risk. Include tests or reference vectors for matrix-level changes.

## License

By contributing, you agree that your contribution is licensed under the GNU General Public License, version 3 only, as described in [LICENSE](./LICENSE).

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).
