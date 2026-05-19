package thermointegration

import (
	"bytes"
	"ipcr/internal/thermoapp"
	"os"
	"strings"
	"testing"
)

func writeFA(t *testing.T, name, data string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
}

func TestThermo_VersionDoesNotRequireInputs(t *testing.T) {
	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{"--version"}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s out=%s", code, errB.String(), out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("ipcr-thermo 5.0.0")) {
		t.Fatalf("expected ipcr-thermo version output, got:\n%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("thermo: nn-structure-v1")) {
		t.Fatalf("expected nn-structure-v1 component version, got:\n%s", out.String())
	}
}

func TestThermo_EndToEnd_TSVWithScore(t *testing.T) {
	// Tiny FASTA with a simple amplicon.
	fa := writeFA(t, "thermo_it.fa", ">s\nACGTACAAAAAAGGTACC\n")
	defer func() { _ = os.Remove(fa) }()

	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{
		"--forward", "ACGTAC",
		"--reverse", "GGTACC",
		"--sequences", fa,
		"--output", "text",
		"--sort",
		"--rank", "score",
	}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	s := out.String()
	if s == "" || s[0] == 0 {
		t.Fatalf("expected non-empty text output")
	}
	if !bytes.Contains(out.Bytes(), []byte("score")) {
		t.Fatalf("expected 'score' in header:\n%s", s)
	}
}

func TestThermo_EndToEnd_TSVWithThermoDetails(t *testing.T) {
	fa := writeFA(t, "thermo_details_it.fa", ">s\nACGTACGTACGTACGTACGTAAAAACGTACGTACGTACGTACGT\n")
	defer func() { _ = os.Remove(fa) }()

	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{
		"--forward", "ACGTACGTACGTACGTACGT",
		"--reverse", "ACGTACGTACGTACGTACGT",
		"--sequences", fa,
		"--output", "text",
		"--thermo-model", "nn-duplex-v1",
		"--thermo-details",
	}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	s := out.String()
	if !bytes.Contains(out.Bytes(), []byte("thermo_model\tsalt_model\tna_m\tmg_m\tdntp_m\teffective_na_m\tfree_mg_m\tanneal_temp_c")) {
		t.Fatalf("expected thermo details header:\n%s", s)
	}
	if !bytes.Contains(out.Bytes(), []byte("nn-duplex-v1")) {
		t.Fatalf("expected nn-duplex-v1 detail row:\n%s", s)
	}
}

func rc5to3IT(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		switch s[i] {
		case 'A':
			out[len(s)-1-i] = 'T'
		case 'C':
			out[len(s)-1-i] = 'G'
		case 'G':
			out[len(s)-1-i] = 'C'
		case 'T':
			out[len(s)-1-i] = 'A'
		default:
			out[len(s)-1-i] = 'N'
		}
	}
	return string(out)
}

func tsvColumnValue(t *testing.T, text, column string) string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header and row, got %q", text)
	}
	head := strings.Split(lines[0], "\t")
	row := strings.Split(lines[1], "\t")
	for i, h := range head {
		if h == column {
			if i >= len(row) {
				t.Fatalf("column %q index %d beyond row width %d", column, i, len(row))
			}
			return row[i]
		}
	}
	t.Fatalf("missing column %q in header %q", column, lines[0])
	return ""
}

