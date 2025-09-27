// core/primer/rc_test.go
package primer

import (
	"bytes"
	"testing"
)

func TestRevCompSimple(t *testing.T) {
	got := RevComp([]byte("AGTC"))
	want := []byte("GACT")
	if !bytes.Equal(got, want) {
		t.Errorf("RevComp(AGTC) = %s, want %s", got, want)
	}
}

func TestRevCompAmbiguous(t *testing.T) {
	in := []byte("RYSWKMBDHVN")
	want := []byte("NBDHVKMWSRY")
	got := RevComp(in)
	if !bytes.Equal(got, want) {
		t.Errorf("RevComp(%s) = %s, want %s", in, got, want)
	}
}

func TestRevCompEmpty(t *testing.T) {
	if RevComp(nil) != nil {
		t.Errorf("RevComp(nil) should return nil")
	}
	if out := RevComp([]byte("")); len(out) != 0 {
		t.Errorf("RevComp(\"\") length = %d, want 0", len(out))
	}
}

// ===
