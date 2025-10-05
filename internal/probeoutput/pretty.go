// internal/probeoutput/pretty.go
package probeoutput

import "ipcr/internal/pretty"

// RenderPretty renders the standard ASCII alignment block for the base Product
// inside an AnnotatedProduct (used by writers when --pretty is on).
func RenderPretty(ap AnnotatedProduct) string {
	return pretty.RenderProduct(ap.Product)
}

// RenderPrettyWithOptions allows custom pretty glyph/options when needed.
func RenderPrettyWithOptions(ap AnnotatedProduct, opt pretty.Options) string {
	return pretty.RenderProductWithOptions(ap.Product, opt)
}
