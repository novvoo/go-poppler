package main

import (
	"fmt"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	doc, _ := pdf.Open("test/test.pdf")
	defer doc.Close()

	extractor := pdf.NewTextExtractor(doc)
	text, _ := extractor.ExtractPage(1)

	fmt.Println(text[:500])
}
