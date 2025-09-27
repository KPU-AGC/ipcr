// internal/output/rows.go  (NEW)
package output

import (
	"fmt"
	"ipcr-core/engine"
	"strconv"
	"strings"
)

func IntsCSV(a []int) string {
	if len(a) == 0 { return "" }
	ss := make([]string, len(a))
	for i, v := range a { ss[i] = strconv.Itoa(v) }
	return strings.Join(ss, ",")
}

// FormatBaseRowTSV returns the 11 base columns (no trailing newline).
func FormatBaseRowTSV(p engine.Product) string {
	return fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\t%s",
		p.SourceFile, p.SequenceID, p.ExperimentID,
		p.Start, p.End, p.Length, p.Type,
		p.FwdMM, p.RevMM,
		IntsCSV(p.FwdMismatchIdx), IntsCSV(p.RevMismatchIdx),
	)
}
