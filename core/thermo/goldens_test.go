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

func goldenBase(t *testing.T, row map[string]string, key string) byte {
	t.Helper()
	v := row[key]
	if len(v) != 1 {
		t.Fatalf("%s: expected one base, got %q", key, v)
	}
	return v[0]
}

func goldenOptionalBase(row map[string]string, key string) byte {
	v := row[key]
	if v == "" {
		return 0
	}
	return []byte(v)[0]
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

func TestGoldenMismatchTriplets(t *testing.T) {
	for _, row := range readGoldenTSV(t, "testdata/mismatch_triplet_goldens.golden") {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			model, err := ParseSaltModel(row["salt_model"])
			if err != nil {
				t.Fatalf("ParseSaltModel(%q): %v", row["salt_model"], err)
			}
			cond := Conditions{
				AnnealC:      goldenFloat(t, row, "anneal_c"),
				NaM:          goldenFloat(t, row, "na_m"),
				MgM:          goldenFloat(t, row, "mg_m"),
				DntpM:        goldenFloat(t, row, "dntp_m"),
				PrimerTotalM: DefaultConditions().PrimerTotalM,
				SaltModel:    model,
			}

			got, err := ImperfectDuplex(row["primer"], row["target"], cond)
			if err != nil {
				t.Fatalf("ImperfectDuplex: %v", err)
			}
			tol := goldenFloat(t, row, "tolerance_delta_g")
			assertNearGolden(t, "delta_delta_g", got.DeltaGPenaltyKcal, goldenFloat(t, row, "expected_delta_delta_g_kcal"), tol)

			if got.MismatchCount != goldenInt(t, row, "expected_mismatch_count") {
				t.Fatalf("mismatch count: got %d want %s", got.MismatchCount, row["expected_mismatch_count"])
			}
			tripletCount := got.TripletTmCount + got.TripletDeltaGCount
			if tripletCount != goldenInt(t, row, "expected_triplet_count") {
				t.Fatalf("triplet count: got %d want %s; result=%+v", tripletCount, row["expected_triplet_count"], got)
			}
			fallbackCount := got.HeuristicFallbackCount + got.DefaultFallbackCount
			if fallbackCount != goldenInt(t, row, "expected_fallback_count") {
				t.Fatalf("fallback count: got %d want %s; result=%+v", fallbackCount, row["expected_fallback_count"], got)
			}
			if got.MismatchPolicy != MismatchPolicyImperfectTriplet {
				t.Fatalf("policy: got %q want %q", got.MismatchPolicy, MismatchPolicyImperfectTriplet)
			}

			perfectTarget, ok := compStrict(row["primer"])
			if !ok {
				t.Fatalf("compStrict failed for %q", row["primer"])
			}
			perfect, err := PerfectDuplex(row["primer"], perfectTarget, cond)
			if err != nil {
				t.Fatalf("PerfectDuplex: %v", err)
			}
			switch row["expected_tm_direction"] {
			case "decrease":
				if !(got.TmC < perfect.TmC) {
					t.Fatalf("expected mismatch to decrease Tm: perfect=%g got=%g", perfect.TmC, got.TmC)
				}
			case "increase":
				if !(got.TmC > perfect.TmC) {
					t.Fatalf("expected mismatch to increase Tm: perfect=%g got=%g", perfect.TmC, got.TmC)
				}
			default:
				t.Fatalf("unknown expected_tm_direction %q", row["expected_tm_direction"])
			}

			if len(got.Contributions) != 1 {
				t.Fatalf("expected one mismatch contribution, got %d: %+v", len(got.Contributions), got.Contributions)
			}
			c := got.Contributions[0]
			key := MismatchKey{P5: c.P5, P: c.PrimerBase, P3: c.P3, T5: c.T5, T: c.TargetBase, T3: c.T3}
			param, ok := LookupMismatchParameterInfo(key)
			if !ok {
				t.Fatalf("missing parameter info for %+v", key)
			}
			if param.Source != MismatchSourceTripletDeltaG {
				t.Fatalf("source: got %q want %q for %+v", param.Source, MismatchSourceTripletDeltaG, key)
			}
			if param.ParameterSet != row["expected_parameter_set"] {
				t.Fatalf("parameter set: got %q want %q", param.ParameterSet, row["expected_parameter_set"])
			}
		})
	}
}

