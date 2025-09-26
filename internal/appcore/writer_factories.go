// internal/appcore/writer_factories.go
package appcore

import (
	"io"

	"ipcr/internal/engine"
	"ipcr/internal/probeoutput"
	"ipcr/internal/writers"
)

// ---------------- Product writer ----------------

type ProductWriterFactory struct {
	Format   string
	Sort     bool
	Header   bool
	Pretty   bool
	Products bool
}

func NewProductWriterFactory(format string, sort, header, pretty, products bool) ProductWriterFactory {
	return ProductWriterFactory{Format: format, Sort: sort, Header: header, Pretty: pretty, Products: products}
}

func (w ProductWriterFactory) NeedSites() bool {
	// Pretty blocks require sites in the Product.
	return w.Format == "text" && w.Pretty
}

func (w ProductWriterFactory) NeedSeq() bool {
	// We need amplicon sequences if --products, FASTA output, or pretty text.
	if w.Products {
		return true
	}
	if w.Format == "fasta" {
		return true
	}
	if w.Format == "text" && w.Pretty {
		return true
	}
	return false
}

func (w ProductWriterFactory) Start(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	return writers.StartProductWriter(out, w.Format, w.Sort, w.Header, w.Pretty, bufSize)
}

// ---------------- Annotated writer (probe) ----------------

type AnnotatedWriterFactory struct {
	Format string
	Sort   bool
	Header bool
	Pretty bool
}

func NewAnnotatedWriterFactory(format string, sort, header, pretty bool) AnnotatedWriterFactory {
	return AnnotatedWriterFactory{Format: format, Sort: sort, Header: header, Pretty: pretty}
}

func (w AnnotatedWriterFactory) NeedSites() bool {
	// Pretty-annotated text requires sites for the primer bars.
	return w.Pretty
}

func (w AnnotatedWriterFactory) NeedSeq() bool {
	// Probe annotation requires the amplicon sequence to search the probe.
	return true
}

func (w AnnotatedWriterFactory) Start(out io.Writer, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	return writers.StartAnnotatedWriter(out, w.Format, w.Sort, w.Header, w.Pretty, bufSize)
}
