package appcore

import "testing"

func TestProductWriterFactory_NeedSitesAndSeq(t *testing.T) {
	w := NewProductWriterFactory("text", false, false, true, false)
	if !w.NeedSites() {
		t.Fatal("pretty text must NeedSites")
	}
	if !w.NeedSeq() {
		t.Fatal("pretty text must NeedSeq")
	}

	w = NewProductWriterFactory("json", false, false, false, false)
	if w.NeedSites() || w.NeedSeq() {
		t.Fatal("json without products/pretty should not need sites/seq")
	}

	w = NewProductWriterFactory("fasta", false, false, false, false)
	if !w.NeedSeq() {
		t.Fatal("fasta must NeedSeq")
	}

	w = NewProductWriterFactory("json", false, false, false, true)
	if !w.NeedSeq() {
		t.Fatal("json + --products must NeedSeq")
	}
}

func TestAnnotatedWriterFactory_AlwaysNeedsSeq(t *testing.T) {
	w := NewAnnotatedWriterFactory("text", false, false, false)
	if !w.NeedSeq() {
		t.Fatal("annotated writer always needs seq for probe")
	}
	if w.NeedSites() {
		t.Fatal("no-pretty should not NeedSites")
	}

	w = NewAnnotatedWriterFactory("text", false, false, true)
	if !w.NeedSites() {
		t.Fatal("pretty annotated writer needs sites")
	}
}
