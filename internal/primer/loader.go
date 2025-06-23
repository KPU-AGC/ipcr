// internal/primer/loader.go
package primer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadTSV reads a whitespace‑separated file with
// id forward reverse min max
// min / max are optional (0 if absent).
func LoadTSV(path string) ([]Pair, error) {
	fh, err := os.Open(path)
	if err != nil { return nil, err }
	defer fh.Close()

	var list []Pair
	sc := bufio.NewScanner(fh)
	ln := 0
	for sc.Scan() {
		ln++
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' { continue }
		f := strings.Fields(line)
		if len(f) < 3 || len(f) == 4 || len(f) > 5 {
			return nil, fmt.Errorf("%s:%d bad field count", path, ln)
		}
		p := Pair{
			ID:      f[0],
			Forward: strings.ToUpper(f[1]),
			Reverse: strings.ToUpper(f[2]),
		}
		if len(f) >= 4 {
			fmt.Sscan(f[3], &p.MinProduct)
		}
		if len(f) == 5 {
			fmt.Sscan(f[4], &p.MaxProduct)
		}
		list = append(list, p)
	}
	if err := sc.Err(); err != nil { return nil, err }
	return list, nil
}
