package pdf

import (
	"bytes"
	"fmt"
	"os"
)

// ExtractPage extracts a single page from a document and saves it to a file
func ExtractPage(doc *Document, pageNum int, outputFile string) error {
	if pageNum < 1 || pageNum > doc.NumPages() {
		return fmt.Errorf("page %d out of range", pageNum)
	}

	page := doc.Pages[pageNum-1]

	// Create a minimal PDF with just this page
	var buf bytes.Buffer

	// Write header
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("%\xe2\xe3\xcf\xd3\n") // Binary marker

	// Object 1: Catalog
	obj1Offset := buf.Len()
	buf.WriteString("1 0 obj\n")
	buf.WriteString("<< /Type /Catalog /Pages 2 0 R >>\n")
	buf.WriteString("endobj\n")

	// Object 2: Pages
	obj2Offset := buf.Len()
	buf.WriteString("2 0 obj\n")
	buf.WriteString("<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n")
	buf.WriteString("endobj\n")

	// Object 3: Page
	obj3Offset := buf.Len()
	buf.WriteString("3 0 obj\n")
	buf.WriteString("<< /Type /Page /Parent 2 0 R ")
	buf.WriteString(fmt.Sprintf("/MediaBox [%g %g %g %g] ",
		page.MediaBox.LLX, page.MediaBox.LLY,
		page.MediaBox.URX, page.MediaBox.URY))

	// Get page contents
	contents, err := page.GetContents()
	if err == nil && len(contents) > 0 {
		buf.WriteString("/Contents 4 0 R ")
	}
	buf.WriteString(">>\n")
	buf.WriteString("endobj\n")

	// Object 4: Contents (if any)
	obj4Offset := 0
	if len(contents) > 0 {
		obj4Offset = buf.Len()
		buf.WriteString("4 0 obj\n")
		buf.WriteString(fmt.Sprintf("<< /Length %d >>\n", len(contents)))
		buf.WriteString("stream\n")
		buf.Write(contents)
		buf.WriteString("\nendstream\n")
		buf.WriteString("endobj\n")
	}

	// Write xref
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	if obj4Offset > 0 {
		buf.WriteString("0 5\n")
	} else {
		buf.WriteString("0 4\n")
	}
	buf.WriteString("0000000000 65535 f \n")
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj1Offset))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj2Offset))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj3Offset))
	if obj4Offset > 0 {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj4Offset))
	}

	// Write trailer
	buf.WriteString("trailer\n")
	if obj4Offset > 0 {
		buf.WriteString("<< /Size 5 /Root 1 0 R >>\n")
	} else {
		buf.WriteString("<< /Size 4 /Root 1 0 R >>\n")
	}
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return os.WriteFile(outputFile, buf.Bytes(), 0644)
}

// MergeDocuments merges multiple PDF documents into one
func MergeDocuments(docs []*Document, outputFile string) error {
	if len(docs) == 0 {
		return fmt.Errorf("no documents to merge")
	}

	var buf bytes.Buffer

	// Write header
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("%\xe2\xe3\xcf\xd3\n")

	// Count total pages
	totalPages := 0
	for _, doc := range docs {
		totalPages += doc.NumPages()
	}

	// Object 1: Catalog
	obj1Offset := buf.Len()
	buf.WriteString("1 0 obj\n")
	buf.WriteString("<< /Type /Catalog /Pages 2 0 R >>\n")
	buf.WriteString("endobj\n")

	// Object 2: Pages
	obj2Offset := buf.Len()
	buf.WriteString("2 0 obj\n")
	buf.WriteString("<< /Type /Pages /Kids [")
	for i := 0; i < totalPages; i++ {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%d 0 R", 3+i*2))
	}
	buf.WriteString(fmt.Sprintf("] /Count %d >>\n", totalPages))
	buf.WriteString("endobj\n")

	// Write page objects
	offsets := []int{obj1Offset, obj2Offset}
	objNum := 3

	for _, doc := range docs {
		for _, page := range doc.Pages {
			// Page object
			pageOffset := buf.Len()
			offsets = append(offsets, pageOffset)

			buf.WriteString(fmt.Sprintf("%d 0 obj\n", objNum))
			buf.WriteString("<< /Type /Page /Parent 2 0 R ")
			buf.WriteString(fmt.Sprintf("/MediaBox [%g %g %g %g] ",
				page.MediaBox.LLX, page.MediaBox.LLY,
				page.MediaBox.URX, page.MediaBox.URY))

			contents, err := page.GetContents()
			if err == nil && len(contents) > 0 {
				buf.WriteString(fmt.Sprintf("/Contents %d 0 R ", objNum+1))
			}
			buf.WriteString(">>\n")
			buf.WriteString("endobj\n")
			objNum++

			// Contents object
			if len(contents) > 0 {
				contentsOffset := buf.Len()
				offsets = append(offsets, contentsOffset)

				buf.WriteString(fmt.Sprintf("%d 0 obj\n", objNum))
				buf.WriteString(fmt.Sprintf("<< /Length %d >>\n", len(contents)))
				buf.WriteString("stream\n")
				buf.Write(contents)
				buf.WriteString("\nendstream\n")
				buf.WriteString("endobj\n")
				objNum++
			}
		}
	}

	// Write xref
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", len(offsets)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	// Write trailer
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(offsets)+1))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return os.WriteFile(outputFile, buf.Bytes(), 0644)
}
