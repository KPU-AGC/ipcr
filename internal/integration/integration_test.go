// internal/integration/integration_test.go
package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
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
			"--sort", // deterministic order
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

/*** Canonicalization helpers (robust to output format changes) ***/

// parse base ID and chunk offset from IDs like "s:8-24".
// Returns base="s", off=8 if present; else base=id, off=0.
func baseAndOffset(id string) (string, int) {
	colon := strings.LastIndex(id, ":")
	if colon == -1 || colon == len(id)-1 {
		return id, 0
	}
	suffix := id[colon+1:]
	dash := strings.IndexByte(suffix, '-')
	if dash == -1 {
		return id, 0
	}
	startStr := suffix[:dash]
	if start, err := strconv.Atoi(startStr); err == nil {
		return id[:colon], start
	}
	return id, 0
}

// canonicalizeJSON parses the JSON array of products, normalizes chunked coords
// to global coords, and returns a sorted, de-duplicated set of signature rows.
// Using JSON avoids any brittleness from text/TSV column changes.
func canonicalizeJSON(js string) ([]string, error) {
	var prods []engine.Product
	if err := json.Unmarshal([]byte(js), &prods); err != nil {
		return nil, err
	}
	uniq := make(map[string]struct{}, len(prods))
	for _, p := range prods {
		base, off := baseAndOffset(p.SequenceID)
		gs, ge := p.Start+off, p.End+off
		// Signature captures the amplicon identity invariant under chunking.
		sig := fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%s",
			base, p.ExperimentID, gs, ge, p.Length, p.Type)
		uniq[sig] = struct{}{}
	}
	out := make([]string, 0, len(uniq))
	for k := range uniq {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

/*** The chunking equivalence test (now using JSON output) ***/

// Chunking large enough (and with safe overlap) should match no-chunking results
// after canonicalization. We run in JSON to avoid dependence on text/TSV columns.
func TestChunkingKeepsBoundaryHits(t *testing.T) {
	// Sequence long enough that many products exist
	fa := write(t, "chunk.fa", ">s\nACGTACGTACGTACGTACGTACGTACGT\n")
	defer os.Remove(fa)

	runJSON := func(chunk int) string {
		var out, errB bytes.Buffer
		args := []string{
			"--forward", "ACGTAC",
			"--reverse", "ACGTAC",
			"--sequences", fa,
			"--output", "json",
			"--sort",
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

	noChunkJSON := runJSON(0)
	chunkedJSON := runJSON(16) // safe: > max-length; code uses overlap ≥ max-length

	nc, err := canonicalizeJSON(noChunkJSON)
	if err != nil {
		t.Fatalf("canonicalize no-chunk (json): %v", err)
	}
	ck, err := canonicalizeJSON(chunkedJSON)
	if err != nil {
		t.Fatalf("canonicalize chunked (json): %v", err)
	}

	if strings.Join(nc, "\n") != strings.Join(ck, "\n") {
		// For debugging, also dump the *text* versions alongside the normalized sets.
		var rawNo, rawCh bytes.Buffer
		_ = app.Run([]string{
			"--forward", "ACGTAC", "--reverse", "ACGTAC",
			"--sequences", fa, "--output", "text", "--sort", "--no-header",
			"--max-length", "8",
		}, &rawNo, &bytes.Buffer{})
		_ = app.Run([]string{
			"--forward", "ACGTAC", "--reverse", "ACGTAC",
			"--sequences", fa, "--output", "text", "--sort", "--no-header",
			"--max-length", "8", "--chunk-size", "16",
		}, &rawCh, &bytes.Buffer{})

		t.Fatalf("chunked output differs from no-chunking\nno-chunk(norm):\n%s\nchunked(norm):\n%s\n\nno-chunk(text):\n%s\nchunked(text):\n%s",
			strings.Join(nc, "\n"), strings.Join(ck, "\n"),
			trimHead(rawNo.String()), trimHead(rawCh.String()))
	}
}

func trimHead(out string) string {
	var b strings.Builder
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" ||
			strings.HasPrefix(line, "sequence_id") ||
			strings.HasPrefix(line, "source_file") ||
			strings.HasPrefix(line, "FWD ") ||
			strings.HasPrefix(line, "REV ") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
