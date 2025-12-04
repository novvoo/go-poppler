package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	firstPage    int
	lastPage     int
	box          bool
	meta         bool
	js           bool
	rawDates     bool
	dests        bool
	enc          string
	listEnc      bool
	ownerPw      string
	userPw       string
	printVersion bool
	printHelp    bool
)

func init() {
	flag.IntVar(&firstPage, "f", 1, "first page to examine")
	flag.IntVar(&lastPage, "l", 0, "last page to examine")
	flag.BoolVar(&box, "box", false, "print the page bounding boxes")
	flag.BoolVar(&meta, "meta", false, "print the document metadata (XML)")
	flag.BoolVar(&js, "js", false, "print all JavaScript in the PDF")
	flag.BoolVar(&rawDates, "rawdates", false, "print the raw (undecoded) date strings")
	flag.BoolVar(&dests, "dests", false, "print all named destinations in the PDF")
	flag.StringVar(&enc, "enc", "UTF-8", "output text encoding name")
	flag.BoolVar(&listEnc, "listenc", false, "list available encodings")
	flag.StringVar(&ownerPw, "opw", "", "owner password")
	flag.StringVar(&userPw, "upw", "", "user password")
	flag.BoolVar(&printVersion, "v", false, "print copyright and version info")
	flag.BoolVar(&printHelp, "h", false, "print usage information")
	flag.BoolVar(&printHelp, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdfinfo version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdfinfo [options] <PDF-file>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -f <int>          : first page to examine\n")
		fmt.Fprintf(os.Stderr, "  -l <int>          : last page to examine\n")
		fmt.Fprintf(os.Stderr, "  -box              : print the page bounding boxes\n")
		fmt.Fprintf(os.Stderr, "  -meta             : print the document metadata (XML)\n")
		fmt.Fprintf(os.Stderr, "  -js               : print all JavaScript in the PDF\n")
		fmt.Fprintf(os.Stderr, "  -rawdates         : print the raw (undecoded) date strings\n")
		fmt.Fprintf(os.Stderr, "  -dests            : print all named destinations in the PDF\n")
		fmt.Fprintf(os.Stderr, "  -enc <string>     : output text encoding name\n")
		fmt.Fprintf(os.Stderr, "  -listenc          : list available encodings\n")
		fmt.Fprintf(os.Stderr, "  -opw <string>     : owner password\n")
		fmt.Fprintf(os.Stderr, "  -upw <string>     : user password\n")
		fmt.Fprintf(os.Stderr, "  -v                : print copyright and version info\n")
		fmt.Fprintf(os.Stderr, "  -h                : print usage information\n")
		fmt.Fprintf(os.Stderr, "  -help             : print usage information\n")
	}
}

