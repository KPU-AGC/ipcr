# ipcr — fast in-silico PCR toolkit (primers, probes, nested & multiplex)

[![CI](https://img.shields.io/github/actions/workflow/status/KPU-AGC/ipcr/ci.yml?branch=main\&label=ci)](https://github.com/KPU-AGC/ipcr/actions/workflows/ci.yml)
[![Anaconda-Server Badge](https://anaconda.org/bioconda/ipcr/badges/downloads.svg)](https://anaconda.org/bioconda/ipcr)
[![install with bioconda](https://img.shields.io/badge/install%20with-bioconda-brightgreen.svg?style=flat)](http://bioconda.github.io/recipes/ipcr/README.html)
[![Go](https://img.shields.io/badge/go-%3E=%201.22-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

**ipcr** is a fast, streaming, IUPAC-aware in-silico PCR toolkit for large (including .gzipped) references.
It finds amplicons from primer pairs under a mismatch model with a **3′ terminal window**, supports **internal probes**, **nested PCR**, **multiplex panels**, circular templates, and emits **TSV**, **FASTA**, **JSON**, or **JSONL**.

---

* **Fast & parallel**: multi-threaded seeded scanner with per-hit verification.
* **Streaming**: chunked FASTA with safe overlap.
* **IUPAC-aware**: primer ambiguity codes; genome `N` is a **hard mismatch** to avoid spurious hits.
* **Pretty mode**: readable ASCII alignment blocks.
* **Deterministic**: `--sort` gives stable order; JSON/JSONL use versioned, stable wire schemas.
* **Good UX**: cancelable I/O (Ctrl-C → exit 130), consistent warnings (gated by `--quiet`), clear validation errors.

---

## Binaries (CLIs)

| Binary           | Description	              | Common use                   |
| ---------------- | ------------------------------------------------ | ---------------------------- |
| `ipcr`           | Standard in-silico PCR	   | general PCR                  |
| `ipcr-probe`     | ipcr + **internal probe** annotation & filtering | qPCR/TaqMan-style assays     |
| `ipcr-nested`    | **Nested PCR**: outer amplicon + inner scan      | Two-round/nested assays      |
| `ipcr-multiplex` | Panels from TSV or **pooled inline** primers     | Screens / large panels       |
| `ipcr-thermo`    | Thermodynamically-informed scoring & ranking     | Ranking / assay robustness   |

---

## Quick start

### Standard PCR

```shell
# PCR with standard 27F and 1492R primers.
ipcr \
  --forward AGAGTTTGATCMTGGCTCAG --reverse TACGGYTACCTTGTTAYGACTT \
  --circular \
  --pretty \
  Escherichia-coli.fna.gz
```
```
source_file     sequence_id     experiment_id   start   end     length  type    fwd_mm  rev_mm  fwd_mm_i        rev_mm_i
Escherichia-coli.fna.gz   NC_000913.3     manual  223777  225283  1506    forward 0       0
# 5'-AGAGTTTGATCMTGGCTCAG-3'
#    |||||||||||¦||||||||-->
# 5'-AGAGTTTGATCATGGCTCAG.................................................................................................-3' # (+)
# 3'-...............................................................................................ATGCCAATGGAACAATGCTGAA-5' # (-)
#				<--|||||¦||||||||||¦|||||
#				3'-TTCAGYATTGTTCCATYGGCAT-5'
#
...
```

### Probe (with JSON):
```bash
# With primers + probe described by Parker et al. (2017); DOI: 10.1371/journal.pone.0173422
ipcr-probe \
  -f TCTAATTTTTTCATCATCGCTAATGC -r TCAGGCCTTTGCTACAATGAAC -P AACTGCATCATATCACATACT \
  --circular \
  --output json \
  Mycoplasmopsis-bovis.fna.gz
```
```json
[
  {
    "experiment_id": "manual",
    "sequence_id": "NC_014760.1",
    "start": 353221,
    "end": 353333,
    "length": 112,
    "type": "revcomp",
    "seq": "TCAGGCCTTTGCTACAATGAACTTATTTTTAACTAACGCAAATAAAACATATAGTATGTGATATGATGCAGTTTTAAATAATAAGAGCATTAGCGATGATGAAAAAATTAGA",
    "source_file": "Mycoplasmopsis-bovis.fna.gz",
    "probe_name": "probe",
    "probe_seq": "AACTGCATCATATCACATACT",
    "probe_found": true,
    "probe_strand": "-",
    "probe_pos": 52,
    "probe_site": "AGTATGTGATATGATGCAGTT"
  }
]
```

### Nested PCR (outer TSV, inner TSV): 

```bash
# With external and internal primers described by Figuero et al. (2011); DOI: 10.1902/jop.2011.100719
ipcr-nested \
  --outer-primers 27F-1492R.tsv \
  --inner-primers Fn_nested-primers.tsv \
  --output text \
  --mismatches 1 \
  --sort \
  Fusobacterium-nucleatum.fna.gz
```
```
source_file	sequence_id	outer_experiment_id	outer_start	outer_end	outer_length	outer_type	inner_experiment_id	inner_found	inner_start	inner_end	inner_length	inner_type	inner_fwd_mm	inner_rev_mm
Fusobacterium-nucleatum.fna.gz	NZ_CP028101.1	outer	534786	536270	1484	forward	Fn-F517-R1214	true	541	1237	696	forward	0	0
Fusobacterium-nucleatum.fna.gz	NZ_CP028101.1	outer	613679	615163	1484	forward	Fn-F517-R1214	true	541	1237	696	forward	0	0
Fusobacterium-nucleatum.fna.gz	NZ_CP028101.1	outer	1079673	1081157	1484	forward	Fn-F517-R1214	true	541	1237	696	forward	0	0
Fusobacterium-nucleatum.fna.gz	NZ_CP028101.1	outer	335297	336781	1484	revcomp	Fn-F517-R1214	true	247	943	696	revcomp	0	0
Fusobacterium-nucleatum.fna.gz	NZ_CP028101.1	outer	2054505	2055989	1484	revcomp	Fn-F517-R1214	true	247	943	696	revcomp	0	0
```

### Multiplex panel (TSV of many pairs):

```bash
# With multiplex primer pool described by Park and Ricke (2015); DOI: 10.1111/jam.12678
ipcr-multiplex \
  --primers multiplex-pcr-assay.tsv \
  --output jsonl \
  --products \
  --sort \
  Salmonella-Typhimurium.fna.gz
```
```json
{"experiment_id":"ST","sequence_id":"NC_003197.2","start":4750875,"end":4751186,"length":311,"type":"revcomp","seq":"ATGACAAACTCTTGATTCTGAAGATCGACTTTTTTTGCTATGTAATCCGCGATCTTTTTCTGATTCAATAAGCCAACGAGTTGTTTTTTCAGCGCTTCGGTACCGACTTTCACTTCCTGCTGACAGACGCGGTCAAATAACCCACGTTCAGTGAGCATGTCGACGATGATCTGAAAGATGTTGAGGTGCGCGAACTTGTGGTCCTTTTCCAGATTACGCAACAGATACTTCAGGTGTTCACGCACCTGCAGCTCATTCTGAGCAGGATAATCAAAAATCCAGAACCCAATCTCATTACCGGAGCCGTTGTT","source_file":"S-typhimurium.fna.gz"}
```

### Thermodynamically-informed :

```bash
# With multiplex primer pool described by Xiong (2017); DOI: 10.3389/fmicb.2017.00420
ipcr-thermo \
  --oligo ATGTCTATAAGCACCACAATG         --oligo TCATTTCAATAATGATTCAAGC \
  --oligo CATTCTGACCTTTAAGCCGGTCAATGAG  --oligo CCAAAAAGCGAGACCTCAAACTTACTCAG \
  --oligo GCGGACGTCATTGTCACTAACCCGACG   --oligo TCTAAAGTGGGAACCCGATGTTCAGCG \
  --mismatches 3 --circular \
  Salmonella-Enteritidis.fna.gz
```
```text
source_file	sequence_id	experiment_id	start	end	length	type	fwd_mm	rev_mm	fwd_mm_i	rev_mm_i	score
Salmonella-Enteritidis	NZ_CP025559.1	O3+O4	2446500	2446839	339	revcomp	0	0	-110.73864769266693
Salmonella-Enteritidis	NZ_CP025559.1	O5+O6	2734882	2735037	155	revcomp	0	0	-120.79822631649216
Salmonella-Enteritidis	NZ_CP025559.1	O1+O2	1853303	1854185	882	revcomp	0	0	-137.31230787351492
```

---

## Inputs

* **Inline primers**: `-f/--forward`, `-r/--reverse` (5′→3′; IUPAC allowed).
* **TSV primers** (panel file):

  ```
  id  FORWARD_PRIMER  REVERSE_PRIMER  [min_len]  [max_len]
  ```

  Optional per-pair `min_len`/`max_len` override global bounds.
* **FASTA**: Positional paths/globs. Use `-` for **stdin**. gz is auto-detected. (Also accepts `--sequences FILE[.gz]` (repeatable), soon to be deprecated)

---

## Performance & streaming

* **Threads**: `--threads N` (0 = all CPUs).
* **Chunking**: `--chunk-size N` splits records into windows; **overlap** is chosen safely from `max(--max-length, primer_len-1)` so boundary hits survive.

  * If `--circular`, chunking is disabled.
  * If `--max-length` is missing or `--chunk-size <= --max-length`, chunking auto-disables with a warning.
* **Seeding**: exact 3′ suffix seeds (default length 12 or primer-length if shorter) for forward primers; rc seeds use 5′ prefixes. Ambiguous primers fall back to full verification.
* **Cancelable I/O**: FASTA scanners honor context; Ctrl-C exits with **130**.

---

## Flags you’ll actually use

* `--mismatches N` — max mismatches per primer (default 0)
* `--terminal-window N` — no mismatches allowed in the last *N* bases (3 nt by default in `realistic` mode)
* `--min-length / --max-length` — product length bounds
* `--circular` — permit wrap-around amplicons
* `--output text|json|jsonl|fasta` — choose format; `--sort` for stable order; `--products` to emit sequences in text/json
* `--pretty` — ASCII alignment blocks (text)
* `--self=true|false` — include **single-oligo amplification** (A×rc(A), B×rc(B)) (default **true**)

---

## License & citation

MIT. See [LICENSE](./LICENSE).

If you use **ipcr** in your work, please cite this repository (a manuscript is in progress).

```
```
