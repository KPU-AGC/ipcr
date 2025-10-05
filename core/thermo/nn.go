// core/thermo/nn.go
// Nearest-neighbor thermodynamics for DNA duplexes (SantaLucia unified set).
// Units: ΔH in kcal/mol, ΔS in cal/(K·mol). Tm in °C.
//
// Steps:
//  1) Sum initiation + per-stack ΔH/ΔS (Table 1) + terminal AT penalties + symmetry.
//  2) Salt correction to ΔS for monovalent ions: ΔS([Na+]) = ΔS(1M) + 0.368*(N/2)*ln[Na+].
//  3) Two-state Tm (K): Tm = ΔH*1000 / (ΔS_Na + R ln(CT/x)) − 273.15 (°C).
//
// This package has no app/output deps; engine can import it cleanly.

package thermo

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	// Gas constant in cal/(K·mol)
	Rcal = 1.9872
)

// NNParams holds nearest-neighbor propagation parameters.
type NNParams struct {
	DH float64 // kcal/mol
	DS float64 // cal/(K·mol)
}

// Watson–Crick propagation parameters (1 M Na+), dimers 5'→3' top/bottom.
// SantaLucia & Hicks (2004), Table 1.
var dimerParams = map[string]NNParams{
	// Canonical 10
	"AA/TT": {-7.6, -21.3},
	"AT/TA": {-7.2, -20.4},
	"TA/AT": {-7.2, -21.3},
	"CA/GT": {-8.5, -22.7},
	"GT/CA": {-8.4, -22.4},
	"CT/GA": {-7.8, -21.0},
	"GA/CT": {-8.2, -22.2},
	"CG/GC": {-10.6, -27.2},
	"GC/CG": {-9.8, -24.4},
	"GG/CC": {-8.0, -19.9},

	// Synonym keys (reverse-orientation / swapped forms)
	"TT/AA": {-7.6, -21.3}, // same as AA/TT
	"CC/GG": {-8.0, -19.9}, // same as GG/CC
	"AC/TG": {-8.5, -22.7}, // same as CA/GT
	"TG/AC": {-8.4, -22.4}, // same as GT/CA
	"AG/TC": {-8.2, -22.2}, // same as GA/CT
	"TC/AG": {-7.8, -21.0}, // same as CT/GA
}

// Initiation / terminal / symmetry (1 M Na+).
var (
	initDH, initDS       = +0.2, -5.7 // initiation
	termAT_DH, termAT_DS = +2.2, +6.9 // once per terminal AT pair
	symmDH, symmDS       = 0.0, -1.4  // self-complementary correction
)

// TmInput describes solution and concentration.
type TmInput struct {
	CT float64 // total strand conc (mol/L); non-self: formula uses ln(CT/x)
	Na float64 // monovalent cations (mol/L), e.g. 0.05 for 50 mM
	X  int     // duplex type: 4 (non-self, default) or 1 (self-compl)
}

// Result reports ΔH/ΔS (1M and salt-corrected) and Tm.
type Result struct {
	DH_kcal float64 // total ΔH (kcal/mol)
	DS_cal  float64 // total ΔS at 1 M (cal/K·mol)
	DS_Na   float64 // ΔS corrected by [Na+] (cal/K·mol)
	TmC     float64 // melting temperature (°C)
}

