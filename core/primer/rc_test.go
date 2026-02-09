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

func TestRevCompPreservesCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// canonical bases
		{in: "acgt", want: "acgt"},
		{in: "aCgT", want: "AcGt"}, // reverse + complement with per-base case preservation
		// IUPAC ambiguity (lowercase)
		{in: "ryswkmbdhvn", want: "nbdhvkmwsry"},
	}

	for _, tc := range tests {
		got := string(RevComp([]byte(tc.in)))
		if got != tc.want {
			t.Errorf("RevComp(%q) = %q, want %q", tc.in, got, tc.want)
		}
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
