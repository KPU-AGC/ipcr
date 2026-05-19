// core/engine/approx_seed_oracle_test.go
package engine

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"ipcr-core/primer"
)

type productOracleSig struct {
	SequenceID string
	ID         string
	Start      int
	End        int
	Length     int
	Type       string
	FwdMM      int
	RevMM      int
	FwdMMIdx   string
	RevMMIdx   string
}

func productMultiset(products []Product) map[productOracleSig]int {
	out := make(map[productOracleSig]int, len(products))
	for _, p := range products {
		out[productOracleSig{
			SequenceID: p.SequenceID,
			ID:         p.ExperimentID,
			Start:      p.Start,
			End:        p.End,
			Length:     p.Length,
			Type:       p.Type,
			FwdMM:      p.FwdMM,
			RevMM:      p.RevMM,
			FwdMMIdx:   intsKey(p.FwdMismatchIdx),
			RevMMIdx:   intsKey(p.RevMismatchIdx),
		}]++
	}
	return out
}

func intsKey(v []int) string {
	if len(v) == 0 {
		return ""
	}
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = fmt.Sprint(n)
	}
	return strings.Join(parts, ",")
}

func assertProductMultisetEqual(t *testing.T, got, want []Product) {
	t.Helper()

	gotSet := productMultiset(got)
	wantSet := productMultiset(want)
	if len(gotSet) == len(wantSet) {
		equal := true
		for sig, gotN := range gotSet {
			if wantSet[sig] != gotN {
				equal = false
				break
			}
		}
		if equal {
			return
		}
	}

	t.Fatalf("product multiset mismatch\nfast (%d):\n%s\nbrute (%d):\n%s",
		len(got), formatProductMultiset(gotSet), len(want), formatProductMultiset(wantSet))
}

