package runutil

import "testing"

func TestEffectiveTerminalWindow(t *testing.T) {
	if got := EffectiveTerminalWindow(3); got != 3 {
		t.Fatalf("want 3, got %d", got)
	}
	if got := EffectiveTerminalWindow(1); got != 1 {
		t.Fatalf("want 1, got %d", got)
	}
	if got := EffectiveTerminalWindow(0); got != 0 {
		t.Fatalf("0 disables → want 0, got %d", got)
	}
	if got := EffectiveTerminalWindow(-1); got != 0 {
		t.Fatalf("-1 disables → want 0, got %d", got)
	}
}

func TestComputeOverlap(t *testing.T) {
	if got := ComputeOverlap(100, 21); got != 100 {
		t.Fatalf("expect 100, got %d", got)
	}
	if got := ComputeOverlap(0, 21); got != 20 {
		t.Fatalf("expect maxPrimerLen-1=20, got %d", got)
	}
}

func TestValidateChunking(t *testing.T) {
	// circular disables
	cs, ov, w := ValidateChunking(true, 1000, 500, 25)
	if cs != 0 || ov != 0 || len(w) == 0 {
		t.Fatalf("circular should disable with warning")
	}
	// no maxLen
	cs, ov, w = ValidateChunking(false, 1000, 0, 25)
	if cs != 0 || ov != 0 || len(w) == 0 {
		t.Fatalf("missing maxLen should disable with warning")
	}
	// too small chunk
	cs, ov, w = ValidateChunking(false, 500, 500, 25)
	if cs != 0 || ov != 0 || len(w) == 0 {
		t.Fatalf("chunk<=maxLen should disable with warning")
	}
	// happy path
	cs, ov, w = ValidateChunking(false, 2000, 500, 25)
	if cs != 2000 || ov != 500 || len(w) != 0 {
		t.Fatalf("enabled: cs=%d ov=%d warns=%v", cs, ov, w)
	}
}
