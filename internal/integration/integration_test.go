// internal/integration/integration_test.go
package integration

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"ipcr/internal/app"
	"ipcr/internal/engine"
	"ipcr/internal/output"
)

func write(t *testing.T, fn, data string) string {
	t.Helper()
	if err := os.WriteFile(fn, []byte(data), 0644); err != nil {
		t.Fatalf("write %s: %v", fn, err)
	}
	return fn
}

func TestEndToEnd(t *testing.T) {
	fa := write(t, "itest.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	var out, errBuf bytes.Buffer
	code := app.Run([]string{
		"--forward", "ACG",
		"--reverse", "ACG",
		"--sequences", fa,
	}, &out, &errBuf)

	if code != 0 {
		t.Fatalf("run exit %d, err=%s", code, errBuf.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected text output")
	}
}

func TestParallelMatchesEqualSerial(t *testing.T) {
	fa := write(t, "par.fa", ">s\nACGTACGTACGT\n")
	defer os.Remove(fa)

	run := func(threads int) string {
		var out, errB bytes.Buffer
		code := app.Run([]string{
			"--forward", "ACG",
			"--reverse", "ACG",
			"--sequences", fa,
			"--threads", fmt.Sprint(threads),
			"--output", "json",
			"--sort", // ensure deterministic order
		}, &out, &errB)
		if code != 0 {
			t.Fatalf("exit %d err %s", code, errB.String())
		}
		return out.String()
	}

	serial := run(1)
	parallel := run(4)

	if serial != parallel {
		t.Fatalf("parallel output differs from serial\nserial: %s\nparallel:%s", serial, parallel)
	}
}

// text vs TSV should be byte-identical when header settings match
func TestTextVsTSVParity(t *testing.T) {
	list := []engine.Product{
		{SequenceID: "s:0-12", ExperimentID: "x", Start: 0, End: 12, Length: 12, Type: "forward", FwdMM: 0, RevMM: 1, FwdMismatchIdx: nil, RevMismatchIdx: []int{2}},
		{SequenceID: "s:0-12", ExperimentID: "x", Start: 4, End: 8, Length: 4, Type: "revcomp", FwdMM: 1, RevMM: 0, FwdMismatchIdx: []int{1}, RevMismatchIdx: nil},
	}

	var textB, tsvB bytes.Buffer

	// StreamText with a channel (no header, no pretty)
	ch := make(chan engine.Product, len(list))
	for _, p := range list {
		ch <- p
	}
	close(ch)
	if err := output.StreamText(&textB, ch, false, false); err != nil {
		t.Fatalf("stream text: %v", err)
	}

	// WriteTSV (no header)
	if err := output.WriteTSV(&tsvB, list, false); err != nil {
		t.Fatalf("write tsv: %v", err)
	}

	if textB.String() != tsvB.String() {
		t.Fatalf("parity mismatch\ntext:\n%s\ntsv:\n%s", textB.String(), tsvB.String())
	}
}

// canonicalize: convert chunk-local coords to global by adding the chunk start offset,
// and drop chunk ranges from SequenceID (keep base id) so chunked vs no-chunk are comparable.
func canonicalize(out string) []string {
	sc := bufio.NewScanner(strings.NewReader(out))
	var rows []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "sequence_id") || strings.HasPrefix(line, "FWD ") || strings.HasPrefix(line, "REV ") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 8 {
			continue
		}
		seqID := cols[0]
		exp := cols[1]
		start, _ := strconv.Atoi(cols[2])
		end, _ := strconv.Atoi(cols[3])
		length := cols[4]
		typ := cols[5]
		fwdmm := cols[6]
		revmm := cols[7]

		base := seqID
		offset := 0
		if i := strings.Index(seqID, ":"); i > -1 {
			base = seqID[:i]
			if j := strings.Index(seqID[i+1:], "-"); j > -1 {
				a := seqID[i+1 : i+1+j]
				if v, err := strconv.Atoi(a); err == nil {
					offset = v
				}
			}
		}
		gStart := start + offset
		gEnd := end + offset
		rows = append(rows, fmt.Sprintf("%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s", base, exp, gStart, gEnd, length, typ, fwdmm, revmm))
	}
	sort.Strings(rows)
	return rows
}

// Chunking large enough (and with safe overlap) should match no-chunking results (after canonicalization)
func TestChunkingKeepsBoundaryHits(t *testing.T) {
	// Sequence long enough that many products exist
	fa := write(t, "chunk.fa", ">s\nACGTACGTACGTACGTACGTACGTACGT\n")
	defer os.Remove(fa)

	run := func(chunk int) string {
		var out, errB bytes.Buffer
		args := []string{
			"--forward", "ACGTAC",
			"--reverse", "ACGTAC",
			"--sequences", fa,
			"--output", "text",
			"--sort",
			"--no-header",
			"--max-length", "8", // constrain to 8 so chunking can be exact
		}
		if chunk > 0 {
			args = append(args, "--chunk-size", fmt.Sprint(chunk))
		}
		code := app.Run(args, &out, &errB)
		if code != 0 {
			t.Fatalf("exit %d err %s", code, errB.String())
		}
		return out.String()
	}

	noChunk := run(0)
	// Safe: chunk-size > max-length and code will set overlap >= max-length
	chunked := run(16)

	nc := strings.Join(canonicalize(noChunk), "\n")
	ck := strings.Join(canonicalize(chunked), "\n")

	if nc != ck {
		t.Fatalf("chunked output differs from no-chunking\nno-chunk:\n%s\nchunked:\n%s", nc, ck)
	}
}
