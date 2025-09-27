// Package engine contains the PCR simulation core. It never imports app, writers,
// cli, or pipeline; keep it domain-only.
//
// External outputs must not depend on the internal shape here — use pkg/api
// for stable wire types (JSON/JSONL v1).
package engine
