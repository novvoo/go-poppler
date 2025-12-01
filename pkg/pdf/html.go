// Package pdf provides HTML output capabilities
package pdf

import (
	"fmt"
	"io"
	"strings"
)

// HTMLOptions contains HTML output options
type HTMLOptions struct {
	FirstPage    int
	LastPage     int
	Complex      bool
	SinglePage   bool
	IgnoreImages bool
	NoFrames     bool
	Zoom         float64
	XML          bool
}

// HTMLWriter generates HTML output from PDF
type HTMLWriter struct {
	doc     *Document
	options HTMLOptions
}

// NewHTMLWriter creates a new HTML writer
func NewHTMLWriter(doc *Document, options HTMLOptions) *HTMLWriter {
	if options.Zoom == 0 {
		options.Zoom = 1.5
	}
	return &HTMLWriter{
		doc:     doc,
		options: options,
	}
}

// Write generates HTML output
func (w *HTMLWriter) Write(output io.Writer) error {
	if w.options.XML {
		return w.writeXML(output)
	}
	return w.writeHTML(output)
}

// writeHTML generates HTML output
func (w *HTMLWriter) writeHTML(output io.Writer) error {
	firstPage := w.options.FirstPage
	lastPage := w.options.LastPage

	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > w.doc.NumPages() {
		lastPage = w.doc.NumPages()
	}

	// Get document info
	info := w.doc.GetInfo()

	// Write HTML header
	fmt.Fprintf(output, `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="generator" content="go-poppler">
<title>%s</title>
<style>
body { font-family: sans-serif; margin: 20px; }
.page { margin-bottom: 20px; padding: 20px; border: 1px solid #ccc; background: white; }
.page-header { color: #666; font-size: 12px; margin-bottom: 10px; }
p { margin: 0.5em 0; }
</style>
</head>
<body>
`, escapeHTML(info.Title))

	// Extract text from each page
	extractor := NewTextExtractor(w.doc)

	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		text, err := extractor.ExtractPageText(pageNum)
		if err != nil {
			continue
		}

		fmt.Fprintf(output, `<div class="page" id="page%d">
`, pageNum)
		fmt.Fprintf(output, `<div class="page-header">Page %d</div>
`, pageNum)

		// Convert text to HTML paragraphs
		lines := strings.Split(text, "\n")
		var paragraph strings.Builder
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				if paragraph.Len() > 0 {
					fmt.Fprintf(output, "<p>%s</p>\n", escapeHTML(paragraph.String()))
					paragraph.Reset()
				}
			} else {
				if paragraph.Len() > 0 {
					paragraph.WriteString(" ")
				}
				paragraph.WriteString(line)
			}
		}
		if paragraph.Len() > 0 {
			fmt.Fprintf(output, "<p>%s</p>\n", escapeHTML(paragraph.String()))
		}

		fmt.Fprintf(output, "</div>\n")
	}

	// Write HTML footer
	fmt.Fprintf(output, `</body>
</html>
`)

	return nil
}

// writeXML generates XML output
func (w *HTMLWriter) writeXML(output io.Writer) error {
	firstPage := w.options.FirstPage
	lastPage := w.options.LastPage

	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > w.doc.NumPages() {
		lastPage = w.doc.NumPages()
	}

	// Write XML header
	fmt.Fprintf(output, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE pdf2xml SYSTEM "pdf2xml.dtd">
<pdf2xml producer="go-poppler">
`)

	// Get document info
	info := w.doc.GetInfo()

	// Document info
	fmt.Fprintf(output, `<document pages="%d">
`, w.doc.NumPages())

	if info.Title != "" {
		fmt.Fprintf(output, `<title>%s</title>
`, escapeHTML(info.Title))
	}
	if info.Author != "" {
		fmt.Fprintf(output, `<author>%s</author>
`, escapeHTML(info.Author))
	}

	// Extract text from each page
	extractor := NewTextExtractor(w.doc)

	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		page, err := w.doc.GetPage(pageNum)
		if err != nil {
			continue
		}

		fmt.Fprintf(output, `<page number="%d" width="%.2f" height="%.2f">
`, pageNum, page.Width(), page.Height())

		text, err := extractor.ExtractPageText(pageNum)
		if err == nil && text != "" {
			lines := strings.Split(text, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					fmt.Fprintf(output, `<text>%s</text>
`, escapeHTML(line))
				}
			}
		}

		fmt.Fprintf(output, "</page>\n")
	}

	fmt.Fprintf(output, "</document>\n")
	fmt.Fprintf(output, "</pdf2xml>\n")

	return nil
}

// escapeHTML escapes special HTML characters
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
