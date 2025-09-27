// Package pipeline streams FASTA chunks through an Engine-like Simulator,
// deduplicates cross-boundary hits, and calls a visit callback.
//
// The only contract to implement is Simulator (SimulateBatch).
// This keeps the pipeline swappable and testable.
package pipeline
