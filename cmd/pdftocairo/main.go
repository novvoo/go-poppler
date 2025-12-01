// pdftocairo - PDF to PNG/JPEG/TIFF/PS/EPS/SVG converter
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
	pngOut := flag.Bool("png", false, "generate PNG output")
	tiffOut := flag.Bool("tiff", false, "generate TIFF output")
	psOut := flag.Bool("ps", false, "generate PostScript output")
	epsOut := flag.Bool("eps", false, "generate EPS output")
	svgOut := flag.Bool("svg", false, "generate SVG output")
	pdfOut := flag.Bool("pdf", false, "generate PDF output")
	paperWidth := flag.Float64("paper-width", 0, "paper width in points")
	paperHeight := flag.Float64("paper-height", 0, "paper height in points")
	cropBox := flag.Bool("cropbox", false, "use crop box instead of media box")
	ownerPwd := flag.String("opw", "", "owner password")
	userPwd := flag.String("upw", "", "user password")
	quiet := flag.Bool("q", false, "don't print any messages")
	version := flag.Bool("v", false, "print version info")
	help := flag.Bool("h", false, "print usage information")
	flag.BoolVar(help, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdftocairo version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdftocairo [options] <PDF-file> [<output-file>]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("pdftocairo version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		return
	}

	if *help || flag.NArg() < 1 {
		flag.Usage()
		return
	}

	pdfFile := flag.Arg(0)
	outputFile := flag.Arg(1)
	if outputFile == "" {
		outputFile = strings.TrimSuffix(filepath.Base(pdfFile), ".pdf")
	}

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Determine output format
	format := "png"
	ext := ".png"
	if *svgOut {
		format = "svg"
		ext = ".svg"
	} else if *psOut {
		format = "ps"
		ext = ".ps"
	} else if *epsOut {
		format = "eps"
		ext = ".eps"
	} else if *pdfOut {
		format = "pdf"
		ext = ".pdf"
	} else if *tiffOut {
		format = "tiff"
		ext = ".tif"
	} else if *pngOut {
		format = "png"
		ext = ".png"
	}

	// Handle vector formats (SVG, PS, EPS)
	if format == "svg" || format == "ps" || format == "eps" {
		cairoOpts := pdf.CairoOptions{
			FirstPage:   *firstPage,
			LastPage:    *lastPage,
			Format:      format,
			Resolution:  *resolution,
			PaperWidth:  *paperWidth,
			PaperHeight: *paperHeight,
		}

		renderer := pdf.NewCairoRenderer(doc, cairoOpts)

		outPath := outputFile + ext
		file, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		err = renderer.Render(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering: %v\n", err)
			os.Exit(1)
		}

		if !*quiet {
			fmt.Printf("Wrote %s\n", outPath)
		}
		return
	}

	// Handle raster formats (PNG, TIFF)
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

	first := *firstPage
	last := *lastPage
	if first < 1 {
		first = 1
	}
	if last == 0 || last > doc.NumPages() {
		last = doc.NumPages()
	}

	for pageNum := first; pageNum <= last; pageNum++ {
		rendered, err := renderer.RenderPage(pageNum)
		if err != nil {
			if !*quiet {
				fmt.Fprintf(os.Stderr, "Error rendering page %d: %v\n", pageNum, err)
			}
			continue
		}

		var outPath string
		if last == first {
			outPath = outputFile + ext
		} else {
			outPath = fmt.Sprintf("%s-%d%s", outputFile, pageNum, ext)
		}

		err = os.WriteFile(outPath, rendered.Data, 0644)
		if err != nil {
			if !*quiet {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outPath, err)
			}
			continue
		}

		if !*quiet {
			fmt.Printf("Wrote %s (%dx%d)\n", outPath, rendered.Width, rendered.Height)
		}
	}

	_ = ownerPwd
	_ = userPwd
}
