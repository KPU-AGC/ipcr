// core/primer/match_test.go
package primer

import "testing"

func TestFindMatches(t *testing.T) {
	seq := []byte("ACGTACGTACGT")

	tests := []struct {
		name         string
		primer       string
		maxMM        int
		termWin      int
		wantCount    int
		wantFirstPos int
	}{
		{
			name:         "perfect match",
			primer:       "ACG",
			maxMM:        0,
			termWin:      0,
			wantCount:    3,
			wantFirstPos: 0,
		},
		{
			name:         "one mismatch allowed",
			primer:       "AGG",
			maxMM:        1,
			termWin:      0,
			wantCount:    3,
			wantFirstPos: 0,
		},
		{
			name:         "exceed mismatch threshold",
			primer:       "AGG",
			maxMM:        0,
			termWin:      0,
			wantCount:    0,
			wantFirstPos: -1,
		},
		{
			name:         "3prime mismatch disallowed (window=1)",
			primer:       "ACA", // 3' A mismatches genome G in every window
			maxMM:        1,
			termWin:      1,
			wantCount:    0,
			wantFirstPos: -1,
		},
		{
			name:         "3prime mismatch allowed (window=0)",
			primer:       "ACG",
			maxMM:        1,
			termWin:      0,
			wantCount:    3,
			wantFirstPos: 0,
		},
		{
			name:         "IUPAC degeneracy",
			primer:       "ACN",
			maxMM:        0,
			termWin:      0,
			wantCount:    3,
			wantFirstPos: 0,
		},
	}

	for _, tc := range tests {
		hits := FindMatches(seq, []byte(tc.primer), tc.maxMM, 0, tc.termWin)
		if len(hits) != tc.wantCount {
			t.Errorf("%s: got %d hits, want %d", tc.name, len(hits), tc.wantCount)
		}
		if tc.wantCount > 0 && tc.wantFirstPos != -1 && hits[0].Pos != tc.wantFirstPos {
			t.Errorf("%s: first match pos %d, want %d", tc.name, hits[0].Pos, tc.wantFirstPos)
		}
	}
}