// Tm computes Tm for primer (5'→3') vs target (3'→5') aligned WC.
// Seqs must be equal length; only A/C/G/T bases supported.
func Tm(primer5to3, target3to5 string, in TmInput) (Result, error) {
	var out Result

	p := strings.ToUpper(strings.TrimSpace(primer5to3))
	t := strings.ToUpper(strings.TrimSpace(target3to5))
	if len(p) == 0 || len(t) == 0 || len(p) != len(t) {
		return out, errors.New("Tm: sequences must be equal length and non-empty")
	}
	if in.CT <= 0 {
		return out, errors.New("Tm: CT must be > 0")
	}
	if in.Na <= 0 {
		return out, errors.New("Tm: [Na+] must be > 0")
	}
	x := in.X
	if x != 1 && x != 4 {
		x = 4
	}

	// Validate target bases (keeps prior error semantics).
	if _, ok := revCompStrict(t); !ok {
		return out, errors.New("Tm: non-ACGT base in target")
	}

	// Check per-index WC pairing (target is given 3'→5').
	for i := 0; i < len(p); i++ {
		if !wc(p[i], t[i]) {
			return out, fmt.Errorf("Tm: non-WC pair at pos %d (%c/%c)", i, p[i], t[i])
		}
	}

	// Build bottom 5'→3' as the complement of the top (primer).
	// This yields canonical keys like "GT/CA", "AT/TA", etc.
	bot, ok := compStrict(p)
	if !ok {
		return out, errors.New("Tm: non-ACGT base in primer")
	}

	// 1) Sum ΔH/ΔS (1 M Na+) over stacks + initiation.
	n := len(p)
	DH := initDH
	DS := initDS

	for i := 0; i < n-1; i++ {
		top2 := p[i : i+2]
		bot2 := bot[i : i+2]
		key := top2 + "/" + bot2
		if prm, ok := dimerParams[key]; ok {
			DH += prm.DH
			DS += prm.DS
			continue
		}
		// Also accept the reversed orientation (read both strands opposite).
		rkey := reverse2(top2) + "/" + reverse2(bot2)
		if prm, ok := dimerParams[rkey]; ok {
			DH += prm.DH
			DS += prm.DS
			continue
		}
		return out, fmt.Errorf("Tm: missing NN params for dimer %q", key)
	}

	// Terminal AT penalties (each end).
	if isATPair(p[0], bot[0]) {
		DH += termAT_DH
		DS += termAT_DS
	}
	if isATPair(p[n-1], bot[n-1]) {
		DH += termAT_DH
		DS += termAT_DS
	}

	// Symmetry correction for self-complementary duplexes.
	if isSelfCompl(p) {
		DH += symmDH
		DS += symmDS
	}

	// 2) Salt correction: ΔS([Na+]) = ΔS(1M) + 0.368*(N/2)*ln[Na+]; N = 2*n − 2 phosphates.
	N := float64(2*n - 2)
	DS_Na := DS + 0.368*(N/2.0)*math.Log(in.Na)

	// 3) Two-state Tm (K), then °C. ΔH in cal/mol.
	tmK := (DH * 1000.0) / (DS_Na + Rcal*math.Log(in.CT/float64(x)))
	out.DH_kcal = DH
	out.DS_cal = DS
	out.DS_Na = DS_Na
	out.TmC = tmK - 273.15
	return out, nil
}

// ---------- helpers ----------

func wc(a, b byte) bool {
	switch a {
	case 'A':
		return b == 'T'
	case 'C':
		return b == 'G'
	case 'G':
		return b == 'C'
	case 'T':
		return b == 'A'
	default:
		return false
	}
}

func isATPair(a, b byte) bool { return (a == 'A' && b == 'T') || (a == 'T' && b == 'A') }

func revCompStrict(s string) (string, bool) {
	rc := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A', 'a':
			rc[len(s)-1-i] = 'T'
		case 'C', 'c':
			rc[len(s)-1-i] = 'G'
		case 'G', 'g':
			rc[len(s)-1-i] = 'C'
		case 'T', 't':
			rc[len(s)-1-i] = 'A'
		default:
			return "", false
		}
	}
	return string(rc), true
}

func compStrict(s string) (string, bool) {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 'A', 'a':
			out[i] = 'T'
		case 'C', 'c':
			out[i] = 'G'
		case 'G', 'g':
			out[i] = 'C'
		case 'T', 't':
			out[i] = 'A'
		default:
			return "", false
		}
	}
	return string(out), true
}

func reverse2(s string) string {
	if len(s) <= 1 {
		return s
	}
	b := []byte(s)
	return string([]byte{b[1], b[0]})
}

func isSelfCompl(s string) bool {
	rc, ok := revCompStrict(s)
	return ok && strings.EqualFold(s, rc)
}
