// internal/writers/registry.go
package writers

import (
	"fmt"
	"io"
)

var (
	ProductWriters   = map[string]func(io.Writer, interface{}) error{}
	AnnotatedWriters = map[string]func(io.Writer, interface{}) error{}
	NestedWriters    = map[string]func(io.Writer, interface{}) error{}
)

// Register helpers (idempotent, last-wins).
func RegisterProduct(format string, fn func(io.Writer, interface{}) error) {
	ProductWriters[format] = fn
}

func RegisterAnnotated(format string, fn func(io.Writer, interface{}) error) {
	AnnotatedWriters[format] = fn
}

func RegisterNested(format string, fn func(io.Writer, interface{}) error) {
	NestedWriters[format] = fn
}

// Dispatch helpers used by factories/callers.
func WriteProduct(format string, w io.Writer, payload interface{}) error {
	fn, ok := ProductWriters[format]
	if !ok {
		return fmt.Errorf("unknown product format %q (no writer registered)", format)
	}
	return fn(w, payload)
}

func WriteAnnotated(format string, w io.Writer, payload interface{}) error {
	fn, ok := AnnotatedWriters[format]
	if !ok {
		return fmt.Errorf("unknown annotated format %q (no writer registered)", format)
	}
	return fn(w, payload)
}

func WriteNested(format string, w io.Writer, payload interface{}) error {
	fn, ok := NestedWriters[format]
	if !ok {
		return fmt.Errorf("unknown nested format %q (no writer registered)", format)
	}
	return fn(w, payload)
}
