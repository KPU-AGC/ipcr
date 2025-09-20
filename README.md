# ipcr

[![install with bioconda](https://img.shields.io/badge/install%20with-bioconda-brightgreen.svg?style=flat)](http://bioconda.github.io/recipes/ipcr/README.html)
[![Go](https://img.shields.io/badge/go-%3E=1.19-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

**ipcr** is a fast, parallel, in‑silico PCR tool written in Go.  
It scans large (including gzipped) FASTA references for amplicons from primer pairs and streams results as text, FASTA, or JSON.

---

## Why ipcr?

- **Fast & parallel** – uses all CPU cores by default.
- **Large‑scale friendly** – streams `.fa` / `.fa.gz` and supports chunked windows with boundary‑safe overlap.
- **Pragmatic matching** – k‑mismatch support with a 3′ terminal window policy.
- **Deterministic when needed** – `--sort` provides stable output ordering.
- **Single static binary** – no external dependencies.

---

## Install

```bash
# Build the CLI
go build -o ipcr cmd/ipcr/main.go
# Run tests
go test ./...
````

> Requires Go ≥ 1.19.

---

## Quick start

Inline primers:

```bash
./ipcr --forward ACGTGTA --reverse TTGCCGA --sequences genome.fa
```

Primers from TSV and gzipped FASTA:

```bash
./ipcr --primers primers.tsv --sequences sample.fa.gz
```

Print help:

```bash
./ipcr --help
```

---

## Input formats

### Primer TSV

Tab‑separated file with required columns:

```
id   FORWARD_PRIMER   REVERSE_PRIMER   [min_len]   [max_len]
```

* `id`: experiment/primer‑pair identifier.
* `FORWARD_PRIMER`, `REVERSE_PRIMER`: 5′→3′ sequences (IUPAC allowed).
* `min_len`, `max_len` (optional): per‑pair product length bounds.
  If omitted, global `--min-length/--max-length` apply.

### Sequences (FASTA)

* One or more files via repeated `--sequences`.
* Use `--sequences -` to read from `stdin`.
* `.gz` is detected automatically.

---

## Output formats

### Text (default)

Tabular, one hit per line:

```
sequence_id  experiment_id  start  end  length  type  fwd_mm  rev_mm  fwd_mismatch_idx  rev_mismatch_idx
```

* `type`: `forward` (A × rc(B)) or `revcomp` (B × rc(A)).
* Coordinates are **0‑based half‑open** over the input chunk; when chunking, IDs carry `:<start>-<end>` and de‑duping preserves real genomic intervals.

### FASTA

Amplicon sequences with coordinates in headers.

### JSON

Structured objects for downstream pipelines.

Use `--products` to attach the amplicon sequence to text/JSON outputs.

---

## Key options (most used)

* `--forward`, `--reverse` – inline primers (use **instead of** `--primers`).
* `--primers FILE.tsv` – TSV primer file (see above).
* `--sequences FILE[.gz]` – one or more FASTA files (repeatable or `-`).
* `--mismatches N` – per‑primer mismatch cap (default: `0`).
* `--terminal-window N` – number of 3′ bases where mismatches are disallowed. `-1` = auto (`3` in realistic mode, `0` in debug).
* `--min-length N`, `--max-length N` – product length bounds (global defaults; per‑pair overrides in TSV).
* `--output text|json|fasta` – choose output format (default: `text`).
* `--products` – include product sequence.
* `--pretty` – add ASCII site strings to text output (slower; gated to avoid extra allocations otherwise).
* `--threads N` – worker threads (`0` = all CPUs).
* `--chunk-size N` – process the reference in sliding windows; overlap is set to cover the largest primer/product so boundary hits aren’t lost.
* `--hit-cap N` – cap matches retained per primer/orientation per window (`0` = unlimited).
* `--mode realistic|debug` – sets sensible defaults (e.g., terminal window).
* `--sort` – make stream ordering deterministic (costs some memory).
* `--seed-length N` – seed length for the multi‑pattern scan (`12` default; `0` = full‑length seeds).
  Larger seeds reduce candidates; smaller seeds are more sensitive with more verification.

Run `./ipcr --help` to see all flags and defaults.

---

## Methods (overview)

* **Single‑pass seeded scan.** For each pair, exact seeds anchored at the primer 3′ end (and the 5′ end of the reverse‑complement) are combined into a multi‑pattern automaton. The genome is scanned **once** per chunk/strand; seed hits are then **verified** against full primers allowing up to `--mismatches` and enforcing the 3′ terminal‑window policy.
* **No reverse‑complementing the genome.** Reverse primers are scanned via **reverse‑complemented primers on the forward strand**, eliminating an O(n) pass per pair.
* **Exact‑match fast path.** With `--mismatches=0` and unambiguous A/C/G/T primers, exact matches use a jump‑scanning path.
* **Boundary‑safe chunking.** Sliding windows reuse buffers (allocation‑free) and overlap sufficiently to keep amplicons that span window edges; duplicates are removed across chunks.

---

## Examples

Text (default):

```bash
./ipcr --forward AGCTG --reverse TTGCA --sequences test.fa
sequence_id  experiment_id  start  end  length  type     fwd_mm  rev_mm  fwd_mismatch_idx  rev_mismatch_idx
chr1         manual         12345  12378 33     forward  0       1                        2
```

FASTA:

```bash
./ipcr --forward AGCTG --reverse TTGCA --sequences test.fa --output fasta --products
>manual  chr1:12345-12378  len=33  type=forward
AGCTG...TTGCA
```

---

## Reproducibility & performance tips

* Prefer unambiguous primers with `--mismatches=0` for the fastest path.
* Tune `--seed-length` for your datasets (`8–16` is typical).
  Full‑length seeds (`0`) minimize verification work but can reduce sensitivity with degenerate primers.
* Use `--sort` if you need deterministic order; leave off for maximum throughput.
* Set `--chunk-size` for very large references to limit memory while keeping overlap correctness.

---

## License

MIT. See [LICENSE](./LICENSE).

---

## Citation

I wish.

```
```
