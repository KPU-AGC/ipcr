package pretty

import "testing"

func TestDefaultOptions_Stable(t *testing.T) {
	d := DefaultOptions
	if d.DotGlyph == "" || d.ExactGlyph == "" || d.PartialGlyph == "" {
		t.Fatalf("glyphs must be non-empty")
	}
	// Spot checks of current defaults (don’t lock everything, just the external look)
	if d.DotGlyph != "." || d.ExactGlyph != "|" || d.PartialGlyph != "¦" || !d.ShowProbeBars {
		t.Fatalf("DefaultOptions visual defaults changed")
	}
}
