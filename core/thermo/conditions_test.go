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
	for _, raw := range []string{"", "monovalent", "owczarzy-lite", "owczarzy08"} {
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
	mono := EffectiveMonovalent(na, mg, 0, SaltModelMonovalent)
	if mono != na {
		t.Fatalf("monovalent model changed Na: got %g want %g", mono, na)
	}
	lite := EffectiveMonovalent(na, mg, 0, SaltModelOwczarzyLite)
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

func TestFreeMagnesiumSubtractsDNTPSafely(t *testing.T) {
	got := FreeMagnesium(0.003, 0.0008)
	if !(got > 0.002 && got < 0.003) {
		t.Fatalf("FreeMagnesium got %g, want positive chelated Mg below total Mg", got)
	}
	if got := FreeMagnesium(0.001, 0.002); !(got >= 0 && got < 0.001) {
		t.Fatalf("FreeMagnesium should remain bounded below total Mg, got %g", got)
	}
}

func TestOwczarzy08TmInputPreservesRawNaAndFreeMg(t *testing.T) {
	c := Conditions{NaM: 0.05, MgM: 0.003, DntpM: 0.0008, PrimerTotalM: 2.5e-7, SaltModel: SaltModelOwczarzy08}
	in := c.TmInput()
	if in.SaltModel != SaltModelOwczarzy08 || in.Na != c.NaM || in.Mg != c.MgM || in.Dntp != c.DntpM {
		t.Fatalf("bad owczarzy08 TmInput: %+v", in)
	}
	if !(c.FreeMgM() > 0.002 && c.FreeMgM() < c.MgM) {
		t.Fatalf("bad free Mg: %g", c.FreeMgM())
	}
}
