// internal/nestedoutput/pretty.go
package nestedoutput

import "ipcr/internal/pretty"

// RenderPretty renders the standard ASCII alignment block for the *outer* product.
func RenderPretty(np NestedProduct) string {
	return pretty.RenderProduct(np.Product)
}

// RenderPrettyWithOptions allows custom pretty glyph/options when needed.
func RenderPrettyWithOptions(np NestedProduct, opt pretty.Options) string {
	return pretty.RenderProductWithOptions(np.Product, opt)
}
