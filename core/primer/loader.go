// core/primer/loader.go
package primer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadTSV(path string) ([]Pair, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()

	var list []Pair
	sc := bufio.NewScanner(fh)
	ln := 0
	for sc.Scan() {
		ln++
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		f := strings.Fields(line)
		// Accept 3 (id fwd rev), 4 (… min), or 5 (… min max) fields.
		if len(f) < 3 || len(f) > 5 {
			return nil, fmt.Errorf("%s:%d bad field count", path, ln)
		}
		p := Pair{
			ID:      f[0],
			Forward: strings.ToUpper(f[1]),
			Reverse: strings.ToUpper(f[2]),
		}
		if len(f) >= 4 {
			if _, err := fmt.Sscan(f[3], &p.MinProduct); err != nil {
				return nil, fmt.Errorf("%s:%d bad min: %v", path, ln, err)
			}
		}
		if len(f) == 5 {
			if _, err := fmt.Sscan(f[4], &p.MaxProduct); err != nil {
				return nil, fmt.Errorf("%s:%d bad max: %v", path, ln, err)
			}
		}
		list = append(list, p)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
