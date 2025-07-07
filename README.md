# ipcr

[![Go](https://img.shields.io/badge/go-%3E=1.19-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

**ipcr** is a fast, parallel in-silico PCR tool written in Go.  
It can scan large (including gzipped) FASTA files for PCR products using primer pairs, with a modern and scriptable CLI.

---

## Features

- **Multi-threaded:** Utilizes all CPU cores by default for fast processing.
- **Flexible input:** Accepts primers as a TSV file or via CLI.
- **Compressed or plain input:** Supports both plain and `.gz` FASTA files.
- **Versatile output:** Write matches as text, FASTA, or JSON.
- **Tested and reproducible:** Fully unit/integration tested; robust argument checking.
- **Easy to build:** Single Go binary, no external dependencies.

---

## Usage

Print help:

```sh
./ipcr --help
```

Basic run with inline primers:

```sh
./ipcr --forward ACGTGTA --reverse TTGCCGA --sequences genome.fa
```

With a TSV primer file and gzipped FASTA:

```sh
./ipcr --primers primers.tsv --sequences sample.fa.gz
```

**Primer TSV format:**

```
id   FORWARD_PRIMER   REVERSE_PRIMER   [min_len]   [max_len]
```

(Min/max are optional.)

**Output formats:**

* `--output text` (default): tabular, one hit per line
* `--output fasta`: matched sequences in FASTA format (with product coordinates)
* `--output json`: full structured result

**Other useful flags:**

* `--threads 4` to set number of CPUs (default: all CPUs)
* `--chunk-size 10000` for large input splitting
* `--mismatches 2` to allow mismatches per primer
* `--products` to include actual product sequences in output
* `-v` or `--version` for version info

---

## Example

```sh
./ipcr --forward AGCTG --reverse TTGCA --sequences test.fa --output fasta --products
```

Output (FASTA):

```
>manual_1 start=5 end=34 len=29
AGCTG...TTGCA
```

---

## Advanced

* Supports multiple `--sequences` arguments.
* Use `--min-length`, `--max-length`, and `--hit-cap` for fine-grained filtering.
* Input can be streamed from stdin with `--sequences -`.

---

## Contact

Questions, issues, or suggestions?
Contact: [erick.samera@kpu.ca](mailto:erick.samera@kpu.ca)
```
