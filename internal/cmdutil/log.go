// internal/cmdutil/log.go  (NEW)
package cmdutil

import (
	"fmt"
	"io"
)

func Warnf(dst io.Writer, quiet bool, format string, a ...any) {
	if quiet {
		return
	}
	_, _ = fmt.Fprintf(dst, "WARN: "+format+"\n", a...)
}
