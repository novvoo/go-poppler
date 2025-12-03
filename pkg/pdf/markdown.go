// Package pdf provides Markdown output capabilities
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// MarkdownOptions contains Markdown output options
type MarkdownOptions struct {
	FirstPage        int
	LastPage         int
	IncludeImages    bool
	ImagePrefix      string
	ExtractImages    bool
	PageSeparator    string
	HeadingDetection bool
}

// MarkdownWriter generates Markdown output from PDF
type MarkdownWriter struct {
	doc     *Document
	options MarkdownOptions
}

// NewMarkdownWriter creates a new Markdown writer
func NewMarkdownWriter(doc *Document, options MarkdownOptions) *MarkdownWriter {
	if options.PageSeparator == "" {
		options.PageSeparator = "\n\n---\n\n"
	}
	if options.ImagePrefix == "" {
		options.ImagePrefix = "image"
	}
	return &MarkdownWriter{
		doc:     doc,
		options: options,
	}
}

// Write generates Markdown output
func (w *MarkdownWriter) Write(output io.Writer) error {
	firstPage := w.options.FirstPage
	lastPage := w.options.LastPage

	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > w.doc.NumPages() {
		lastPage = w.doc.NumPages()
	}

	// Get document info and write as front matter
	info := w.doc.GetInfo()
	if info.Title != "" || info.Author != "" || info.Subject != "" {
		fmt.Fprintf(output, "---\n")
		if info.Title != "" {
			fmt.Fprintf(output, "title: %s\n", escapeMarkdown(info.Title))
		}
		if info.Author != "" {
			fmt.Fprintf(output, "author: %s\n", escapeMarkdown(info.Author))
		}
		if info.Subject != "" {
			fmt.Fprintf(output, "subject: %s\n", escapeMarkdown(info.Subject))
		}
		fmt.Fprintf(output, "---\n\n")
	}

	// Extract text from each page
	extractor := NewTextExtractor(w.doc)
	imageExtractor := NewImageExtractor(w.doc)

	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		if pageNum > firstPage {
			fmt.Fprintf(output, "%s", w.options.PageSeparator)
		}

		text, err := extractor.ExtractPageText(pageNum)
		if err != nil {
			continue
		}

		// Process text into markdown
		markdown := w.processTextToMarkdown(text)
		fmt.Fprintf(output, "%s\n", markdown)

		// Extract and reference images if requested
		if w.options.IncludeImages {
			images, err := imageExtractor.ExtractImages(pageNum, pageNum)
			if err == nil && len(images) > 0 {
				fmt.Fprintf(output, "\n")
				for i, img := range images {
					imageName := fmt.Sprintf("%s-p%d-%d.png", w.options.ImagePrefix, pageNum, i+1)

					// Save image if extraction is enabled
					if w.options.ExtractImages {
						data, err := imageExtractor.GetImageData(img, "png")
						if err == nil {
							// Write image data to file
							if err := writeImageFile(imageName, data); err != nil {
								// Log error but continue
								continue
							}
						}
					}

					fmt.Fprintf(output, "![Image %d](%s)\n\n", i+1, imageName)
				}
			}
		}
	}

	return nil
}

// processTextToMarkdown converts plain text to markdown with intelligent formatting
func (w *MarkdownWriter) processTextToMarkdown(text string) string {
	var buf bytes.Buffer
	lines := strings.Split(text, "\n")

	inList := false
	inCodeBlock := false
	prevLineEmpty := true

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines but track them
		if trimmed == "" {
			if !prevLineEmpty && !inCodeBlock {
				buf.WriteString("\n")
			}
			prevLineEmpty = true
			continue
		}

		// Detect and format headings
		if w.options.HeadingDetection && w.isHeading(trimmed, i, lines) {
			if inList {
				inList = false
			}
			level := w.detectHeadingLevel(trimmed)
			buf.WriteString(fmt.Sprintf("%s %s\n\n", strings.Repeat("#", level), trimmed))
			prevLineEmpty = false
			continue
		}

		// Detect bullet points and numbered lists
		if w.isBulletPoint(trimmed) {
			if !inList && !prevLineEmpty {
				buf.WriteString("\n")
			}
			inList = true
			buf.WriteString(w.formatListItem(trimmed))
			buf.WriteString("\n")
			prevLineEmpty = false
			continue
		}

		// Detect code blocks (lines with significant indentation)
		if w.isCodeLine(line) && !inList {
			if !inCodeBlock {
				buf.WriteString("```\n")
				inCodeBlock = true
			}
			buf.WriteString(line)
			buf.WriteString("\n")
			prevLineEmpty = false
			continue
		} else if inCodeBlock {
			buf.WriteString("```\n\n")
			inCodeBlock = false
		}

		// Regular paragraph text
		if inList && !w.isBulletPoint(trimmed) {
			inList = false
			buf.WriteString("\n")
		}

		// Add line with proper spacing
		if !prevLineEmpty && !inList {
			buf.WriteString(" ")
		}
		buf.WriteString(trimmed)

		// Check if next line is empty or different type
		if i+1 < len(lines) {
			nextTrimmed := strings.TrimSpace(lines[i+1])
			if nextTrimmed == "" || w.isHeading(nextTrimmed, i+1, lines) ||
				w.isBulletPoint(nextTrimmed) != w.isBulletPoint(trimmed) {
				buf.WriteString("\n\n")
				prevLineEmpty = true
			} else {
				prevLineEmpty = false
			}
		} else {
			buf.WriteString("\n")
		}
	}

	// Close any open code block
	if inCodeBlock {
		buf.WriteString("```\n")
	}

	return buf.String()
}

