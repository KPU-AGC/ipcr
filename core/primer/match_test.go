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

func TestFindMatches_SoftmaskToggle(t *testing.T) {
	// Mixed-case reference window.
	seq := []byte("ACgtAC")
	pat := []byte("ACGTAC")

	// Default: lowercase reference bases reject the candidate immediately (not a mismatch).
	if hits := FindMatches(seq, pat, 10, 0, 0); len(hits) != 0 {
		t.Fatalf("FindMatches default: expected 0 hits, got %d", len(hits))
	}

	// Opt-in: lowercase behaves like uppercase for matching (case-insensitive).
	hits := FindMatchesSoftmask(seq, pat, 0, 0, 0)
	if len(hits) != 1 {
		t.Fatalf("FindMatchesSoftmask: expected 1 hit, got %d", len(hits))
	}
	if hits[0].Mismatches != 0 {
		t.Fatalf("FindMatchesSoftmask: expected 0 mismatches, got %d", hits[0].Mismatches)
	}
	if hits[0].Pos != 0 {
		t.Fatalf("FindMatchesSoftmask: expected Pos=0, got %d", hits[0].Pos)
	}

	// Lowercase should not be “paid for” via the mismatch budget in default mode.
	seq2 := []byte("AAAaA")
	pat2 := []byte("AAAAA")
	if hits := FindMatches(seq2, pat2, 1, 0, 0); len(hits) != 0 {
		t.Fatalf("FindMatches default: expected 0 hits with maxMM=1, got %d", len(hits))
	}
	if hits := FindMatchesSoftmask(seq2, pat2, 0, 0, 0); len(hits) != 1 || hits[0].Mismatches != 0 {
		t.Fatalf("FindMatchesSoftmask: expected 1 perfect hit, got %+v", hits)
	}
}