func TestThermo_MismatchProvenanceAppearsInTSVJSONAndJSONL(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "TTGCGTATCGATCGTACGTA"
	leftSite := []byte(fwd)
	leftSite[6] = 'A' // G/T internal mismatch in the primer-template duplex.
	fa := writeFA(t, "thermo_mismatch_provenance.fa", ">s\n"+string(leftSite)+"AAAA"+rc5to3IT(rev)+"\n")
	defer func() { _ = os.Remove(fa) }()

	baseArgs := []string{
		"--forward", fwd,
		"--reverse", rev,
		"--sequences", fa,
		"--mismatches", "1",
		"--thermo-model", "nn-duplex-v1",
	}

	var tsvOut, tsvErr bytes.Buffer
	code := thermoapp.Run(append(append([]string{}, baseArgs...), "--output", "text", "--thermo-details"), &tsvOut, &tsvErr)
	if code != 0 {
		t.Fatalf("tsv exit %d err=%s", code, tsvErr.String())
	}
	tsv := tsvOut.String()
	if got := tsvColumnValue(t, tsv, "fwd_mismatch_sources"); got != "triplet-ddg" {
		t.Fatalf("fwd_mismatch_sources: got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_mismatch_parameter_sets"); got != "santalucia-hicks-2004-internal-mismatch-compiled-dimer-gauge-v1" {
		t.Fatalf("fwd_mismatch_parameter_sets: got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_mismatch_citations"); !strings.Contains(got, "SantaLucia & Hicks 2004") {
		t.Fatalf("fwd_mismatch_citations missing citation: got %q\n%s", got, tsv)
	}

	for _, format := range []string{"json", "jsonl"} {
		var out, errB bytes.Buffer
		code := thermoapp.Run(append(append([]string{}, baseArgs...), "--output", format), &out, &errB)
		if code != 0 {
			t.Fatalf("%s exit %d err=%s", format, code, errB.String())
		}
		if !bytes.Contains(out.Bytes(), []byte("mismatch_parameter_sets")) ||
			!bytes.Contains(out.Bytes(), []byte("santalucia-hicks-2004-internal-mismatch-compiled-dimer-gauge-v1")) ||
			!bytes.Contains(out.Bytes(), []byte("mismatch_sources")) {
			t.Fatalf("expected mismatch provenance in %s output:\n%s", format, out.String())
		}
	}
}

func TestThermo_TerminalMismatchProvenanceAppearsInTSVJSONAndJSONL(t *testing.T) {
	fwd := "ACGTACGTACGTACGTACGT"
	rev := "TTGCGTATCGATCGTACGTA"
	leftSite := []byte(fwd)
	leftSite[len(leftSite)-1] = 'A' // primer 3' terminal T/T mismatch after target-orientation conversion.
	fa := writeFA(t, "thermo_terminal_mismatch_provenance.fa", ">s\n"+string(leftSite)+"AAAA"+rc5to3IT(rev)+"\n")
	defer func() { _ = os.Remove(fa) }()

	baseArgs := []string{
		"--forward", fwd,
		"--reverse", rev,
		"--sequences", fa,
		"--mismatches", "1",
		"--terminal-window", "0",
		"--seed-length", "-1",
		"--thermo-model", "nn-duplex-v1",
	}

	var tsvOut, tsvErr bytes.Buffer
	code := thermoapp.Run(append(append([]string{}, baseArgs...), "--output", "text", "--thermo-details"), &tsvOut, &tsvErr)
	if code != 0 {
		t.Fatalf("tsv exit %d err=%s", code, tsvErr.String())
	}
	tsv := tsvOut.String()
	if got := tsvColumnValue(t, tsv, "fwd_terminal_mismatch_sources"); got != "ipcr-terminal-mismatch-heuristic" {
		t.Fatalf("fwd_terminal_mismatch_sources: got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_terminal_mismatch_parameter_sets"); got != "ipcr-terminal-mismatch-heuristic-v1" {
		t.Fatalf("fwd_terminal_mismatch_parameter_sets: got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_terminal_mismatch_citations"); !strings.Contains(got, "ipcr internal terminal-mismatch heuristic") {
		t.Fatalf("fwd_terminal_mismatch_citations missing heuristic citation: got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_terminal_mismatch_parameter_notes"); !strings.Contains(got, "Empirical fixed terminal mismatch") {
		t.Fatalf("fwd_terminal_mismatch_parameter_notes missing heuristic note: got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_terminal_mismatch_penalty_c"); got == "" || got == "0" {
		t.Fatalf("expected nonzero fwd_terminal_mismatch_penalty_c, got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_3p_terminal_mismatch_penalty_c"); got == "" || got == "0" {
		t.Fatalf("expected nonzero fwd_3p_terminal_mismatch_penalty_c, got %q\n%s", got, tsv)
	}
	if got := tsvColumnValue(t, tsv, "fwd_terminal_mismatch_dg_kcal"); got == "" || got == "0" {
		t.Fatalf("expected nonzero fwd_terminal_mismatch_dg_kcal, got %q\n%s", got, tsv)
	}

	for _, format := range []string{"json", "jsonl"} {
		var out, errB bytes.Buffer
		code := thermoapp.Run(append(append([]string{}, baseArgs...), "--output", format), &out, &errB)
		if code != 0 {
			t.Fatalf("%s exit %d err=%s", format, code, errB.String())
		}
		for _, want := range [][]byte{
			[]byte("terminal_mismatch_sources"),
			[]byte("ipcr-terminal-mismatch-heuristic"),
			[]byte("terminal_mismatch_parameter_sets"),
			[]byte("ipcr-terminal-mismatch-heuristic-v1"),
			[]byte("terminal_mismatch_citations"),
			[]byte("terminal_mismatch_parameter_notes"),
			[]byte("terminal_mismatch_count"),
			[]byte("terminal_mismatch_penalty_c"),
		} {
			if !bytes.Contains(out.Bytes(), want) {
				t.Fatalf("expected %q in %s output:\n%s", want, format, out.String())
			}
		}
	}
}

func TestThermo_ProbeThermoDefaultModelUsesNNStructure(t *testing.T) {
	fwd := "GCGCGCGCGCGCGCGCGCGC"
	rev := "CGCGCGCGCGCGCGCGCGCG"
	probe := "GCGCGATCGCGATCGCGCGC"
	fa := writeFA(t, "thermo_probe_auto_nn.fa", ">s\n"+fwd+"AAAA"+probe+"AAAA"+rc5to3IT(rev)+"\n")
	defer func() { _ = os.Remove(fa) }()

	var out, errB bytes.Buffer
	code := thermoapp.Run([]string{
		"--forward", fwd,
		"--reverse", rev,
		"--sequences", fa,
		"--probe", probe,
		"--probe-thermo",
		"--probe-min-margin", "-100",
		"--output", "text",
		"--thermo-details",
	}, &out, &errB)
	if code != 0 {
		t.Fatalf("exit %d err=%s", code, errB.String())
	}
	s := out.String()
	if !bytes.Contains(out.Bytes(), []byte("nn-structure-v1")) {
		t.Fatalf("expected default nn-structure-v1 scoring when --probe-thermo is used:\n%s", s)
	}
	if !bytes.Contains(out.Bytes(), []byte("probe_found")) || !bytes.Contains(out.Bytes(), []byte("\ttrue\tgate\tprobe\t")) {
		t.Fatalf("expected populated probe thermo detail columns:\n%s", s)
	}
}
