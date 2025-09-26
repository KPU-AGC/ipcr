// internal/common/ids.go
package common

import (
	"strconv"
	"strings"
)

// SplitChunkSuffix extracts the base ID and the chunk's start offset if the
// input looks like "record_id:123-456". It returns base, start, ok.
func SplitChunkSuffix(id string) (string, int, bool) {
	colon := strings.LastIndex(id, ":")
	if colon == -1 || colon == len(id)-1 {
		return id, 0, false
	}
	suffix := id[colon+1:]
	dash := strings.IndexByte(suffix, '-')
	if dash == -1 {
		return id, 0, false
	}
	startStr := suffix[:dash]
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return id, 0, false
	}
	return id[:colon], start, true
}
