// internal/primer/iupac_test.go
package primer

import "testing"

func TestBaseMatch(t *testing.T) {
	tests := []struct {
		g, p byte
		want bool
	}{
		{'A', 'A', true},
		{'G', 'R', true},  // R = A/G
		{'C', 'R', false},
		{'T', 'N', true},  // N = any
		{'G', 'N', true},
		{'A', 'B', false}, // B = C/G/T  (not A)
		{'C', 'B', true},
		{'T', 'X', false}, // unknown primer char
	}
	for _, tt := range tests {
		if got := BaseMatch(tt.g, tt.p); got != tt.want {
			t.Errorf("BaseMatch(%q,%q) = %v, want %v", tt.g, tt.p, got, tt.want)
		}
	}
}
