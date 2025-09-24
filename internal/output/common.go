package output

// TSVHeader is the canonical header row for text/TSV outputs.
// Keep this as the single source of truth; all writers should use it.
const TSVHeader = "source_file\tsequence_id\texperiment_id\tstart\tend\tlength\ttype\tfwd_mm\trev_mm\tfwd_mm_i\trev_mm_i"