// isHeading detects if a line is likely a heading
func (w *MarkdownWriter) isHeading(line string, index int, lines []string) bool {
	if !w.options.HeadingDetection {
		return false
	}

	// Check length (headings are usually short)
	if len(line) > 100 {
		return false
	}

	// Check if all caps (common for headings)
	if isAllCaps(line) && len(line) > 3 {
		return true
	}

	// Check if followed by empty line (common for headings)
	if index+1 < len(lines) && strings.TrimSpace(lines[index+1]) == "" {
		// Check if line ends with punctuation (headings usually don't)
		if !strings.HasSuffix(line, ".") && !strings.HasSuffix(line, ",") {
			// Check if line is relatively short
			if len(line) < 80 {
				return true
			}
		}
	}

	// Check for numbered headings like "1. Introduction" or "1.1 Overview"
	matched, _ := regexp.MatchString(`^\d+(\.\d+)*\.?\s+[A-Z]`, line)
	return matched
}

// detectHeadingLevel determines the heading level (1-6)
func (w *MarkdownWriter) detectHeadingLevel(line string) int {
	// Check for numbered sections
	if matched, _ := regexp.MatchString(`^\d+\.\s+`, line); matched {
		return 1 // Top level
	}
	if matched, _ := regexp.MatchString(`^\d+\.\d+\s+`, line); matched {
		return 2 // Second level
	}
	if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+\s+`, line); matched {
		return 3 // Third level
	}

	// All caps likely top level
	if isAllCaps(line) {
		return 1
	}

	// Default to level 2
	return 2
}

// isBulletPoint detects if a line is a bullet point or numbered list item
func (w *MarkdownWriter) isBulletPoint(line string) bool {
	// Check for bullet characters
	if len(line) > 0 {
		first := rune(line[0])
		if len(line) >= 3 {
			// Check for multi-byte UTF-8 characters
			r := []rune(line)
			if len(r) > 0 {
				first = r[0]
			}
		}
		if first == '•' || first == '◦' || first == '▪' || first == '▫' ||
			first == '■' || first == '□' || first == '●' || first == '○' ||
			first == '-' || first == '*' || first == '+' {
			return true
		}
	}

	// Check for numbered lists
	matched, _ := regexp.MatchString(`^\d+[\.\)]\s+`, line)
	return matched
}

// formatListItem formats a list item for markdown
func (w *MarkdownWriter) formatListItem(line string) string {
	// Replace bullet characters with markdown bullets
	if len(line) > 0 {
		r := []rune(line)
		if len(r) > 0 {
			first := r[0]
			if first == '•' || first == '◦' || first == '▪' || first == '▫' ||
				first == '■' || first == '□' || first == '●' || first == '○' {
				return "- " + strings.TrimSpace(string(r[1:]))
			}
		}
	}

	// Keep numbered lists as is
	if matched, _ := regexp.MatchString(`^\d+[\.\)]\s+`, line); matched {
		// Normalize to use period
		re := regexp.MustCompile(`^(\d+)\)\s+`)
		line = re.ReplaceAllString(line, "$1. ")
		return line
	}

	// Keep markdown bullets as is
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
		return line
	}

	return "- " + line
}

// isCodeLine detects if a line looks like code (significant indentation)
func (w *MarkdownWriter) isCodeLine(line string) bool {
	if len(line) == 0 {
		return false
	}

	// Count leading spaces
	spaces := 0
	for _, ch := range line {
		if ch == ' ' {
			spaces++
		} else if ch == '\t' {
			spaces += 4
		} else {
			break
		}
	}

	// Consider it code if indented by 4+ spaces
	return spaces >= 4
}

// isAllCaps checks if a string is all uppercase (ignoring numbers and punctuation)
func isAllCaps(s string) bool {
	hasLetter := false
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			return false
		}
		if ch >= 'A' && ch <= 'Z' {
			hasLetter = true
		}
	}
	return hasLetter
}

// escapeMarkdown escapes special markdown characters
func escapeMarkdown(s string) string {
	// Only escape in metadata, not in content
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// writeImageFile writes image data to a file
func writeImageFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// ConvertToMarkdown is a convenience function to convert a PDF file to Markdown
func ConvertToMarkdown(filename string, options MarkdownOptions) (string, error) {
	doc, err := Open(filename)
	if err != nil {
		return "", err
	}
	defer doc.Close()

	writer := NewMarkdownWriter(doc, options)
	var buf bytes.Buffer
	err = writer.Write(&buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
