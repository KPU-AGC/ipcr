package output

import "testing"

func TestFormats_Stable(t *testing.T) {
	if FormatText != "text" || FormatJSON != "json" || FormatJSONL != "jsonl" || FormatFASTA != "fasta" {
		t.Fatalf("output format constants changed")
	}
}
