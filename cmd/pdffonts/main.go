// pdffonts - PDF font analyzer
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	firstPage = flag.Int("f", 1, "first page to scan")
	lastPage  = flag.Int("l", 0, "last page to scan")
	subst     = flag.Bool("subst", false, "show font substitutions")
	printHelp = flag.Bool("h", false, "print usage information")
	printVer  = flag.Bool("v", false, "print version information")
)

func usage() {
	fmt.Fprintf(os.Stderr, "pdffonts version 0.1.0\n")
	fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n")
	fmt.Fprintf(os.Stderr, "Usage: pdffonts [options] <PDF-file>\n")
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
		fmt.Println("pdffonts version 0.1.0")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	pdfFile := args[0]

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

	// Extract fonts
	fonts, err := pdf.ExtractFonts(doc, first, last)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting fonts: %v\n", err)
		os.Exit(1)
	}

	// Print header
	fmt.Printf("name                                 type              encoding         emb sub uni object ID\n")
	fmt.Printf("------------------------------------ ----------------- ---------------- --- --- --- ---------\n")

	// Print fonts
	for _, font := range fonts {
		emb := "no"
		if font.Embedded {
			emb = "yes"
		}
		sub := "no"
		if font.Subset {
			sub = "yes"
		}
		uni := "no"
		if font.Unicode {
			uni = "yes"
		}

		name := font.Name
		if len(name) > 36 {
			name = name[:36]
		}

		fontType := font.Type
		if len(fontType) > 17 {
			fontType = fontType[:17]
		}

		encoding := font.Encoding
		if len(encoding) > 16 {
			encoding = encoding[:16]
		}

		fmt.Printf("%-36s %-17s %-16s %-3s %-3s %-3s %5d %d\n",
			name, fontType, encoding, emb, sub, uni,
			font.ObjectNum, font.Generation)
	}
}
