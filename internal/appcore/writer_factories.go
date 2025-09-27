// internal/appcore/writer_factories.go
package appcore

import (
	"io"
	"ipcr-core/engine"
	"ipcr/internal/nestedoutput"
	"ipcr/internal/output"
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
	return w.Format == output.FormatText && w.Pretty
}

func (w ProductWriterFactory) NeedSeq() bool {
	if w.Products {
		return true
	}
	if w.Format == output.FormatFASTA {
		return true
	}
	if w.Format == output.FormatText && w.Pretty {
		return true
	}
	return false
}

func (w ProductWriterFactory) Start(out io.Writer, bufSize int) (chan<- engine.Product, <-chan error) {
	return writers.StartProductWriter(out, w.Format, w.Sort, w.Header, w.Pretty, bufSize)
}

// ---------------- Annotated writer ----------------

type AnnotatedWriterFactory struct {
	Format string
	Sort   bool
	Header bool
	Pretty bool
}

func NewAnnotatedWriterFactory(format string, sort, header, pretty bool) AnnotatedWriterFactory {
	return AnnotatedWriterFactory{Format: format, Sort: sort, Header: header, Pretty: pretty}
}

func (w AnnotatedWriterFactory) NeedSites() bool { return w.Pretty }
func (w AnnotatedWriterFactory) NeedSeq() bool   { return true } // probe overlay requires sequence

func (w AnnotatedWriterFactory) Start(out io.Writer, bufSize int) (chan<- probeoutput.AnnotatedProduct, <-chan error) {
	return writers.StartAnnotatedWriter(out, w.Format, w.Sort, w.Header, w.Pretty, bufSize)
}

// ---------------- Nested writer ----------------

type NestedWriterFactory struct {
	Format string
	Sort   bool
	Header bool
}

func NewNestedWriterFactory(format string, sort, header bool) NestedWriterFactory {
	return NestedWriterFactory{Format: format, Sort: sort, Header: header}
}

// Pretty rendering for nested isn’t implemented (sites aren’t needed here).
func (w NestedWriterFactory) NeedSites() bool { return false }

// IMPORTANT: visitor needs Product.Seq to rescan inner primers.
func (w NestedWriterFactory) NeedSeq() bool { return true }

func (w NestedWriterFactory) Start(out io.Writer, bufSize int) (chan<- nestedoutput.NestedProduct, <-chan error) {
	return writers.StartNestedWriter(out, w.Format, w.Sort, w.Header, bufSize)
}
