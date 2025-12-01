//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"os"
)

func main() {
	var buf bytes.Buffer

	// Header
	buf.WriteString("%PDF-1.4\n")

	// Object 1: Catalog
	obj1Offset := buf.Len()
	buf.WriteString("1 0 obj\n")
	buf.WriteString("<</Type/Catalog/Pages 2 0 R>>\n")
	buf.WriteString("endobj\n")

	// Object 2: Pages
	obj2Offset := buf.Len()
	buf.WriteString("2 0 obj\n")
	buf.WriteString("<</Type/Pages/Kids[3 0 R]/Count 1>>\n")
	buf.WriteString("endobj\n")

	// Object 3: Page
	obj3Offset := buf.Len()
	buf.WriteString("3 0 obj\n")
	buf.WriteString("<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R/Resources<<>>>>\n")
	buf.WriteString("endobj\n")

	// XRef table
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString("0 4\n")
	// Each entry is exactly 20 bytes: nnnnnnnnnn ggggg n/f EOL (where EOL is 2 chars: space + newline or CR+LF)
	buf.WriteString(fmt.Sprintf("%010d %05d f \r\n", 0, 65535))
	buf.WriteString(fmt.Sprintf("%010d %05d n \r\n", obj1Offset, 0))
	buf.WriteString(fmt.Sprintf("%010d %05d n \r\n", obj2Offset, 0))
	buf.WriteString(fmt.Sprintf("%010d %05d n \r\n", obj3Offset, 0))

	// Trailer
	buf.WriteString("trailer\n")
	buf.WriteString("<</Size 4/Root 1 0 R>>\n")
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF")

	os.WriteFile("test.pdf", buf.Bytes(), 0644)
	fmt.Printf("Created test.pdf with xref at offset %d\n", xrefOffset)
}
