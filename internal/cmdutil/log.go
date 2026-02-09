// internal/cmdutil/log.go
package cmdutil

import (
	"fmt"
	"io"
)

func Warnf(dst io.Writer, quiet bool, format string, a ...any) {
	if quiet || dst == nil {
		return
	}
	_, _ = fmt.Fprintf(dst, "WARN: "+format+"\n", a...)
}
