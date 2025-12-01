// pdftoppm - PDF to PPM/PNG/TIFF image converter
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
	resolution := flag.Float64("r", 150, "resolution in DPI")
	scaleToX := flag.Int("scale-to-x", 0, "scale width to specified pixels")
	scaleToY := flag.Int("scale-to-y", 0, "scale height to specified pixels")
	scaleTo := flag.Int("scale-to", 0, "scale to specified pixels")
	mono := flag.Bool("mono", false, "generate monochrome image")
	gray := flag.Bool("gray", false, "generate grayscale image")
	png := flag.Bool("png", false, "generate PNG output")
	tiff := flag.Bool("tiff", false, "generate TIFF output")
	cropBox := flag.Bool("cropbox", false, "use crop box instead of media box")
	ownerPwd := flag.String("opw", "", "owner password")
	userPwd := flag.String("upw", "", "user password")
	quiet := flag.Bool("q", false, "don't print any messages")
	version := flag.Bool("v", false, "print version info")
	help := flag.Bool("h", false, "print usage information")
	flag.BoolVar(help, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdftoppm version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdftoppm [options] <PDF-file> [<output-root>]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("pdftoppm version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		return
	}

	if *help || flag.NArg() < 1 {
		flag.Usage()
		return
	}

	pdfFile := flag.Arg(0)
	outputRoot := flag.Arg(1)
	if outputRoot == "" {
		outputRoot = strings.TrimSuffix(filepath.Base(pdfFile), ".pdf")
	}

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Determine output format
	format := "ppm"
	ext := ".ppm"
	if *png {
		format = "png"
		ext = ".png"
	} else if *tiff {
		format = "tiff"
		ext = ".tif"
	}

	// Set up render options
	options := pdf.RenderOptions{
		DPI:       *resolution,
		Format:    format,
		FirstPage: *firstPage,
		LastPage:  *lastPage,
		Gray:      *gray,
		Mono:      *mono,
		CropBox:   *cropBox,
		ScaleTo:   *scaleTo,
		ScaleToX:  *scaleToX,
		ScaleToY:  *scaleToY,
		OwnerPwd:  *ownerPwd,
		UserPwd:   *userPwd,
	}

	renderer := pdf.NewPageRenderer(doc, options)

	// Determine page range
	first := *firstPage
	last := *lastPage
	if first < 1 {
		first = 1
	}
	if last == 0 || last > doc.NumPages() {
		last = doc.NumPages()
	}

	// Render pages
	for pageNum := first; pageNum <= last; pageNum++ {
		rendered, err := renderer.RenderPage(pageNum)
		if err != nil {
			if !*quiet {
				fmt.Fprintf(os.Stderr, "Error rendering page %d: %v\n", pageNum, err)
			}
			continue
		}

		// Generate output filename
		var outputFile string
		if last == first {
			outputFile = outputRoot + ext
		} else {
			outputFile = fmt.Sprintf("%s-%d%s", outputRoot, pageNum, ext)
		}

		// Write output
		err = os.WriteFile(outputFile, rendered.Data, 0644)
		if err != nil {
			if !*quiet {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputFile, err)
			}
			continue
		}

		if !*quiet {
			fmt.Printf("Wrote %s (%dx%d)\n", outputFile, rendered.Width, rendered.Height)
		}
	}

	_ = ownerPwd
	_ = userPwd
}
