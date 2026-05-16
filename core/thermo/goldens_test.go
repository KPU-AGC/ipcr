package thermo

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
	"testing"
)

func readGoldenTSV(t *testing.T, path string) []map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.Comma = '\t'
	r.Comment = '#'
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(rows) < 2 {
		t.Fatalf("%s: expected header and at least one row", path)
	}
	header := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		m := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		out = append(out, m)
	}
	return out
}

func goldenFloat(t *testing.T, row map[string]string, key string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(row[key], 64)
	if err != nil {
		t.Fatalf("parse %s=%q: %v", key, row[key], err)
	}
	return v
}

func goldenInt(t *testing.T, row map[string]string, key string) int {
	t.Helper()
	v, err := strconv.Atoi(row[key])
	if err != nil {
		t.Fatalf("parse %s=%q: %v", key, row[key], err)
	}
	return v
}

func goldenConditions(t *testing.T, row map[string]string) Conditions {
	t.Helper()
	model, err := ParseSaltModel(row["salt_model"])
	if err != nil {
		t.Fatalf("ParseSaltModel(%q): %v", row["salt_model"], err)
	}
	return Conditions{
		AnnealC:      goldenFloat(t, row, "anneal_c"),
		NaM:          goldenFloat(t, row, "na_m"),
		MgM:          goldenFloat(t, row, "mg_m"),
		DntpM:        goldenFloat(t, row, "dntp_m"),
		PrimerTotalM: goldenFloat(t, row, "primer_total_m"),
		SaltModel:    model,
	}
}

func assertNearGolden(t *testing.T, label string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s: got %.15g want %.15g tolerance %.3g", label, got, want, tol)
	}
}

func TestGoldenPerfectDuplexes(t *testing.T) {
	for _, row := range readGoldenTSV(t, "testdata/perfect_duplex_goldens.golden") {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			got, err := PerfectDuplex(row["seq"], row["target3to5"], goldenConditions(t, row))
			if err != nil {
				t.Fatalf("PerfectDuplex: %v", err)
			}
			tol := goldenFloat(t, row, "tolerance")
			assertNearGolden(t, "tm_c", got.TmC, goldenFloat(t, row, "tm_c"), tol)
			assertNearGolden(t, "margin_c", got.AnnealMarginC, goldenFloat(t, row, "margin_c"), tol)
			assertNearGolden(t, "dg_kcal", got.DeltaGAtAnnealKcal, goldenFloat(t, row, "dg_kcal"), tol)
		})
	}
}

func TestGoldenSaltModels(t *testing.T) {
	for _, row := range readGoldenTSV(t, "testdata/salt_goldens.golden") {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			cond := goldenConditions(t, row)
			got, err := PerfectDuplex(row["seq"], row["target3to5"], cond)
			if err != nil {
				t.Fatalf("PerfectDuplex: %v", err)
			}
			tol := goldenFloat(t, row, "tolerance")
			assertNearGolden(t, "effective_na_m", cond.EffectiveNaM(), goldenFloat(t, row, "effective_na_m"), tol)
			assertNearGolden(t, "free_mg_m", cond.FreeMgM(), goldenFloat(t, row, "free_mg_m"), tol)
			assertNearGolden(t, "tm_c", got.TmC, goldenFloat(t, row, "tm_c"), tol)
			assertNearGolden(t, "margin_c", got.AnnealMarginC, goldenFloat(t, row, "margin_c"), tol)
			assertNearGolden(t, "dg_kcal", got.DeltaGAtAnnealKcal, goldenFloat(t, row, "dg_kcal"), tol)
		})
	}
}

func TestGoldenImperfectDuplexes(t *testing.T) {
	for _, row := range readGoldenTSV(t, "testdata/mismatch_goldens.golden") {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			got, err := ImperfectDuplex(row["primer"], row["target3to5"], goldenConditions(t, row))
			if err != nil {
				t.Fatalf("ImperfectDuplex: %v", err)
			}
			tol := goldenFloat(t, row, "tolerance")
			assertNearGolden(t, "tm_c", got.TmC, goldenFloat(t, row, "tm_c"), tol)
			assertNearGolden(t, "mismatch_penalty_c", got.MismatchPenaltyC, goldenFloat(t, row, "mismatch_penalty_c"), tol)
			assertNearGolden(t, "dg_penalty_kcal", got.DeltaGPenaltyKcal, goldenFloat(t, row, "dg_penalty_kcal"), tol)
			if got.MismatchCount != goldenInt(t, row, "mismatch_count") ||
				got.FivePrimeMismatchCount != goldenInt(t, row, "five_prime_count") ||
				got.ThreePrimeMismatchCount != goldenInt(t, row, "three_prime_count") ||
				got.TerminalMismatchCount != goldenInt(t, row, "terminal_count") ||
				got.HeuristicFallbackCount+got.DefaultFallbackCount != goldenInt(t, row, "fallback_count") ||
				got.TripletTmCount+got.TripletDeltaGCount != goldenInt(t, row, "triplet_count") {
				t.Fatalf("mismatch counts changed: got %+v row %+v", got, row)
			}
			if got.MismatchPolicy != row["policy"] {
				t.Fatalf("policy: got %q want %q", got.MismatchPolicy, row["policy"])
			}
		})
	}
}

func TestGoldenStructures(t *testing.T) {
	for _, row := range readGoldenTSV(t, "testdata/structure_goldens.golden") {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			opts := DefaultStructureOptions(DefaultConditions())
			var got StructureResult
			var ok bool
			var err error
			switch row["mode"] {
			case "hairpin":
				got, ok, err = BestHairpinV2(row["seq_a"], opts)
			case "cross":
				got, ok, err = BestCrossDimerV2(row["seq_a"], row["seq_b"], opts)
			default:
				t.Fatalf("unknown structure mode %q", row["mode"])
			}
			if err != nil {
				t.Fatalf("structure scoring: %v", err)
			}
			if !ok {
				t.Fatalf("expected structure candidate")
			}
			tol := goldenFloat(t, row, "tolerance")
			if got.Kind != row["kind"] {
				t.Fatalf("kind: got %q want %q", got.Kind, row["kind"])
			}
			assertNearGolden(t, "tm_c", got.TmC, goldenFloat(t, row, "tm_c"), tol)
			assertNearGolden(t, "dg_kcal", got.DeltaGAtAnnealKcal, goldenFloat(t, row, "dg_kcal"), tol)
			if got.StemLen != goldenInt(t, row, "stem_len") || got.LoopLen != goldenInt(t, row, "loop_len") || got.BulgeCount != goldenInt(t, row, "bulge_count") || got.InternalLoopCount != goldenInt(t, row, "internal_loop_count") {
				t.Fatalf("structure counts changed: got %+v row %+v", got, row)
			}
		})
	}
}
