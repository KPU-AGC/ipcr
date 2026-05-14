package primer

import (
	"fmt"
	"strings"
	"unicode"
)

// Normalize removes surrounding/embedded whitespace and quotes, then uppercases
// an oligo/primer sequence. It intentionally does not reinterpret unsupported
// bases: validation is handled by Validate.
func Normalize(raw string) string {
	out := make([]rune, 0, len(raw))
	for _, r := range raw {
		if unicode.IsSpace(r) || r == '\'' || r == '"' {
			continue
		}
		out = append(out, unicode.ToUpper(r))
	}
	return string(out)
}

// Validate returns a normalized IUPAC DNA primer sequence, or an error when an
// unsupported character is present. Only DNA IUPAC symbols are accepted; U is
// rejected rather than silently treated as T at user-input boundaries.
func Validate(raw string) (string, error) {
	s := Normalize(raw)
	if s == "" {
		return "", fmt.Errorf("empty primer")
	}
	for i, r := range s {
		if !strings.ContainsRune("ACGTRYSWKMBDHVN", r) {
			return "", fmt.Errorf("invalid primer base %q at position %d; allowed: A C G T R Y S W K M B D H V N", r, i+1)
		}
	}
	return s, nil
}
