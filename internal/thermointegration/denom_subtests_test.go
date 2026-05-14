package thermointegration

import (
	"bytes"
	"ipcr/internal/thermoapp"
	"os"
	"strconv"
	"strings"
	"testing"
)

func writeFA2(t *testing.T, name, data string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func runThermo(t *testing.T, argv []string) string {
	t.Helper()
	var out, errB bytes.Buffer
	code := thermoapp.Run(argv, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	return out.String()
}

func firstScoreFromTSV(t *testing.T, text string) float64 {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) < 2 {
		t.Fatalf("unexpected output:\n%s", text)
	}
	header := strings.Split(lines[0], "\t")
	scoreIdx := -1
	for i, h := range header {
		if h == "score" {
			scoreIdx = i
			break
		}
	}
	if scoreIdx == -1 {
		t.Fatalf("no 'score' column in header: %q", lines[0])
	}
	row := strings.Split(lines[1], "\t")
	if scoreIdx >= len(row) {
		t.Fatalf("row shorter than header: %v vs %v", row, header)
	}
	sc, err := strconv.ParseFloat(row[scoreIdx], 64)
	if err != nil {
		t.Fatalf("parse score: %v", err)
	}
	return sc
}

func TestThermo_DenomMode_Subtests(t *testing.T) {
	// Simple amplicon: forward=A?GTAC (1 internal mismatch), reverse=GGTACC (perfect).
	// Seed the last 3 nt so the mismatch doesn't kill seeding; allow 1 mismatch.
	fa := writeFA2(t, "thermo_seed.fa", ">s\nACGTACAAAAAAGGTACC\n")
	t.Cleanup(func() { _ = os.Remove(fa) })

	baseArgs := []string{
		"--forward", "AAGTAC", // 1 mismatch vs ACGTAC at pos 2 (seed = TAC still exact)
		"--reverse", "GGTACC",
		"--sequences", fa,
		"--output", "text",
		"--sort",
		"--rank", "score",
		"--mismatches", "1",
		"--seed-length", "3",
		"--terminal-window", "0",
		"--self=false",
	}

	t.Run("fixed_vs_auto_changes_score", func(t *testing.T) {
		outFixed := runThermo(t, append([]string{}, baseArgs...)) // default: --denom fixed
		sFixed := firstScoreFromTSV(t, outFixed)

		outAuto := runThermo(t, append(append([]string{}, baseArgs...), "--denom", "auto"))
		sAuto := firstScoreFromTSV(t, outAuto)

		if sFixed == sAuto {
			t.Fatalf("expected different scores: fixed=%g auto=%g", sFixed, sAuto)
		}
	})

	t.Run("fixed_ignores_solution_conditions", func(t *testing.T) {
		low := append(append([]string{}, baseArgs...), "--na", "10mM", "--primer-conc", "100nM")
		high := append(append([]string{}, baseArgs...), "--na", "200mM", "--primer-conc", "1uM")

		sLow := firstScoreFromTSV(t, runThermo(t, low))
		sHigh := firstScoreFromTSV(t, runThermo(t, high))

		if sLow != sHigh {
			t.Fatalf("with --denom fixed, scores should be equal; got %g vs %g", sLow, sHigh)
		}
	})

	t.Run("auto_reflects_solution_conditions", func(t *testing.T) {
		auto := append(append([]string{}, baseArgs...), "--denom", "auto")

		low := append(append([]string{}, auto...), "--na", "10mM", "--primer-conc", "100nM")
		high := append(append([]string{}, auto...), "--na", "200mM", "--primer-conc", "1uM")

		sLow := firstScoreFromTSV(t, runThermo(t, low))
		sHigh := firstScoreFromTSV(t, runThermo(t, high))

		if sLow == sHigh {
			t.Fatalf("with --denom auto, scores should differ across conditions; got %g vs %g", sLow, sHigh)
		}
	})
}

func TestThermo_LegacyModelGoldenOutput(t *testing.T) {
	fa := writeFA2(t, "thermo_stage0.fa", ">s\nACGTACAAAAAAGGTACC\n")
	t.Cleanup(func() { _ = os.Remove(fa) })

	baseArgs := []string{
		"--forward", "AAGTAC",
		"--reverse", "GGTACC",
		"--sequences", fa,
		"--output", "text",
		"--sort",
		"--rank", "score",
		"--mismatches", "1",
		"--seed-length", "3",
		"--terminal-window", "0",
		"--self=false",
	}

	want := "source_file\tsequence_id\texperiment_id\tstart\tend\tlength\ttype\tfwd_mm\trev_mm\tfwd_mm_i\trev_mm_i\tscore\n" +
		"thermo_stage0.fa\ts\tmanual\t0\t18\t18\tforward\t1\t0\t1\t\t-6.375\n" +
		"thermo_stage0.fa\ts\tmanual\t11\t18\t7\tforward\t1\t0\t1\t\t-8.25\n"

	gotDefault := runThermo(t, baseArgs)
	if gotDefault != want {
		t.Fatalf("legacy default output changed:\ngot:\n%s\nwant:\n%s", gotDefault, want)
	}

	gotExplicit := runThermo(t, append(append([]string{}, baseArgs...), "--thermo-model", "legacy-heuristic"))
	if gotExplicit != want {
		t.Fatalf("explicit legacy output changed:\ngot:\n%s\nwant:\n%s", gotExplicit, want)
	}
}

func TestThermo_NNDuplexModelAcceptedAndTemperatureAware(t *testing.T) {
	fa := writeFA2(t, "thermo_nn_duplex.fa", ">s\nACGTACGTACGTACGTACGTAAAAAAACGTACGTACGTACGTACGT\n")
	t.Cleanup(func() { _ = os.Remove(fa) })

	baseArgs := []string{
		"--forward", "ACGTACGTACGTACGTACGT",
		"--reverse", "ACGTACGTACGTACGTACGT",
		"--sequences", fa,
		"--output", "text",
		"--rank", "score",
		"--self=false",
		"--thermo-model", "nn-duplex-v1",
	}

	lowAnneal := firstScoreFromTSV(t, runThermo(t, append(append([]string{}, baseArgs...), "--anneal-temp", "55")))
	highAnneal := firstScoreFromTSV(t, runThermo(t, append(append([]string{}, baseArgs...), "--anneal-temp", "70")))

	if !(lowAnneal > highAnneal) {
		t.Fatalf("expected lower anneal temp to improve NN margin score; 55C=%g 70C=%g", lowAnneal, highAnneal)
	}
}

func TestThermo_NNDuplexJSONIncludesThermoComponents(t *testing.T) {
	fa := writeFA2(t, "thermo_nn_duplex_json.fa", ">s\nACGTACGTACGTACGTACGTAAAAAAACGTACGTACGTACGTACGT\n")
	t.Cleanup(func() { _ = os.Remove(fa) })

	out := runThermo(t, []string{
		"--forward", "ACGTACGTACGTACGTACGT",
		"--reverse", "ACGTACGTACGTACGTACGT",
		"--sequences", fa,
		"--output", "json",
		"--sort",
		"--self=false",
		"--thermo-model", "nn-duplex-v1",
	})
	for _, want := range []string{`"thermo"`, `"model": "nn-duplex-v1"`, `"anneal_margin_c"`, `"delta_g_at_anneal_kcal"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected JSON output to contain %s; got:\n%s", want, out)
		}
	}
}

func TestThermo_NNDuplexModelRejectsDegeneratePrimers(t *testing.T) {
	fa := writeFA2(t, "thermo_nn_duplex_iupac.fa", ">s\nACGTACAAAAAAGGTACC\n")
	t.Cleanup(func() { _ = os.Remove(fa) })

	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{
		"--forward", "ACGRAC",
		"--reverse", "GGTACC",
		"--sequences", fa,
		"--thermo-model", "nn-duplex-v1",
	}, &out, &errB)
	if code != 2 {
		t.Fatalf("expected exit 2 for strict NN IUPAC policy, got %d stdout=%q stderr=%q", code, out.String(), errB.String())
	}
	if !strings.Contains(errB.String(), "strict A/C/G/T") {
		t.Fatalf("unexpected stderr: %q", errB.String())
	}
}

func TestThermo_FutureStructureModelRejectedAtCLI(t *testing.T) {
	fa := writeFA2(t, "thermo_future_model.fa", ">s\nACGTACAAAAAAGGTACC\n")
	t.Cleanup(func() { _ = os.Remove(fa) })

	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{
		"--forward", "ACGTAC",
		"--reverse", "GGTACC",
		"--sequences", fa,
		"--thermo-model", "nn-structure-v1",
	}, &out, &errB)
	if code != 2 {
		t.Fatalf("expected exit 2 for reserved model, got %d stdout=%q stderr=%q", code, out.String(), errB.String())
	}
	if !strings.Contains(errB.String(), "not implemented yet") {
		t.Fatalf("unexpected stderr: %q", errB.String())
	}
}
