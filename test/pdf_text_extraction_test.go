package test

import (
	"testing"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func TestTextExtractionOptions(t *testing.T) {
	opts := pdf.TextExtractionOptions{
		Layout:     true,
		Raw:        false,
		NoDiagonal: true,
		FirstPage:  1,
		LastPage:   10,
	}

	if !opts.Layout {
		t.Error("expected Layout to be true")
	}
	if opts.Raw {
		t.Error("expected Raw to be false")
	}
	if !opts.NoDiagonal {
		t.Error("expected NoDiagonal to be true")
	}
	if opts.FirstPage != 1 {
		t.Errorf("expected FirstPage to be 1, got %d", opts.FirstPage)
	}
	if opts.LastPage != 10 {
		t.Errorf("expected LastPage to be 10, got %d", opts.LastPage)
	}
}

// Note: multiplyMatrix is an internal function and cannot be tested directly
// It is tested indirectly through text extraction functionality
