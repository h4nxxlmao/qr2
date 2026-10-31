# Third-party notices

qr2 has no third-party runtime dependency. The library uses only the Go standard library.

## Verification-only dependencies

These tools are used by the corpus check and CI. They are not linked into the engine and are not shipped.

- zxing-cpp 3.1.1, Apache-2.0, used by `tools/decode_corpus.py`.
- Pillow 12.2.0, HPND, used by `tools/decode_corpus.py` to read PNG corpus images.
- GitHub Actions runners and actions, used only for continuous integration.

## Trademarks

QR Code is a registered trademark of DENSO WAVE INCORPORATED. This project is an independent implementation and is not affiliated with or endorsed by DENSO WAVE.
