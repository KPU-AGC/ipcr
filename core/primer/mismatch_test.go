// core/primer/mismatch_test.go
package primer

import "testing"

func TestMismatchCount(t *testing.T) {
	// genome := []byte("ACGTACGT")
	tests := []struct {
		window string
		primer string
		want   int
	}{
		{"ACGT", "ACGT", 0}, // perfect
		{"ACGT", "NNNN", 0}, // N matches everything
		{"ACGT", "RRRR", 2}, // R=A/G  => mismatches at C,T = 2
		{"ACGT", "TTTT", 3}, // only final T matches
	}
	for _, tc := range tests {
		got := MismatchCount([]byte(tc.window), []byte(tc.primer))
		if got != tc.want {
			t.Errorf("MismatchCount(%q,%q) = %d, want %d",
				tc.window, tc.primer, got, tc.want)
		}
	}

	// Length-mismatch panic check (optional)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on unequal lengths")
		}
	}()
	MismatchCount([]byte("AAA"), []byte("AA"))
}
