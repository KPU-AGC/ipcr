package writers

import (
	"bytes"
	"strings"
	"testing"

	"ipcr-core/engine"
)

func TestUnknownProductFormatError(t *testing.T) {
	var b bytes.Buffer
	in, done := StartProductWriter(&b, "nope-format", false, false, false, 1)
	close(in) // no payload; writer should error out immediately on dispatch
	err := <-done
	if err == nil {
		t.Fatalf("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown product format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestUnknownAnnotatedFormatError(t *testing.T) {
	var b bytes.Buffer
	in, done := StartAnnotatedWriter(&b, "wat", false, false, false, 1)
	close(in)
	err := <-done
	if err == nil || !strings.Contains(err.Error(), "unknown annotated format") {
		t.Fatalf("want 'unknown annotated format' error, got: %v", err)
	}
}

func TestUnknownNestedFormatError(t *testing.T) {
	var b bytes.Buffer
	in, done := StartNestedWriter(&b, "???", false, false, 1)
	close(in)
	err := <-done
	if err == nil || !strings.Contains(err.Error(), "unknown nested format") {
		t.Fatalf("want 'unknown nested format' error, got: %v", err)
	}
}

// Make sure the package compiles with a trivial reference (avoid unused imports complaints).
var _ chan<- engine.Product
