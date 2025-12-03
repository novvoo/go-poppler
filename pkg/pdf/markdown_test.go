package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarkdownWriter(t *testing.T) {
	// This is a basic test structure
	// In real usage, you would test with actual PDF files
	
	t.Run("isAllCaps", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bool
		}{
			{"HELLO WORLD", true},
			{"Hello World", false},
			{"HELLO123", true},
			{"hello", false},
			{"123", false},
			{"HELLO!", true},
		}
		
		for _, tt := range tests {
			result := isAllCaps(tt.input)
			if result != tt.expected {
				t.Errorf("isAllCaps(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		}
	})
	
	t.Run("isBulletPoint", func(t *testing.T) {
		w := &MarkdownWriter{}
		
		tests := []struct {
			input    string
			expected bool
		}{
			{"• Item one", true},
			{"- Item two", true},
			{"* Item three", true},
			{"1. Numbered item", true},
			{"2) Another numbered", true},
			{"Regular text", false},
			{"", false},
		}
		
		for _, tt := range tests {
			result := w.isBulletPoint(tt.input)
			if result != tt.expected {
				t.Errorf("isBulletPoint(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		}
	})
	
	t.Run("formatListItem", func(t *testing.T) {
		w := &MarkdownWriter{}
		
		tests := []struct {
			input    string
			expected string
		}{
			{"• Item one", "- Item one"},
			{"- Item two", "- Item two"},
			{"1. Numbered", "1. Numbered"},
			{"2) Another", "2. Another"},
		}
		
		for _, tt := range tests {
			result := w.formatListItem(tt.input)
			if result != tt.expected {
				t.Errorf("formatListItem(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})
	
	t.Run("isCodeLine", func(t *testing.T) {
		w := &MarkdownWriter{}
		
		tests := []struct {
			input    string
			expected bool
		}{
			{"    code line", true},
			{"        more indented", true},
			{"\tcode with tab", true},
			{"no indent", false},
			{"  two spaces", false},
		}
		
		for _, tt := range tests {
			result := w.isCodeLine(tt.input)
			if result != tt.expected {
				t.Errorf("isCodeLine(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		}
	})
	
	t.Run("detectHeadingLevel", func(t *testing.T) {
		w := &MarkdownWriter{
			options: MarkdownOptions{HeadingDetection: true},
		}
		
		tests := []struct {
			input    string
			expected int
		}{
			{"1. Introduction", 1},
			{"1.1 Overview", 2},
			{"1.1.1 Details", 3},
			{"CHAPTER ONE", 1},
			{"Regular Heading", 2},
		}
		
		for _, tt := range tests {
			result := w.detectHeadingLevel(tt.input)
			if result != tt.expected {
				t.Errorf("detectHeadingLevel(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	})
	
	t.Run("processTextToMarkdown", func(t *testing.T) {
		w := &MarkdownWriter{
			options: MarkdownOptions{HeadingDetection: true},
		}
		
		input := `INTRODUCTION

This is a paragraph with some text.
It continues on the next line.

• First bullet point
• Second bullet point

1. First numbered item
2. Second numbered item

    code example
    more code

Regular paragraph again.`
		
		result := w.processTextToMarkdown(input)
		
		// Check that output contains markdown elements
		if !strings.Contains(result, "# INTRODUCTION") {
			t.Error("Expected heading to be formatted")
		}
		if !strings.Contains(result, "- First bullet point") {
			t.Error("Expected bullet points to be formatted")
		}
		if !strings.Contains(result, "1. First numbered item") {
			t.Error("Expected numbered list to be preserved")
		}
		if !strings.Contains(result, "```") {
			t.Error("Expected code block to be formatted")
		}
	})
}

func TestMarkdownOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		doc := &Document{} // Mock document
		options := MarkdownOptions{}
		
		writer := NewMarkdownWriter(doc, options)
		
		if writer.options.PageSeparator != "\n\n---\n\n" {
			t.Errorf("Expected default page separator, got %q", writer.options.PageSeparator)
		}
		if writer.options.ImagePrefix != "image" {
			t.Errorf("Expected default image prefix, got %q", writer.options.ImagePrefix)
		}
	})
	
	t.Run("custom options", func(t *testing.T) {
		doc := &Document{} // Mock document
		options := MarkdownOptions{
			PageSeparator: "\n\n***\n\n",
			ImagePrefix:   "fig",
		}
		
		writer := NewMarkdownWriter(doc, options)
		
		if writer.options.PageSeparator != "\n\n***\n\n" {
			t.Errorf("Expected custom page separator, got %q", writer.options.PageSeparator)
		}
		if writer.options.ImagePrefix != "fig" {
			t.Errorf("Expected custom image prefix, got %q", writer.options.ImagePrefix)
		}
	})
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`Hello "World"`, `Hello \"World\"`},
		{"Simple text", "Simple text"},
		{`Quote: "test"`, `Quote: \"test\"`},
	}
	
	for _, tt := range tests {
		result := escapeMarkdown(tt.input)
		if result != tt.expected {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMarkdownWriterWrite(t *testing.T) {
	// This would require a real PDF document to test properly
	// For now, we just test that the writer can be created
	
	t.Run("writer creation", func(t *testing.T) {
		doc := &Document{} // Mock document
		options := MarkdownOptions{
			FirstPage:        1,
			LastPage:         0,
			HeadingDetection: true,
		}
		
		writer := NewMarkdownWriter(doc, options)
		if writer == nil {
			t.Error("Expected writer to be created")
		}
		
		// Test that Write doesn't panic with empty document
		var buf bytes.Buffer
		// Note: This will fail with a real implementation since doc is empty
		// In production, you'd use a proper test PDF
		_ = writer.Write(&buf)
	})
}
