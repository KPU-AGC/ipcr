package thermo

import (
	"math"
	"testing"
)

func TestParseConcAcceptsMicroVariants(t *testing.T) {
	for _, s := range []string{"3uM", "3µM", "3μM"} {
		got, err := ParseConc(s)
		if err != nil {
			t.Fatalf("ParseConc(%q): %v", s, err)
		}
		if math.Abs(got-3e-6) > 1e-15 {
			t.Fatalf("ParseConc(%q)=%g, want 3e-6", s, got)
		}
	}
}

func TestParseSaltModel(t *testing.T) {
	for _, raw := range []string{"", "monovalent", "owczarzy-lite"} {
		if _, err := ParseSaltModel(raw); err != nil {
			t.Fatalf("ParseSaltModel(%q): %v", raw, err)
		}
	}
	if _, err := ParseSaltModel("hidden-env"); err == nil {
		t.Fatal("expected unknown salt model error")
	}
}

func TestEffectiveMonovalentSaltModels(t *testing.T) {
	na := 0.05
	mg := 0.003
	mono := EffectiveMonovalent(na, mg, SaltModelMonovalent)
	if mono != na {
		t.Fatalf("monovalent model changed Na: got %g want %g", mono, na)
	}
	lite := EffectiveMonovalent(na, mg, SaltModelOwczarzyLite)
	if !(lite > na) {
		t.Fatalf("owczarzy-lite should increase effective Na with Mg: got %g <= %g", lite, na)
	}
}

func TestConditionsTmInputUsesEffectiveSaltAndSelfFactor(t *testing.T) {
	c := Conditions{
		AnnealC:           55,
		NaM:               0.05,
		MgM:               0.003,
		PrimerTotalM:      2.5e-7,
		SaltModel:         SaltModelOwczarzyLite,
		SelfComplementary: true,
	}
	in := c.TmInput()
	if in.CT != c.PrimerTotalM || in.X != 1 {
		t.Fatalf("bad TmInput concentration/factor: %+v", in)
	}
	if !(in.Na > c.NaM) {
		t.Fatalf("expected effective Na > raw Na under owczarzy-lite: %+v", in)
	}
}
