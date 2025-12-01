// pdfattach - PDF attachment tool
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	// Define flags
	replace := flag.Bool("replace", false, "replace existing attachment")
	ownerPwd := flag.String("opw", "", "owner password")
	userPwd := flag.String("upw", "", "user password")
	version := flag.Bool("v", false, "print version info")
	help := flag.Bool("h", false, "print usage information")
	flag.BoolVar(help, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdfattach version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdfattach [options] <PDF-file> <file-to-attach> <output-PDF>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("pdfattach version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		return
	}

	if *help || flag.NArg() < 3 {
		flag.Usage()
		return
	}

	pdfFile := flag.Arg(0)
	attachFile := flag.Arg(1)
	outputFile := flag.Arg(2)

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Read attachment file
	attachData, err := os.ReadFile(attachFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading attachment: %v\n", err)
		os.Exit(1)
	}

	// Add attachment
	err = pdf.AddAttachment(doc, attachFile, attachData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding attachment: %v\n", err)
		os.Exit(1)
	}

	// Write output
	err = pdf.WriteToFile(doc, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added attachment '%s' to '%s'\n", attachFile, outputFile)

	// Suppress unused variable warnings
	_ = replace
	_ = ownerPwd
	_ = userPwd
}
