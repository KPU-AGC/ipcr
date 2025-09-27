// ./internal/oligo/validate.go
package oligo

import (
	"fmt"
	"strings"
	"unicode"
)

// Allowed IUPAC DNA codes and their base sets.
var iupac = map[rune]string{
	'A': "A",
	'C': "C",
	'G': "G",
	'T': "T",
	'R': "AG",
	'Y': "CT",
	'S': "CG",
	'W': "AT",
	'K': "GT",
	'M': "AC",
	'B': "CGT",
	'D': "AGT",
	'H': "ACT",
	'V': "ACG",
	'N': "ACGT",
}

// Complement map for IUPAC codes.
var complement = map[rune]rune{
	'A': 'T', 'C': 'G', 'G': 'C', 'T': 'A',
	'R': 'Y', 'Y': 'R', 'S': 'S', 'W': 'W',
	'K': 'M', 'M': 'K', 'B': 'V', 'D': 'H',
	'H': 'D', 'V': 'B', 'N': 'N',
}

// Normalize removes spaces/quotes and uppercases bases.
func Normalize(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsSpace(r) || r == '\'' || r == '"' {
			continue
		}
		out = append(out, unicode.ToUpper(r))
	}
	return string(out)
}

// Validate returns a normalized sequence or an error if any char is non-IUPAC.
func Validate(raw string) (string, error) {
	s := Normalize(raw)
	if s == "" {
		return s, fmt.Errorf("empty oligo")
	}
	for i, r := range s {
		if _, ok := iupac[r]; !ok {
			return "", fmt.Errorf("invalid base %q at %d; allowed: A C G T R Y S W K M B D H V N", r, i+1)
		}
	}
	return s, nil
}

// RevComp returns the reverse-complement of an IUPAC sequence.
func RevComp(seq string) string {
	r := []rune(Normalize(seq))
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = comp(r[j]), comp(r[i])
	}
	if len(r)%2 == 1 {
		m := len(r) / 2
		r[m] = comp(r[m])
	}
	return string(r)
}

func comp(r rune) rune {
	if c, ok := complement[r]; ok {
		return c
	}
	return r
}

// Matches reports whether primer base p can bind genome base g (IUPAC-aware).
func Matches(p, g rune) bool {
	P, G := unicode.ToUpper(p), unicode.ToUpper(g)
	setP, okP := iupac[P]
	setG, okG := iupac[G]
	if !okP || !okG {
		return false
	}
	for _, a := range setP {
		if strings.ContainsRune(setG, a) {
			return true
		}
	}
	return false
}
