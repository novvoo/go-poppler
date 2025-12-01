// pdftohtml - PDF to HTML converter
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	// Define flags
	firstPage := flag.Int("f", 1, "first page to convert")
	lastPage := flag.Int("l", 0, "last page to convert")
	exchangeLinks := flag.Bool("p", false, "exchange .pdf links by .html")
	complexMode := flag.Bool("c", false, "generate complex output")
	singlePage := flag.Bool("s", false, "generate single HTML page")
	ignoreImages := flag.Bool("i", false, "ignore images")
	noFrames := flag.Bool("noframes", false, "generate no frames")
	stdout := flag.Bool("stdout", false, "use standard output")
	zoom := flag.Float64("zoom", 1.5, "zoom factor")
	xml := flag.Bool("xml", false, "output for XML post-processing")
	noRoundedCoords := flag.Bool("noroundcoord", false, "don't round coordinates")
	hidden := flag.Bool("hidden", false, "output hidden text")
	noDrm := flag.Bool("nodrm", false, "override document DRM settings")
	wordBreak := flag.Bool("wbt", false, "word break threshold")
	fontFullName := flag.Bool("fontfullname", false, "output font full name")
	ownerPwd := flag.String("opw", "", "owner password")
	userPwd := flag.String("upw", "", "user password")
	quiet := flag.Bool("q", false, "don't print any messages")
	version := flag.Bool("v", false, "print version info")
	help := flag.Bool("h", false, "print usage information")
	flag.BoolVar(help, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdftohtml version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdftohtml [options] <PDF-file> [<HTML-file>]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("pdftohtml version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		return
	}

	if *help || flag.NArg() < 1 {
		flag.Usage()
		return
	}

	pdfFile := flag.Arg(0)
	htmlFile := flag.Arg(1)
	if htmlFile == "" {
		htmlFile = strings.TrimSuffix(filepath.Base(pdfFile), ".pdf")
	}

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Determine output
	var output *os.File
	if *stdout {
		output = os.Stdout
	} else {
		var outPath string
		if *xml {
			outPath = htmlFile + ".xml"
		} else {
			outPath = htmlFile + ".html"
		}
		output, err = os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer output.Close()
	}

	// Create HTML writer
	htmlWriter := pdf.NewHTMLWriter(doc, pdf.HTMLOptions{
		FirstPage:    *firstPage,
		LastPage:     *lastPage,
		Complex:      *complexMode,
		SinglePage:   *singlePage,
		IgnoreImages: *ignoreImages,
		NoFrames:     *noFrames,
		Zoom:         *zoom,
		XML:          *xml,
	})

	err = htmlWriter.Write(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing HTML: %v\n", err)
		os.Exit(1)
	}

	if !*quiet && !*stdout {
		if *xml {
			fmt.Printf("Wrote %s.xml\n", htmlFile)
		} else {
			fmt.Printf("Wrote %s.html\n", htmlFile)
		}
	}

	// Suppress unused variable warnings
	_ = exchangeLinks
	_ = noRoundedCoords
	_ = hidden
	_ = noDrm
	_ = wordBreak
	_ = fontFullName
	_ = ownerPwd
	_ = userPwd
}
