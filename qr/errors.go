// SPDX-License-Identifier: GPL-3.0-only

package qr

import "fmt"

// Error is a structured engine error. The WASM bridge and UI rely on Code and
// Hint being stable enough to render a useful message.
type Error struct {
	Code    string
	Message string
	Hint    string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func errf(code, hint string, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Hint: hint}
}

var (
	errInvalidRequest = func(msg string) *Error {
		return errf("invalid_request", "Check the selected options and try again.", "%s", msg)
	}
	errUnsupported = func(msg string) *Error {
		return errf("unsupported", "This combination is not part of the current engine build.", "%s", msg)
	}
	errCapacity = func(bits, capacity int) *Error {
		return errf("capacity", "Use a smaller version, a lower error correction level, or shorter content.",
			"data needs %d bits, but the selected symbol holds %d", bits, capacity)
	}
)
