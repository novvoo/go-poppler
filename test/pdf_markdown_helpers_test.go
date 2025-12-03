package test

import (
	"testing"
)

// Note: isAllCaps, escapeMarkdown, and processTextToMarkdown are internal functions
// in the pdf package and cannot be tested directly from the test package.
// They are tested indirectly through the public MarkdownWriter.Write() method
// in the existing pdf_markdown_test.go file.

func TestMarkdownInternalFunctions(t *testing.T) {
	t.Skip("Internal markdown helper functions are tested through public API in pdf_markdown_test.go")
}
