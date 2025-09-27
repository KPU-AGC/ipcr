// Package writers turns domain products into serialized outputs.
//
// Design:
//   • Writers own all presentation knowledge (pretty blocks, JSON/JSONL/FASTA).
//   • Engine stays domain-only; Pipeline stays orchestration-only.
//   • JSON/JSONL go through pkg/api (v1) for a stable wire format.
package writers