func TestGoldenDanglingEnds(t *testing.T) {
	rows := readGoldenTSV(t, "testdata/dangling_end_goldens.golden")
	if len(rows) != len(CuratedDanglingEndParameters) {
		t.Fatalf("dangling-end golden count: got %d want %d", len(rows), len(CuratedDanglingEndParameters))
	}
	for _, row := range rows {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			var strandEnd byte
			switch row["template_end"] {
			case "5p":
				strandEnd = DanglingEndStrand5Prime
			case "3p":
				strandEnd = DanglingEndStrand3Prime
			default:
				t.Fatalf("unknown template_end %q", row["template_end"])
			}

			param, ok := LookupDanglingEndParameter(DanglingEndKey{
				StrandEnd:    strandEnd,
				DanglingBase: goldenBase(t, row, "dangling_base"),
				PairedBase:   goldenBase(t, row, "terminal_target_base"),
				OppositeBase: goldenBase(t, row, "terminal_primer_base"),
			})
			if !ok {
				t.Fatalf("missing dangling-end parameter for row %+v", row)
			}
			tol := goldenFloat(t, row, "tolerance")
			assertNearGolden(t, "delta_h", param.DeltaHkcal, goldenFloat(t, row, "expected_delta_h_kcal"), tol)
			assertNearGolden(t, "delta_s", param.DeltaScalK, goldenFloat(t, row, "expected_delta_s_cal_k"), tol)
			assertNearGolden(t, "delta_g37", param.DeltaG37kcal, goldenFloat(t, row, "expected_delta_g37_kcal"), tol)
			if param.ParameterSet != row["expected_parameter_set"] || param.Source != row["source_id"] || param.Citation == "" || param.Note == "" {
				t.Fatalf("parameter provenance missing/changed: %+v", param)
			}
		})
	}
}

func TestGoldenDanglingEndContexts(t *testing.T) {
	for _, row := range readGoldenTSV(t, "testdata/dangling_end_context_goldens.golden") {
		row := row
		t.Run(row["id"], func(t *testing.T) {
			cond := goldenConditions(t, row)
			base, err := ImperfectDuplex(row["primer"], row["target3to5"], cond)
			if err != nil {
				t.Fatalf("base ImperfectDuplex: %v", err)
			}
			got, err := ImperfectDuplexWithOptionsAndContext(
				row["primer"],
				row["target3to5"],
				cond,
				DefaultImperfectDuplexOptions(),
				DanglingEndContext{
					FivePrimeBase:  goldenOptionalBase(row, "five_prime_base"),
					ThreePrimeBase: goldenOptionalBase(row, "three_prime_base"),
				},
			)
			if err != nil {
				t.Fatalf("dangling ImperfectDuplex: %v", err)
			}
			tol := goldenFloat(t, row, "tolerance_delta_g")
			assertNearGolden(t, "dangling_delta_g", got.DanglingEndDeltaGKcal, goldenFloat(t, row, "expected_delta_g_kcal"), tol)
			if got.DanglingEndCount != goldenInt(t, row, "expected_dangling_count") || len(got.DanglingContributions) != goldenInt(t, row, "expected_dangling_count") {
				t.Fatalf("dangling count: got result=%+v row=%+v", got, row)
			}
			for _, c := range got.DanglingContributions {
				if c.ParameterSet != row["expected_parameter_set"] || c.Source != row["source_id"] || c.Citation == "" || c.ParameterNote == "" {
					t.Fatalf("dangling provenance missing/changed: %+v", c)
				}
			}
			switch row["expected_tm_direction"] {
			case "increase":
				if !(got.TmC > base.TmC && got.DeltaGAtAnnealKcal < base.DeltaGAtAnnealKcal) {
					t.Fatalf("expected dangling end to stabilize endpoint: base=%+v got=%+v", base.DuplexResult, got.DuplexResult)
				}
			case "decrease":
				if !(got.TmC < base.TmC && got.DeltaGAtAnnealKcal > base.DeltaGAtAnnealKcal) {
					t.Fatalf("expected dangling end to destabilize endpoint: base=%+v got=%+v", base.DuplexResult, got.DuplexResult)
				}
			default:
				t.Fatalf("unknown expected_tm_direction %q", row["expected_tm_direction"])
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
