package primer

import "testing"

func TestIUPACMask_Snapshot(t *testing.T) {
	// Spot check canonical bases
	if iupacMask['A'] != 1 || iupacMask['C'] != 2 || iupacMask['G'] != 4 || iupacMask['T'] != 8 {
		t.Fatalf("canonical masks corrupted: A=%d C=%d G=%d T=%d", iupacMask['A'], iupacMask['C'], iupacMask['G'], iupacMask['T'])
	}
	// U must behave like T
	if iupacMask['U'] != iupacMask['T'] || iupacMask['u'] != iupacMask['t'] {
		t.Fatalf("U/u must equal T/t")
	}
	// Ambiguity spot checks (these guard accidental removals)
	if iupacMask['R'] != (1|4) || iupacMask['Y'] != (2|8) || iupacMask['N'] != (1|2|4|8) {
		t.Fatalf("ambiguity masks corrupted: R=%d Y=%d N=%d", iupacMask['R'], iupacMask['Y'], iupacMask['N'])
	}
	// Lowercase mirrors uppercase
	if iupacMask['r'] != iupacMask['R'] || iupacMask['n'] != iupacMask['N'] {
		t.Fatalf("lowercase masks must mirror uppercase")
	}
}
