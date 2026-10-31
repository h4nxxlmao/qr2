#!/usr/bin/env python3
"""Decode every image in a corpus manifest with zxing-cpp."""
import base64
import json
import pathlib
import sys

import zxingcpp
from PIL import Image


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: decode_corpus.py <corpus-dir>", file=sys.stderr)
        return 2
    root = pathlib.Path(sys.argv[1])
    cases = json.loads((root / "manifest.json").read_text(encoding="utf-8"))
    failures = []
    decoded = 0
    for case in cases:
        image = Image.open(root / case["file"])
        results = zxingcpp.read_barcodes(image)
        if not results:
            failures.append(f'{case["name"]}: no barcode found')
            continue
        result = results[0]
        expected = base64.b64decode(case["expected_b64"])
        if result.bytes != expected:
            failures.append(
                f'{case["name"]}: bytes {result.bytes!r} != {expected!r} '
                f'(text={result.text!r}, format={result.format})'
            )
            continue
        decoded += 1
        print(
            f'ok {case["name"]}: {result.format} v{case["version"]}-{case["ecc"]} '
            f'mask {case["mask"]}'
        )
    print(f"decoded {decoded}/{len(cases)}")
    if failures:
        print("\nfailures:", file=sys.stderr)
        for failure in failures:
            print("  " + failure, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
