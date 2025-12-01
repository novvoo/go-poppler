// pdfunite - PDF merger
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	printHelp = flag.Bool("h", false, "print usage information")
	printVer  = flag.Bool("v", false, "print version information")
)

func usage() {
	fmt.Fprintf(os.Stderr, "pdfunite version 0.1.0\n")
	fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n")
	fmt.Fprintf(os.Stderr, "Usage: pdfunite [options] <PDF-file-1> ... <PDF-file-n> <output-PDF-file>\n")
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
		fmt.Println("pdfunite version 0.1.0")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}

	// Last argument is output file
	outputFile := args[len(args)-1]
	inputFiles := args[:len(args)-1]

	// Open all input PDFs
	var docs []*pdf.Document
	for _, inputFile := range inputFiles {
		doc, err := pdf.Open(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", inputFile, err)
			os.Exit(1)
		}
		docs = append(docs, doc)
		defer doc.Close()
	}

	// Merge documents
	err := pdf.MergeDocuments(docs, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error merging PDFs: %v\n", err)
		os.Exit(1)
	}

	// Count total pages
	totalPages := 0
	for _, doc := range docs {
		totalPages += doc.NumPages()
	}

	fmt.Printf("Merged %d files (%d pages) into %s\n", len(inputFiles), totalPages, outputFile)
}
