package thermointegration

import (
	"bytes"
	"ipcr/internal/thermoapp"
	"os"
	"testing"
)

func writeFA(t *testing.T, name, data string) string {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return name
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

func TestThermo_ProbeThermoDefaultModelAutoPromotesToNNDuplex(t *testing.T) {
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
	if !bytes.Contains(out.Bytes(), []byte("nn-duplex-v1")) {
		t.Fatalf("expected implicit nn-duplex-v1 scoring when --probe-thermo is used with default model:\n%s", s)
	}
	if !bytes.Contains(out.Bytes(), []byte("probe_found")) || !bytes.Contains(out.Bytes(), []byte("\ttrue\tgate\tprobe\t")) {
		t.Fatalf("expected populated probe thermo detail columns:\n%s", s)
	}
}
