package output

import "testing"

func TestTSVHeader_Stable(t *testing.T) {
	const want = "source_file\tsequence_id\texperiment_id\tstart\tend\tlength\ttype\tfwd_mm\trev_mm\tfwd_mm_i\trev_mm_i"
	if TSVHeader != want {
		t.Fatalf("TSVHeader changed:\n got:  %q\n want: %q", TSVHeader, want)
	}
}
