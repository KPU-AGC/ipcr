#!/usr/bin/env bash
set -euo pipefail

# 1) Fix ineffectual assignments in core/engine/engine.go (unused minL/maxL in Simulate).
sed -i -E \
  -e '/^\s*minL := p\.MinProduct$/d' \
  -e '/^\s*maxL := p\.MaxProduct$/d' \
  -e '/^\s*if minL == 0 \{ minL = e\.cfg\.MinLen \}$/d' \
  -e '/^\s*if maxL == 0 \{ maxL = e\.cfg\.MaxLen \}$/d' \
  core/engine/engine.go

# 2) Preallocate where the linter asked.
# core/fasta/reader_test.go: ids := make([]string, 0, 2)
sed -i -E 's/^(\s*)var ids \[\]string/\1ids := make([]string, 0, 2)/' core/fasta/reader_test.go

# core/oligo/validate.go: var out []rune → out := make([]rune, 0, len(s))
sed -i -E 's/^\s*var out \[\]rune$/\tout := make([]rune, 0, len(s))/' core/oligo/validate.go

# 3) Make core’s depguard only enforce the core boundary (no “app-hygiene” in core).
cat > core/.golangci.yml <<'YAML'
run:
  timeout: 5m
  issues-exit-code: 1
  tests: true

linters:
  enable:
    - depguard
    - errcheck
    - gofumpt
    - govet
    - gocritic
    - misspell
    - prealloc
    - exportloopref

linters-settings:
  gofumpt:
    extra-rules: true
  depguard:
    rules:
      core-boundary:
        files: ["**/*.go"]
        allow:
          - $gostd
          - "^ipcr-core/"
          - "^github.com/"
          - "^golang.org/"
          - "^gopkg.in/"
        deny:
          - pkg: "^ipcr(/|$)"
            desc: "ipcr-core must not import app-side packages (ipcr/*)"
  errcheck:
    exclude-functions:
      - fmt.Fprint
      - fmt.Fprintf
      - fmt.Fprintln
      - os.Remove
      - (*bufio.Writer).Flush

issues:
  exclude-use-default: false
YAML

# 4) Silence Setup-Go cache warning in CI (no go.sum yet at restore time).
#    Flip 'cache: true' → 'cache: false' for the "Setup Go (tests)" step.
perl -0777 -pi -e \
's/(Setup Go \(tests\)[^\n]*\n\s*uses: actions\/setup-go@v5\n\s*with:\n\s*go-version:\s*'\''1\.22\.x'\''\n\s*check-latest:\s*true)\n\s*cache:\s*true/\1\n          cache: false/s' \
.github/workflows/ci.yml

echo "Fixes applied. Now run:"
echo "  GOWORK=off golangci-lint run --timeout=5m"
echo "  go test ./..."
