// internal/output/fasta_test.go
package output

import (
	"bytes"
	"ipcr-core/engine"
	"strings"
	"testing"
)

func TestWriteFASTA(t *testing.T) {
	buf := &bytes.Buffer{}
	list := []engine.Product{{
		ExperimentID: "p1", Seq: "ACGT",
		Start: 0, End: 4, Length: 4,
	}}
	if err := WriteFASTA(buf, list); err != nil {
		t.Fatalf("fasta: %v", err)
	}
	if !strings.Contains(buf.String(), ">p1_1") || !strings.Contains(buf.String(), "ACGT") {
		t.Fatalf("unexpected FASTA output: %s", buf.String())
	}
}

// ===

func TestWriteFASTA_DefaultHeaderUnchanged(t *testing.T) {
	buf := &bytes.Buffer{}
	list := []engine.Product{{
		ExperimentID: "manual", SequenceID: "seqA", Seq: "ACGT",
		Start: 307, End: 772, Length: 465, Type: "forward", SourceFile: "refs.fa",
	}}
	if err := WriteFASTA(buf, list); err != nil {
		t.Fatalf("fasta: %v", err)
	}
	want := ">manual_1 start=307 end=772 len=465 source_file=refs.fa\nACGT\n"
	if buf.String() != want {
		t.Fatalf("default header changed:\n got %q\nwant %q", buf.String(), want)
	}
}

func sourceIDFixture() []engine.Product {
	mk := func(seqID string, start int, seq string) engine.Product {
		return engine.Product{
			ExperimentID: "manual", SequenceID: seqID, Seq: seq,
			Start: start, End: start + len(seq), Length: len(seq), Type: "forward", SourceFile: "refs.fa",
		}
	}
	return []engine.Product{
		mk("seqA", 10, "AAAA"),
		mk("seqA", 50, ""), // no sequence: skipped and must not consume a number
		mk("seqB", 5, "CCCC"),
		mk("seqA", 90, "GGGG"),
	}
}

const sourceIDWant = "" +
	">seqA_amp1 experiment=manual sequence_id=seqA start=10 end=14 len=4 type=forward source_file=refs.fa\nAAAA\n" +
	">seqB_amp1 experiment=manual sequence_id=seqB start=5 end=9 len=4 type=forward source_file=refs.fa\nCCCC\n" +
	">seqA_amp2 experiment=manual sequence_id=seqA start=90 end=94 len=4 type=forward source_file=refs.fa\nGGGG\n"

func TestWriteFASTAWithIDs_Source(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := WriteFASTAWithIDs(buf, sourceIDFixture(), FastaIDsSource); err != nil {
		t.Fatalf("fasta: %v", err)
	}
	if buf.String() != sourceIDWant {
		t.Fatalf("source IDs:\n got %q\nwant %q", buf.String(), sourceIDWant)
	}
}

func TestStreamFASTAWithIDs_Source(t *testing.T) {
	in := make(chan engine.Product)
	go func() {
		for _, p := range sourceIDFixture() {
			in <- p
		}
		close(in)
	}()
	buf := &bytes.Buffer{}
	if err := StreamFASTAWithIDs(buf, in, FastaIDsSource); err != nil {
		t.Fatalf("fasta: %v", err)
	}
	if buf.String() != sourceIDWant {
		t.Fatalf("source IDs:\n got %q\nwant %q", buf.String(), sourceIDWant)
	}
}

func TestValidFastaIDs(t *testing.T) {
	for _, ok := range []string{FastaIDsExperiment, FastaIDsSource} {
		if !ValidFastaIDs(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "Source", "seq"} {
		if ValidFastaIDs(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
