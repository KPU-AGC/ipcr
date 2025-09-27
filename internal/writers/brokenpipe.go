package writers

import (
	"errors"
	"io"
	"syscall"
)

// IsBrokenPipe reports whether an error is a broken pipe / closed pipe.
// Useful when downstream consumers (like `head`) close early.
func IsBrokenPipe(err error) bool {
	return err != nil && (errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe))
}
