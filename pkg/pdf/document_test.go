package pdf

import (
	"bytes"
	"testing"
)

// TestNewDocument tests creating a new document from PDF data
func TestNewDocument(t *testing.T) {
	// Create a minimal valid PDF
	pdfData := createMinimalPDF()

	doc, err := NewDocument(pdfData)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.Close()

	if doc.Version == "" {
		t.Error("Document version should not be empty")
	}
}

// TestInvalidPDF tests handling of invalid PDF data
func TestInvalidPDF(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"not pdf", []byte("This is not a PDF file")},
		{"invalid header", []byte("%PDF-")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDocument(tt.data)
			if err == nil {
				t.Error("Expected error for invalid PDF data")
			}
		})
	}
}

// TestDocumentInfo tests document info extraction
func TestDocumentInfo(t *testing.T) {
	pdfData := createMinimalPDF()

	doc, err := NewDocument(pdfData)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.Close()

	info := doc.GetInfo()

	// Check that PDFVersion is set
	if info.PDFVersion == "" {
		t.Error("PDF version should not be empty")
	}
}

// TestNumPages tests page count
func TestNumPages(t *testing.T) {
	pdfData := createMinimalPDF()

	doc, err := NewDocument(pdfData)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.Close()

	numPages := doc.NumPages()
	if numPages < 0 {
		t.Errorf("NumPages should be >= 0, got %d", numPages)
	}
}

// TestGetPage tests page retrieval
func TestGetPage(t *testing.T) {
	pdfData := createMinimalPDF()

	doc, err := NewDocument(pdfData)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.Close()

	// Test invalid page numbers
	_, err = doc.GetPage(0)
	if err == nil {
		t.Error("Expected error for page 0")
	}

	_, err = doc.GetPage(-1)
	if err == nil {
		t.Error("Expected error for negative page number")
	}

	_, err = doc.GetPage(1000000)
	if err == nil {
		t.Error("Expected error for page number out of range")
	}
}

// TestRectangle tests rectangle operations
func TestRectangle(t *testing.T) {
	r := Rectangle{LLX: 0, LLY: 0, URX: 612, URY: 792}

	if r.Width() != 612 {
		t.Errorf("Expected width 612, got %f", r.Width())
	}

	if r.Height() != 792 {
		t.Errorf("Expected height 792, got %f", r.Height())
	}
}

// TestParsePDFDate tests PDF date parsing
func TestParsePDFDate(t *testing.T) {
	tests := []struct {
		input    string
		expected int // year
	}{
		{"D:20240101120000", 2024},
		{"D:20231225", 2023},
		{"20220615", 2022},
		{"", 0},
		{"D:", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parsePDFDate(tt.input)
			if result.Year() != tt.expected {
				t.Errorf("Expected year %d, got %d", tt.expected, result.Year())
			}
		})
	}
}

// TestDocumentClose tests document closing
func TestDocumentClose(t *testing.T) {
	pdfData := createMinimalPDF()

	doc, err := NewDocument(pdfData)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	err = doc.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}

	// Verify internal state is cleared
	if doc.data != nil {
		t.Error("Document data should be nil after close")
	}
}

// createMinimalPDF creates a minimal valid PDF for testing
func createMinimalPDF() []byte {
	var buf bytes.Buffer

	// PDF header
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("%\xe2\xe3\xcf\xd3\n") // Binary marker

	// Catalog (object 1)
	obj1Offset := buf.Len()
	buf.WriteString("1 0 obj\n")
	buf.WriteString("<< /Type /Catalog /Pages 2 0 R >>\n")
	buf.WriteString("endobj\n")

	// Pages (object 2)
	obj2Offset := buf.Len()
	buf.WriteString("2 0 obj\n")
	buf.WriteString("<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n")
	buf.WriteString("endobj\n")

	// Page (object 3)
	obj3Offset := buf.Len()
	buf.WriteString("3 0 obj\n")
	buf.WriteString("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\n")
	buf.WriteString("endobj\n")

	// Cross-reference table
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString("0 4\n")
	buf.WriteString("0000000000 65535 f \n")
	buf.WriteString(formatXRefEntry(obj1Offset))
	buf.WriteString(formatXRefEntry(obj2Offset))
	buf.WriteString(formatXRefEntry(obj3Offset))

	// Trailer
	buf.WriteString("trailer\n")
	buf.WriteString("<< /Size 4 /Root 1 0 R >>\n")
	buf.WriteString("startxref\n")
	buf.WriteString(formatInt(xrefOffset))
	buf.WriteString("\n%%EOF\n")

	return buf.Bytes()
}

func formatXRefEntry(offset int) string {
	return formatIntPadded(offset, 10) + " 00000 n \n"
}

func formatIntPadded(n, width int) string {
	s := formatInt(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
