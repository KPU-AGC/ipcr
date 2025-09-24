# `ipcr`

[![install with bioconda](https://img.shields.io/badge/install%20with-bioconda-brightgreen.svg?style=flat)](http://bioconda.github.io/recipes/ipcr/README.html)
[![Go](https://img.shields.io/badge/go-%3E=1.19-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

**`ipcr`** is a fast, parallel, in‑silico PCR tool written in Go.  
It scans large (including gzipped) FASTA references for amplicons from primer pairs and streams results as text/TSV, FASTA, or JSON.

---

## Highlights

- **Fast & parallel by default.**
- **Seeded multi‑pattern scan.**
- **IUPAC support on primers.**
- **Deterministic when needed.**
- **Pretty text mode.**
- **Robust outputs.**

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

## Usage & inputs

`ipcr` accepts primer pairs **either** inline **or** from a TSV file, plus one or more FASTA references:

* **Inline primers:** `-f/--forward`, `-r/--reverse` (5′→3′).
* **TSV primers:** `--primers FILE.tsv` (see format below).
* **FASTA inputs:** repeat `--sequences FILE[.gz]` **or** supply positional FASTA paths/globs (`ref*.fa gz/*.fa.gz`). Using `-` reads from **stdin**. Globs are expanded only for positional arguments; `--sequences` is literal.

At least one primer source **and** at least one FASTA input are required. The program prints a structured usage banner on `-h/--help`.

### Primer TSV format

Whitespace‑separated columns:

```
id  FORWARD_PRIMER  REVERSE_PRIMER  [min_len]  [max_len]
```

* `id`: experiment/primer‑pair identifier.
* `FORWARD_PRIMER`, `REVERSE_PRIMER`: 5′→3′ sequences; IUPAC ambiguity codes are allowed.
* Optional `min_len`, `max_len`: per‑pair length bounds; otherwise global `--min-length/--max-length` apply.

---

## Matching model (what’s simulated)

* **Mismatches:** Allow up to `--mismatches` per primer.
  A **3′ terminal window** (`--terminal-window`) forbids mismatches at the 3′ end (auto: `3` in `--mode realistic`)
* **Primer ambiguity (IUPAC):** Primer codes like R/Y/W... match the allowed bases; genome `N` and non‑ACGT characters are treated as mismatches to prevent flood hits in unknown regions.
* **Seeded scan:**

  * 3′‑anchored suffix seeds for forward primers (**A**, **B**) and 5′‑anchored prefix seeds for reverse‑complements (**a=rc(A)**, **b=rc(B)**) are built (exact A/C/G/T only) and packed into an AC automaton.
  * Each seed hit is **verified in place** against the full primer with the mismatch + 3′ policy; orientations lacking usable seeds fall back to a direct verifier.
* **Product typing:** Hits are labeled `forward` (A × rc(B)) or `revcomp` (B × rc(A)). Coordinates are 0‑based, half‑open over the scanned window; when chunking is enabled, IDs carry `:<start>-<end>` to indicate window origin. De‑duplication yields the true intervals across chunk boundaries.
* **Circular templates:** When `--circular` is set, amplicons that wrap from the end to the beginning are allowed (and recognizable as Start > End in the per‑chunk coordinates). Chunking is disabled in this mode.

---

## Performance

* **Threads:** `--threads N` (default `0` = all CPUs). Work is sharded across records/chunks.
* **Chunking:** `--chunk-size N` streams a fixed‑size sliding window per record; overlap is chosen to cover `max(primer_len-1, --max-length)` so boundary hits are kept. If `--circular` is set, chunking is disabled. If `--max-length` is missing or `--chunk-size <= --max-length`, chunking is auto‑disabled with a warning (suppress with `--quiet`).
* **Seeds:** `--seed-length L` (default `12`; `0`=auto=min(12, primer length)). Only unambiguous seed segments are used; others fall back to verifier.
* **Hit cap:** `--hit-cap N` limits matches stored per primer/orientation/window (`0`=unlimited) to control memory.

---

## Outputs

### Text / TSV (default)

Canonical header (always the same order):

```
source_file  sequence_id  experiment_id  start  end  length  type  fwd_mm  rev_mm  fwd_mm_i  rev_mm_i
```

* `source_file` is the input FASTA file the hit came from (handy with many inputs).
* `type` ∈ `{forward, revcomp}`.
* Use `--no-header` to suppress the header row.
* `--sort` sorts in‑memory for determinism (SequenceID, Start, End, Type, ExperimentID).

**Pretty mode:** `--pretty` appends a multi‑line diagram per hit, with `# `‑prefixed lines that you can grep away. Exact matches render as `|`; ambiguity‑mediated matches render as `¦`; mismatches are spaces. The (+) strand is 5′→3′ left→right; the (–) strand is 3′→5′ under it.

### FASTA

FASTA records for each product when `--output fasta` (and `--products` to include sequences for text/JSON too). Headers carry metadata:

```
>EXPERIMENT_i start=... end=... len=... source_file=...
SEQUENCE
```

Streamed or batch‑written depending on `--sort`.

### JSON

Pretty‑printed array of objects with fields:

```json
{
  "experiment_id": "...",
  "sequence_id": "...",
  "start": 0,
  "end": 0,
  "length": 0,
  "type": "forward|revcomp",
  "fwd_mm": 0,
  "rev_mm": 0,
  "fwd_mismatch_idx": [],
  "rev_mismatch_idx": [],
  "seq": "ACGT...",          // only when --products
  "source_file": "ref.fa"
}
```

---

## Examples

Default text/TSV:

```bash
./ipcr \
  --forward AGRGTTYGATYMTGGCTCAG \
  --reverse RGYTACCTTGTTACGACTT \
  --mismatches 1 \
  --pretty \
  ref.fna.gz
```
```
source_file      sequence_id     experiment_id   start   end     length  type    fwd_mm  rev_mm  fwd_mm_i        rev_mm_i
ref.fna.gz       REF:0-1003404   manual          316104  317579  1475    forward 0       1                       2
# 5'-AGRGTTYGATYMTGGCTCAG-3'
#    ||¦|||¦|||¦¦||||||||-->
# 5'-AGAGTTTGATCCTGGCTCAG..............................-3' # (+)
# 3'-...............................CCTATGGAACAATGCTGAA-5' # (-)
#                                <--|||||||||||||||| |¦
#                                3'-TTCAGCATTGTTCCATYGR-5'
#
```

FASTA with sequences:

```bash
./ipcr \
  --forward AGRGTTYGATYMTGGCTCAG \
  --reverse RGYTACCTTGTTACGACTT \
  --mismatches 1 \
  --output fasta \
  ref.fna.gz
```
```bash
>manual_1 start=316104 end=317579 len=1475 source_file=ref.fna.gz
AGAGT...TATCC
```

JSON:

```bash
./ipcr \
  --forward AGRGTTYGATYMTGGCTCAG \
  --reverse RGYTACCTTGTTACGACTT \
  --mismatches 1 \
  --output json \
  ref.fna.gz | jq
```
```json
[
  {
    "experiment_id": "manual",
    "sequence_id": "REF:0-1003404",
    "start": 316104,
    "end": 317579,
    "length": 1475,
    "type": "forward",
    "rev_mm": 1,
    "rev_mismatch_idx": [
      2
    ],
    "source_file": "ref.fna.gz"
  },
]
```

---

## License

MIT. See [LICENSE](./LICENSE).

## Citation

I wish.

```
```