// SPDX-License-Identifier: GPL-3.0-only

// Package render turns a qr.Symbol into an SVG document.
//
// The default output is a square QR code with a four-module quiet zone. Spec
// controls size, colors, module shape, per-role colors, rotation, mirroring,
// and an optional logo plate. The renderer emits one merged path per paint
// group instead of one element per module.
package render
