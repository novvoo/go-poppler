package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	firstPage  int
	lastPage   int
	resolution int
	x          int
	y          int
	width      int
	height     int
	layout     bool
	fixed      float64
	raw        bool
	nodiag     bool
	htmlMeta   bool
	bbox       bool
	bboxLayout bool
	enc        string
	eol        string
	nopgbrk    bool
	ownerPw    string
	userPw     string
	quiet      bool
	printHelp  bool
	printVer   bool
)

func init() {
	flag.IntVar(&firstPage, "f", 1, "first page to convert")
	flag.IntVar(&lastPage, "l", 0, "last page to convert")
	flag.IntVar(&resolution, "r", 72, "resolution, in DPI (default 72)")
	flag.IntVar(&x, "x", 0, "x-coordinate of the crop area top left corner")
	flag.IntVar(&y, "y", 0, "y-coordinate of the crop area top left corner")
	flag.IntVar(&width, "W", 0, "width of crop area in pixels")
	flag.IntVar(&height, "H", 0, "height of crop area in pixels")
	flag.BoolVar(&layout, "layout", false, "maintain original physical layout")
	flag.Float64Var(&fixed, "fixed", 0, "character spacing (in points) for fixed-pitch")
	flag.BoolVar(&raw, "raw", false, "keep strings in content stream order")
	flag.BoolVar(&nodiag, "nodiag", false, "discard diagonal text")
	flag.BoolVar(&htmlMeta, "htmlmeta", false, "generate a simple HTML file")
	flag.BoolVar(&bbox, "bbox", false, "output bounding box for each word")
	flag.BoolVar(&bboxLayout, "bbox-layout", false, "output bounding box for each block")
	flag.StringVar(&enc, "enc", "UTF-8", "output text encoding name")
	flag.StringVar(&eol, "eol", "", "output end-of-line convention (unix/dos/mac)")
	flag.BoolVar(&nopgbrk, "nopgbrk", false, "don't insert page breaks between pages")
	flag.StringVar(&ownerPw, "opw", "", "owner password")
	flag.StringVar(&userPw, "upw", "", "user password")
	flag.BoolVar(&quiet, "q", false, "don't print any messages or errors")
	flag.BoolVar(&printHelp, "h", false, "print usage information")
	flag.BoolVar(&printHelp, "help", false, "print usage information")
	flag.BoolVar(&printVer, "v", false, "print copyright and version info")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdftotext version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdftotext [options] <PDF-file> [<text-file>]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -f <int>          : first page to convert\n")
		fmt.Fprintf(os.Stderr, "  -l <int>          : last page to convert\n")
		fmt.Fprintf(os.Stderr, "  -r <fp>           : resolution, in DPI (default 72)\n")
		fmt.Fprintf(os.Stderr, "  -x <int>          : x-coordinate of the crop area top left corner\n")
		fmt.Fprintf(os.Stderr, "  -y <int>          : y-coordinate of the crop area top left corner\n")
		fmt.Fprintf(os.Stderr, "  -W <int>          : width of crop area in pixels\n")
		fmt.Fprintf(os.Stderr, "  -H <int>          : height of crop area in pixels\n")
		fmt.Fprintf(os.Stderr, "  -layout           : maintain original physical layout\n")
		fmt.Fprintf(os.Stderr, "  -fixed <fp>       : character spacing (in points) for fixed-pitch\n")
		fmt.Fprintf(os.Stderr, "  -raw              : keep strings in content stream order\n")
		fmt.Fprintf(os.Stderr, "  -nodiag           : discard diagonal text\n")
		fmt.Fprintf(os.Stderr, "  -htmlmeta         : generate a simple HTML file\n")
		fmt.Fprintf(os.Stderr, "  -bbox             : output bounding box for each word\n")
		fmt.Fprintf(os.Stderr, "  -bbox-layout      : output bounding box for each block\n")
		fmt.Fprintf(os.Stderr, "  -enc <string>     : output text encoding name\n")
		fmt.Fprintf(os.Stderr, "  -eol <string>     : output end-of-line convention (unix/dos/mac)\n")
		fmt.Fprintf(os.Stderr, "  -nopgbrk          : don't insert page breaks between pages\n")
		fmt.Fprintf(os.Stderr, "  -opw <string>     : owner password\n")
		fmt.Fprintf(os.Stderr, "  -upw <string>     : user password\n")
		fmt.Fprintf(os.Stderr, "  -q                : don't print any messages or errors\n")
		fmt.Fprintf(os.Stderr, "  -v                : print copyright and version info\n")
		fmt.Fprintf(os.Stderr, "  -h                : print usage information\n")
		fmt.Fprintf(os.Stderr, "  -help             : print usage information\n")
	}
}