func formatProductMultiset(ms map[productOracleSig]int) string {
	lines := make([]string, 0, len(ms))
	for sig, n := range ms {
		lines = append(lines, fmt.Sprintf("%dx\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s",
			n, sig.SequenceID, sig.ID, sig.Start, sig.End, sig.Length, sig.Type,
			sig.FwdMM, sig.RevMM, sig.FwdMMIdx, sig.RevMMIdx))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func TestSimulateBatchMatchesBruteForceOracle(t *testing.T) {
	type oracleCase struct {
		name  string
		seq   []byte
		pairs []primer.Pair
	}

	cases := []oracleCase{
		{
			name: "forward exact unambiguous",
			seq:  []byte("TTTACGTACAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "forward_exact",
				Forward: "ACGTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "revcomp exact unambiguous",
			seq:  []byte("TTTGGTACCAAAAGTACGTTTT"),
			pairs: []primer.Pair{{
				ID:      "revcomp_exact",
				Forward: "ACGTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "forward mismatch outside protected 3 prime window",
			seq:  []byte("TTTTCGTACAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "forward_mismatch_5prime",
				Forward: "ACGTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "primer RY ambiguity",
			seq:  []byte("TTTACGTACAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "primer_ry",
				Forward: "ACRTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "primer internal N ambiguity",
			seq:  []byte("TTTACGTACAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "primer_internal_n",
				Forward: "ACNTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "primer 3 prime N ambiguity",
			seq:  []byte("TTTACGTACAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "primer_3prime_n",
				Forward: "ACGTAN",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "reference N outside protected terminal window",
			seq:  []byte("TTTNCGTACAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "reference_n",
				Forward: "ACGTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "lowercase reference bases",
			seq:  []byte("TTTacgtacAAAAGGTACCTTT"),
			pairs: []primer.Pair{{
				ID:      "lowercase_reference",
				Forward: "ACGTAC",
				Reverse: "GGTACC",
			}},
		},
		{
			name: "mixed panel with decoy",
			seq:  []byte("TTTACGTACAAAAGGTACCTTTGGGGGGGGGG"),
			pairs: []primer.Pair{
				{ID: "panel_hit", Forward: "ACGTAC", Reverse: "GGTACC"},
				{ID: "panel_decoy", Forward: "TTAACC", Reverse: "CCAATT"},
			},
		},
	}

	configs := []struct {
		name string
		cfg  Config
	}{
		{name: "mm0_tw0", cfg: Config{MaxMM: 0, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm0_tw1", cfg: Config{MaxMM: 0, TerminalWindow: 1, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm0_tw3", cfg: Config{MaxMM: 0, TerminalWindow: 3, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm1_tw0", cfg: Config{MaxMM: 1, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm1_tw1", cfg: Config{MaxMM: 1, TerminalWindow: 1, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm1_tw3", cfg: Config{MaxMM: 1, TerminalWindow: 3, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm2_tw0", cfg: Config{MaxMM: 2, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm2_tw1", cfg: Config{MaxMM: 2, TerminalWindow: 1, MinLen: 1, MaxLen: 100, SeedLen: 12}},
		{name: "mm2_tw3", cfg: Config{MaxMM: 2, TerminalWindow: 3, MinLen: 1, MaxLen: 100, SeedLen: 12}},
	}

	for _, tc := range cases {
		for _, cc := range configs {
			t.Run(tc.name+"/"+cc.name, func(t *testing.T) {
				eng := New(cc.cfg)
				fast := eng.SimulateBatch("seq", tc.seq, tc.pairs)
				brute := eng.SimulateBatchBruteForce("seq", tc.seq, tc.pairs)
				assertProductMultisetEqual(t, fast, brute)
			})
		}
	}
}

func TestSimulateBatchOracleLengthBoundaryCases(t *testing.T) {
	seq := []byte("TTTACGTACAAAAGGTACCTTT")
	pairs := []primer.Pair{{ID: "length", Forward: "ACGTAC", Reverse: "GGTACC"}}

	configs := []struct {
		name string
		cfg  Config
	}{
		{name: "exact_min_and_max", cfg: Config{MaxMM: 0, TerminalWindow: 0, MinLen: 16, MaxLen: 16, SeedLen: 12}},
		{name: "below_min_excluded", cfg: Config{MaxMM: 0, TerminalWindow: 0, MinLen: 17, MaxLen: 100, SeedLen: 12}},
		{name: "above_max_excluded", cfg: Config{MaxMM: 0, TerminalWindow: 0, MinLen: 1, MaxLen: 15, SeedLen: 12}},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			eng := New(tc.cfg)
			fast := eng.SimulateBatch("seq", seq, pairs)
			brute := eng.SimulateBatchBruteForce("seq", seq, pairs)
			assertProductMultisetEqual(t, fast, brute)
		})
	}
}

func TestSimulateBatchOracleCircularCases(t *testing.T) {
	seq := []byte("TGACAAG")
	pairs := []primer.Pair{{ID: "circular", Forward: "AG", Reverse: "TC"}}

	for _, circular := range []bool{false, true} {
		t.Run(fmt.Sprintf("circular_%t", circular), func(t *testing.T) {
			eng := New(Config{Circular: circular, MaxMM: 0, TerminalWindow: 0, MinLen: 0, MaxLen: 0})
			fast := eng.SimulateBatch("seq", seq, pairs)
			brute := eng.SimulateBatchBruteForce("seq", seq, pairs)
			assertProductMultisetEqual(t, fast, brute)
		})
	}
}

func TestSimulateSelfMatchesBruteForceOracle(t *testing.T) {
	seq := []byte("TTTACGTACAAAAGTACGTTTT")
	oligos := []primer.Oligo{{ID: "self", Seq: "ACGTAC"}}
	pairs := primer.SelfPairs(oligos)

	for _, cfg := range []Config{
		{MaxMM: 0, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 12},
		{MaxMM: 1, TerminalWindow: 3, MinLen: 1, MaxLen: 100, SeedLen: 12},
		{MaxMM: 2, TerminalWindow: 0, MinLen: 1, MaxLen: 100, SeedLen: 12},
	} {
		t.Run(fmt.Sprintf("mm%d_tw%d", cfg.MaxMM, cfg.TerminalWindow), func(t *testing.T) {
			eng := New(cfg)
			fast := eng.SimulateSelf("seq", seq, oligos)
			brute := eng.SimulateBatchBruteForce("seq", seq, pairs)
			assertProductMultisetEqual(t, fast, brute)
		})
	}
}
