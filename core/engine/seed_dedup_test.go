package engine

import (
	"testing"

	"ipcr-core/primer"
)

func TestBuildSeedsDeduplicatesPatternsAndFansOutPayloads(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "p1", Forward: "AAAACCCC", Reverse: "GGGGTTTT"},
		{ID: "p2", Forward: "AAAACCCC", Reverse: "GGGGTTTT"},
	}

	seeds, have := buildSeedPatterns(pairs, 4, 0, 0)
	if len(seeds) != 4 {
		t.Fatalf("unique seed pattern count = %d, want 4; seeds=%+v", len(seeds), seeds)
	}

	for i := range pairs {
		for _, which := range []byte{'A', 'B', 'a', 'b'} {
			if !have[i][which] {
				t.Fatalf("have[%d][%c] = false, want true", i, which)
			}
		}
	}

	payloadsByPattern := map[string]int{}
	for _, seed := range seeds {
		payloadsByPattern[string(seed.Pat)] = len(seed.Payloads)
	}
	for _, pattern := range []string{"CCCC", "TTTT", "GGGG", "AAAA"} {
		if payloadsByPattern[pattern] != len(pairs) {
			t.Fatalf("payload count for %s = %d, want %d", pattern, payloadsByPattern[pattern], len(pairs))
		}
	}
}

func TestBuildSeedsDeduplicatesApproximateVariantsAcrossPairs(t *testing.T) {
	pairs := []primer.Pair{
		{ID: "p1", Forward: "AAAACCCC", Reverse: "GGGGTTTT"},
		{ID: "p2", Forward: "AAAACCCC", Reverse: "GGGGTTTT"},
	}

	seeds, _ := buildSeedPatterns(pairs, 4, 0, 1)
	if len(seeds) != 52 { // four unique 4-mers, each with 1 + 4*3 variants.
		t.Fatalf("unique approximate seed pattern count = %d, want 52", len(seeds))
	}

	shared := 0
	for _, seed := range seeds {
		if len(seed.Payloads) > 1 {
			shared++
		}
	}
	if shared == 0 {
		t.Fatalf("expected at least one approximate seed pattern to share payloads")
	}
}

func TestEncodedSeedPatternRoundTrip(t *testing.T) {
	patterns := []string{
		"A",
		"ACGT",
		"AAAACCCCGGGGTTTT",
		"ACGTACGTACGTACGTACGTACGTACGTACGT",
	}
	for _, pattern := range patterns {
		encoded, ok := encodeSeedPattern([]byte(pattern))
		if !ok {
			t.Fatalf("encodeSeedPattern(%q) failed", pattern)
		}
		if got := string(decodeSeedPattern(encoded)); got != pattern {
			t.Fatalf("round trip for %q = %q", pattern, got)
		}
	}
}

func TestEncodedSeedPatternRejectsInvalidOrLongSeeds(t *testing.T) {
	if _, ok := encodeSeedPattern(nil); ok {
		t.Fatal("encodeSeedPattern accepted empty seed")
	}
	if _, ok := encodeSeedPattern([]byte("ACGN")); ok {
		t.Fatal("encodeSeedPattern accepted non-ACGT seed")
	}
	if _, ok := encodeSeedPattern([]byte("ACGTACGTACGTACGTACGTACGTACGTACGTA")); ok {
		t.Fatal("encodeSeedPattern accepted >32 bp seed")
	}
}

func TestBuildSeedsFallsBackForSeedLenOverEncodedLimit(t *testing.T) {
	pairs := []primer.Pair{{
		ID:      "long_seed",
		Forward: "ACGTACGTACGTACGTACGTACGTACGTACGTACGTACGT",
		Reverse: "TGCATGCATGCATGCATGCATGCATGCATGCATGCATGCA",
	}}

	seeds, have := buildSeedPatterns(pairs, 33, 0, 0)
	if len(seeds) != 0 {
		t.Fatalf("seed count = %d, want 0 for >32 bp encoded seed fallback", len(seeds))
	}
	if len(have) != 0 {
		t.Fatalf("have = %+v, want no seeded orientations for >32 bp encoded seed fallback", have)
	}
}
