package thermoaddons

import (
	"math"
	"testing"
)

func TestParseConc_AcceptsMicroVariants(t *testing.T) {
	cases := []string{"3uM", "3µM", "3μM"}
	for _, tc := range cases {
		got, err := ParseConc(tc)
		if err != nil {
			t.Fatalf("ParseConc(%q): %v", tc, err)
		}
		if math.Abs(got-3e-6) > 1e-15 {
			t.Fatalf("ParseConc(%q)=%g, want %g", tc, got, 3e-6)
		}
	}
}
