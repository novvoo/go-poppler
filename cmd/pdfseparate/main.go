// pdfseparate - PDF page separator
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	firstPage = flag.Int("f", 1, "first page to extract")
	lastPage  = flag.Int("l", 0, "last page to extract")
	printHelp = flag.Bool("h", false, "print usage information")
	printVer  = flag.Bool("v", false, "print version information")
)

func usage() {
	fmt.Fprintf(os.Stderr, "pdfseparate version 0.1.0\n")
	fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n")
	fmt.Fprintf(os.Stderr, "Usage: pdfseparate [options] <PDF-file> <PDF-file-pattern>\n")
	fmt.Fprintf(os.Stderr, "\nThe PDF-file-pattern should contain %%d (or %%nd) for page number\n")
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *printHelp {
		usage()
		os.Exit(0)
	}

	if *printVer {
		fmt.Println("pdfseparate version 0.1.0")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}

	pdfFile := args[0]
	pattern := args[1]

	// Validate pattern contains %d
	if !strings.Contains(pattern, "%") {
		fmt.Fprintf(os.Stderr, "Error: PDF-file-pattern must contain %%d for page number\n")
		os.Exit(1)
	}

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Determine page range
	first := *firstPage
	last := *lastPage
	if last == 0 || last > doc.NumPages() {
		last = doc.NumPages()
	}
	if first < 1 {
		first = 1
	}

	// Extract pages
	for pageNum := first; pageNum <= last; pageNum++ {
		outputFile := fmt.Sprintf(pattern, pageNum)

		err := pdf.ExtractPage(doc, pageNum, outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting page %d: %v\n", pageNum, err)
			continue
		}
	}

	fmt.Printf("Extracted %d pages\n", last-first+1)
}
