// internal/output/text.go  (REPLACE)
package output

import (
	"io"

	"ipcr-core/engine"
)

func writeRowTSV(w io.Writer, p engine.Product) error {
	_, err := io.WriteString(w, FormatBaseRowTSV(p)+"\n")
	return err
}

// New: renderer-capable streaming writer for text mode
func StreamTextWithRenderer(w io.Writer, in <-chan engine.Product, header bool, prettyMode bool, render func(engine.Product) string) error {
	if header {
		if _, err := io.WriteString(w, TSVHeader+"\n"); err != nil {
			return err
		}
	}
	for p := range in {
		if err := writeRowTSV(w, p); err != nil {
			return err
		}
		if prettyMode {
			if _, err := io.WriteString(w, render(p)); err != nil {
				return err
			}
		}
	}
	return nil
}

// New: renderer-capable buffered writer for text mode
func WriteTextWithRenderer(w io.Writer, list []engine.Product, header bool, prettyMode bool, render func(engine.Product) string) error {
	if header {
		if _, err := io.WriteString(w, TSVHeader+"\n"); err != nil {
			return err
		}
	}
	for _, p := range list {
		if err := writeRowTSV(w, p); err != nil {
			return err
		}
		if prettyMode {
			if _, err := io.WriteString(w, render(p)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Backward-compat wrappers (use default renderer wired in output package)
func StreamText(w io.Writer, in <-chan engine.Product, header bool, prettyMode bool) error {
	return StreamTextWithRenderer(w, in, header, prettyMode, func(p engine.Product) string { return "" })
}

func WriteText(w io.Writer, list []engine.Product, header bool, prettyMode bool) error {
	return WriteTextWithRenderer(w, list, header, prettyMode, func(p engine.Product) string { return "" })
}
