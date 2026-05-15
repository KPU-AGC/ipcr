package writers

import (
	"bytes"
	"ipcr-core/engine"
	"strings"
	"testing"
)

func TestProductWriter_TSVScoreHeaderAndSort(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartProductWriter(&buf, "text", true, true, false, true, true, 4)

	// Two rows out of order by score; sorting by score desc should flip them.
	in <- engine.Product{SourceFile: "ref.fa", SequenceID: "s", ExperimentID: "x", Start: 0, End: 10, Length: 10, Type: "forward", Score: 1.5}
	in <- engine.Product{SourceFile: "ref.fa", SequenceID: "s", ExperimentID: "y", Start: 1, End: 11, Length: 10, Type: "forward", Score: 3.2}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("unexpected TSV: %q", out)
	}
	if !strings.Contains(lines[0], "score") {
		t.Fatalf("expected header to include 'score', got: %q", lines[0])
	}
	// First data line should be the higher score (3.2)
	if !strings.Contains(lines[1], "\t3.2") {
		t.Fatalf("expected first row to be score=3.2, got: %q", lines[1])
	}
}

func TestProductWriter_TSVThermoDetailsHeaderAndRow(t *testing.T) {
	var buf bytes.Buffer
	in, done := StartProductWriterWithThermoDetails(&buf, "text", false, true, false, true, true, true, 4)

	in <- engine.Product{
		SourceFile:   "ref.fa",
		SequenceID:   "s",
		ExperimentID: "x",
		Start:        0,
		End:          10,
		Length:       10,
		Type:         "forward",
		Score:        1.5,
		Thermo: &engine.ThermoDetails{
			Model:                   "nn-structure-v1",
			SaltModel:               "monovalent",
			NaM:                     0.05,
			MgM:                     0.003,
			DntpM:                   0.0008,
			EffectiveNaM:            0.05,
			FreeMgM:                 0.0022,
			AnnealTempC:             60,
			ScoreProfile:            "binding",
			BaseScoreC:              3.5,
			ScoreC:                  1.5,
			StructurePenaltyC:       2.0,
			LimitingSide:            "fwd",
			PanelCrossDimerPenaltyC: 1.25,
			PanelCrossDimerBurdenC:  2.75,
			PanelCrossDimerCount:    2,
			PanelCrossDimer: &engine.ThermoStructure{
				Kind:     "cross-dimer",
				QueryA:   "fwd",
				QueryB:   "external",
				PenaltyC: 1.25,
			},
		},
	}
	close(in)
	if err := <-done; err != nil {
		t.Fatalf("writer err: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected TSV lines (%d): %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "score\tthermo_model\tsalt_model\tna_m\tmg_m\tdntp_m\teffective_na_m\tfree_mg_m") || !strings.Contains(lines[0], "panel_cross_dimer_penalty_c") {
		t.Fatalf("expected thermo details header, got: %q", lines[0])
	}
	if !strings.Contains(lines[1], "\tnn-structure-v1\tmonovalent\t0.05\t0.003\t0.0008\t0.05\t0.0022\t60\t\t\t\t\tbinding\t3.5\t1.5") || !strings.Contains(lines[1], "\t2\tfwd") {
		t.Fatalf("expected thermo detail values, got: %q", lines[1])
	}
	if !strings.Contains(lines[1], "\t1.25\t2.75\t2\tfwd~external") {
		t.Fatalf("expected panel cross-dimer details, got: %q", lines[1])
	}
}
