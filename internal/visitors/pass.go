package visitors

import "ipcr-core/engine"

// PassThrough returns the product unchanged.
type PassThrough struct{}

func (PassThrough) Visit(p engine.Product) (keep bool, out engine.Product, err error) {
	return true, p, nil
}
