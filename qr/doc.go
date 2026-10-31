// Package qr implements QR Model 2 encoding, matrix construction, mask
// selection, Reed-Solomon error correction, and a self-check decoder.
//
// Call Encode with an EncodeRequest to produce a Symbol. The Symbol contains
// the module grid, a per-module role grid for styling, and metadata about the
// version, error correction level, mask, segments, and capacity.
//
// SelfCheck decodes a Symbol matrix. It verifies the format and version
// information, unmasks the modules, checks every Reed-Solomon block, and parses
// the payload. It does not perform image detection.
package qr