func main() {
	flag.Parse()

	if printVersion {
		fmt.Println("pdfinfo version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		fmt.Println("License: Apache-2.0")
		os.Exit(0)
	}

	if printHelp {
		flag.Usage()
		os.Exit(0)
	}

	if listEnc {
		fmt.Println("Available encodings:")
		fmt.Println("  UTF-8")
		fmt.Println("  Latin1")
		fmt.Println("  ASCII")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := args[0]

	// Open PDF
	doc, err := pdf.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Couldn't open file '%s': %v\n", inputFile, err)
		os.Exit(1)
	}
	defer doc.Close()

	info := doc.GetInfo()
	numPages := doc.NumPages()

	// Print basic info
	fmt.Printf("Title:          %s\n", info.Title)
	fmt.Printf("Subject:        %s\n", info.Subject)
	fmt.Printf("Keywords:       %s\n", info.Keywords)
	fmt.Printf("Author:         %s\n", info.Author)
	fmt.Printf("Creator:        %s\n", info.Creator)
	fmt.Printf("Producer:       %s\n", info.Producer)

	// Print dates
	if rawDates {
		fmt.Printf("CreationDate:   %s\n", info.CreationDateRaw)
		fmt.Printf("ModDate:        %s\n", info.ModDateRaw)
	} else {
		if !info.CreationDate.IsZero() {
			fmt.Printf("CreationDate:   %s\n", formatDate(info.CreationDate))
		}
		if !info.ModDate.IsZero() {
			fmt.Printf("ModDate:        %s\n", formatDate(info.ModDate))
		}
	}

	// Custom metadata
	if len(info.Custom) > 0 {
		fmt.Println("Custom Metadata:")
		for k, v := range info.Custom {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	// Tagged
	fmt.Printf("Tagged:         %s\n", boolToYesNo(info.Tagged))

	// User properties
	fmt.Printf("UserProperties: %s\n", boolToYesNo(info.UserProperties))

	// Suspects
	fmt.Printf("Suspects:       %s\n", boolToYesNo(info.Suspects))

	// Form
	fmt.Printf("Form:           %s\n", info.Form)

	// JavaScript
	fmt.Printf("JavaScript:     %s\n", boolToYesNo(info.JavaScript))

	// Pages
	fmt.Printf("Pages:          %d\n", numPages)

	// Encrypted
	fmt.Printf("Encrypted:      %s\n", boolToYesNo(info.Encrypted))

	// Page size
	if numPages > 0 {
		page, err := doc.GetPage(1)
		if err == nil {
			mediaBox := page.GetMediaBox()
			fmt.Printf("Page size:      %.2f x %.2f pts", mediaBox.Width(), mediaBox.Height())
			// Convert to common paper sizes
			paperSize := detectPaperSize(mediaBox.Width(), mediaBox.Height())
			if paperSize != "" {
				fmt.Printf(" (%s)", paperSize)
			}
			fmt.Println()

			if box {
				printPageBoxes(page)
			}
		}
	}

	// Page rotation
	fmt.Printf("Page rot:       0\n")

	// File size
	fileInfo, err := os.Stat(inputFile)
	if err == nil {
		fmt.Printf("File size:      %d bytes\n", fileInfo.Size())
	}

	// Optimized
	fmt.Printf("Optimized:      %s\n", boolToYesNo(info.Optimized))

	// PDF version
	fmt.Printf("PDF version:    %s\n", info.PDFVersion)

	// Print metadata XML
	if meta {
		metadata := doc.GetMetadata()
		if metadata != "" {
			fmt.Println("\nMetadata:")
			fmt.Println(metadata)
		}
	}

	// Print JavaScript
	if js {
		scripts := doc.GetJavaScript()
		if len(scripts) > 0 {
			fmt.Println("\nJavaScript:")
			for i, script := range scripts {
				fmt.Printf("--- Script %d ---\n", i+1)
				fmt.Println(script)
			}
		}
	}

	// Print named destinations
	if dests {
		destinations := doc.GetNamedDestinations()
		if len(destinations) > 0 {
			fmt.Println("\nNamed Destinations:")
			for name, dest := range destinations {
				fmt.Printf("  %s -> %v\n", name, dest)
			}
		}
	}

	// Print layer information (OCG)
	layerMgr := pdf.NewLayerManager(doc)
	layers := layerMgr.GetLayers()
	if len(layers) > 0 {
		fmt.Printf("\nOptional Content Groups (Layers): %d\n", len(layers))
		for i, layer := range layers {
			visible := "visible"
			if !layer.Visible {
				visible = "hidden"
			}
			locked := ""
			if layer.Locked {
				locked = " [locked]"
			}
			fmt.Printf("  %d. %s (%s)%s\n", i+1, layer.Name, visible, locked)
			if layer.Intent != "" {
				fmt.Printf("     Intent: %s\n", layer.Intent)
			}
		}
	}
}

func formatDate(t time.Time) string {
	return t.Format("Mon Jan 2 15:04:05 2006 MST")
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func detectPaperSize(width, height float64) string {
	// Common paper sizes in points
	sizes := map[string][2]float64{
		"letter":    {612, 792},
		"legal":     {612, 1008},
		"A4":        {595.276, 841.89},
		"A3":        {841.89, 1190.55},
		"A5":        {419.528, 595.276},
		"B5":        {498.898, 708.661},
		"executive": {522, 756},
		"tabloid":   {792, 1224},
	}

	tolerance := 5.0

	for name, size := range sizes {
		// Check both orientations
		if (abs(width-size[0]) < tolerance && abs(height-size[1]) < tolerance) ||
			(abs(width-size[1]) < tolerance && abs(height-size[0]) < tolerance) {
			orientation := "portrait"
			if width > height {
				orientation = "landscape"
			}
			return fmt.Sprintf("%s, %s", name, orientation)
		}
	}

	return ""
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func printPageBoxes(page *pdf.Page) {
	mediaBox := page.GetMediaBox()
	cropBox := page.GetCropBox()
	bleedBox := page.GetBleedBox()
	trimBox := page.GetTrimBox()
	artBox := page.GetArtBox()

	fmt.Printf("MediaBox:       %.2f %.2f %.2f %.2f\n",
		mediaBox.LLX, mediaBox.LLY, mediaBox.URX, mediaBox.URY)
	fmt.Printf("CropBox:        %.2f %.2f %.2f %.2f\n",
		cropBox.LLX, cropBox.LLY, cropBox.URX, cropBox.URY)
	fmt.Printf("BleedBox:       %.2f %.2f %.2f %.2f\n",
		bleedBox.LLX, bleedBox.LLY, bleedBox.URX, bleedBox.URY)
	fmt.Printf("TrimBox:        %.2f %.2f %.2f %.2f\n",
		trimBox.LLX, trimBox.LLY, trimBox.URX, trimBox.URY)
	fmt.Printf("ArtBox:         %.2f %.2f %.2f %.2f\n",
		artBox.LLX, artBox.LLY, artBox.URX, artBox.URY)
}

// Unused but kept for compatibility
var _ = strings.TrimSpace