func main() {
	flag.Parse()

	if printVer {
		fmt.Println("pdftotext version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		fmt.Println("License: Apache-2.0")
		os.Exit(0)
	}

	if printHelp {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := args[0]
	outputFile := ""
	if len(args) >= 2 {
		outputFile = args[1]
	} else {
		// Default: replace .pdf with .txt or use stdout
		if strings.HasSuffix(strings.ToLower(inputFile), ".pdf") {
			outputFile = inputFile[:len(inputFile)-4] + ".txt"
		} else {
			outputFile = inputFile + ".txt"
		}
	}

	// Open PDF
	doc, err := pdf.Open(inputFile)
	if err != nil {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Error: Couldn't open file '%s': %v\n", inputFile, err)
		}
		os.Exit(1)
	}
	defer doc.Close()

	// Determine page range
	numPages := doc.NumPages()
	if lastPage == 0 || lastPage > numPages {
		lastPage = numPages
	}
	if firstPage < 1 {
		firstPage = 1
	}
	if firstPage > lastPage {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Error: Invalid page range\n")
		}
		os.Exit(1)
	}

	// Determine line ending
	lineEnding := "\n"
	switch eol {
	case "dos":
		lineEnding = "\r\n"
	case "mac":
		lineEnding = "\r"
	case "unix":
		lineEnding = "\n"
	}

	// Extract text
	var output *os.File
	if outputFile == "-" {
		output = os.Stdout
	} else {
		output, err = os.Create(outputFile)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Error: Couldn't create output file '%s': %v\n", outputFile, err)
			}
			os.Exit(1)
		}
		defer output.Close()
	}

	// HTML header
	if htmlMeta {
		info := doc.GetInfo()
		fmt.Fprintln(output, "<!DOCTYPE html>")
		fmt.Fprintln(output, "<html>")
		fmt.Fprintln(output, "<head>")
		if info.Title != "" {
			fmt.Fprintf(output, "<title>%s</title>\n", escapeHTML(info.Title))
		}
		if info.Author != "" {
			fmt.Fprintf(output, "<meta name=\"Author\" content=\"%s\">\n", escapeHTML(info.Author))
		}
		if info.Subject != "" {
			fmt.Fprintf(output, "<meta name=\"Subject\" content=\"%s\">\n", escapeHTML(info.Subject))
		}
		if info.Keywords != "" {
			fmt.Fprintf(output, "<meta name=\"Keywords\" content=\"%s\">\n", escapeHTML(info.Keywords))
		}
		fmt.Fprintln(output, "</head>")
		fmt.Fprintln(output, "<body>")
		fmt.Fprintln(output, "<pre>")
	}

	// Extract text from each page
	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		page, err := doc.GetPage(pageNum)
		if err != nil {
			continue
		}

		opts := pdf.TextExtractionOptions{
			Layout:     layout,
			Raw:        raw,
			NoDiagonal: nodiag,
		}

		text, err := pdf.ExtractTextFromPage(page, opts)
		if err != nil {
			continue
		}

		// Apply line ending
		if lineEnding != "\n" {
			text = strings.ReplaceAll(text, "\n", lineEnding)
		}

		if bbox || bboxLayout {
			// Output with bounding boxes (simplified)
			fmt.Fprintf(output, "<page number=\"%d\">\n", pageNum)
			fmt.Fprintf(output, "%s\n", escapeHTML(text))
			fmt.Fprintf(output, "</page>\n")
		} else {
			fmt.Fprint(output, text)
		}

		// Page break
		if !nopgbrk && pageNum < lastPage {
			fmt.Fprint(output, "\f")
		}
	}

	// HTML footer
	if htmlMeta {
		fmt.Fprintln(output, "</pre>")
		fmt.Fprintln(output, "</body>")
		fmt.Fprintln(output, "</html>")
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
