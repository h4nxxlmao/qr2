package qr

import (
	"bytes"
	"testing"
)

func FuzzEncodeRoundTrip(f *testing.F) {
	f.Add([]byte("HELLO WORLD"), uint8(0), int8(-1))
	f.Add([]byte("https://qr.hanx.pro"), uint8(1), int8(3))
	f.Add([]byte("012345678901234567890123456789"), uint8(2), int8(7))
	f.Add([]byte{0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff}, uint8(3), int8(0))

	f.Fuzz(func(t *testing.T, data []byte, eccBits uint8, maskBits int8) {
		if len(data) > 1024 {
			return
		}
		req := EncodeRequest{
			Symbology: QRModel2,
			Data:      data,
			ECC:       ECC(eccBits % 4),
			Mask:      int(maskBits%8) - 1,
		}
		syms, err := Encode(req)
		if err != nil {
			e, ok := err.(*Error)
			if !ok || (e.Code != "capacity" && e.Code != "invalid_request") {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		if len(syms) != 1 {
			t.Fatalf("expected one symbol, got %d", len(syms))
		}
		dec, err := SelfCheck(syms[0])
		if err != nil {
			t.Fatalf("self-check failed: %v", err)
		}
		if !bytes.Equal(dec.Data, data) {
			t.Fatalf("payload mismatch: got %q want %q", dec.Data, data)
		}
	})
}

func FuzzTextRoundTrip(f *testing.F) {
	f.Add("HELLO WORLD")
	f.Add("https://qr.hanx.pro/?q=hello")
	f.Add("1234567890")
	f.Add("multi\nline\ntext")
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 1024 {
			return
		}
		syms, err := Encode(EncodeRequest{
			Symbology: QRModel2,
			Text:      text,
			IsText:    true,
			ECC:       M,
			Mask:      -1,
		})
		if err != nil {
			return
		}
		dec, err := SelfCheck(syms[0])
		if err != nil {
			t.Fatalf("self-check failed: %v", err)
		}
		if !bytes.Equal(dec.Data, []byte(text)) {
			t.Fatalf("payload mismatch: got %q want %q", dec.Data, text)
		}
	})
}
