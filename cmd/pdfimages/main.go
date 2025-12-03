// pdfimages - PDF image extractor
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	firstPage  = flag.Int("f", 1, "first page to scan")
	lastPage   = flag.Int("l", 0, "last page to scan")
	listImages = flag.Bool("list", false, "list images instead of extracting")
	allImages  = flag.Bool("all", false, "write all images (including inline)")
	pageImages = flag.Bool("page", false, "write page images")
	ccitt      = flag.Bool("ccitt", false, "write CCITT images as CCITT files")
	jbig2      = flag.Bool("jbig2", false, "write JBIG2 images as JBIG2 files")
	jpeg       = flag.Bool("j", false, "write JPEG images as JPEG files")
	jp2        = flag.Bool("jp2", false, "write JPEG2000 images as JP2 files")
	png        = flag.Bool("png", false, "write images as PNG files (default)")
	tiff       = flag.Bool("tiff", false, "write images as TIFF files")
	ppm        = flag.Bool("ppm", false, "write images as PPM files")
	upscale    = flag.Int("upscale", 1, "upscale factor for images")
	printHelp  = flag.Bool("h", false, "print usage information")
	printVer   = flag.Bool("v", false, "print version information")
)

func usage() {
	fmt.Fprintf(os.Stderr, "pdfimages version 0.1.0\n")
	fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n")
	fmt.Fprintf(os.Stderr, "Usage: pdfimages [options] <PDF-file> <image-root>\n")
	fmt.Fprintf(os.Stderr, "       pdfimages [options] -list <PDF-file>\n")
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
		fmt.Println("pdfimages version 0.1.0")
		os.Exit(0)
	}

	args := flag.Args()
	if *listImages {
		if len(args) < 1 {
			usage()
			os.Exit(1)
		}
	} else {
		if len(args) < 2 {
			usage()
			os.Exit(1)
		}
	}

	pdfFile := args[0]
	var imageRoot string
	if !*listImages {
		imageRoot = args[1]
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

	// Extract images
	extractor := pdf.NewImageExtractor(doc)
	images, err := extractor.ExtractImages(first, last)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting images: %v\n", err)
		os.Exit(1)
	}

	if *listImages {
		// List mode
		fmt.Printf("page   num  type   width height color comp bpc  enc interp  object ID x-ppi y-ppi size ratio\n")
		fmt.Printf("--------------------------------------------------------------------------------------------\n")
		for _, img := range images {
			colorSpace := img.ColorSpace
			if colorSpace == "" {
				colorSpace = "-"
			}
			enc := img.Filter
			if enc == "" {
				enc = "image"
			}
			interp := "no"
			if img.Interpolate {
				interp = "yes"
			}
			fmt.Printf("%4d %5d  %-6s %5d %5d  %-5s %4d %3d  %-5s %-6s %4d %4d %5d %5d %5dB %3d%%\n",
				img.Page, img.Index, img.Type,
				img.Width, img.Height,
				colorSpace, img.Components, img.BitsPerComponent,
				enc, interp,
				img.ObjectNum, img.Generation,
				img.XPPI, img.YPPI,
				img.Size, img.Ratio)
		}
	} else {
		// Extract mode - determine format
		format := "png"
		ext := "png"
		quality := 85

		if *ppm {
			format = "ppm"
			ext = "ppm"
		} else if *jpeg {
			format = "jpeg"
			ext = "jpg"
		} else if *tiff {
			format = "tiff"
			ext = "tif"
		} else if *jp2 {
			format = "native"
			ext = "jp2"
		} else if *ccitt || *jbig2 {
			format = "native"
		}

		for i, img := range images {
			// Determine extension based on format and native type
			imgExt := ext
			if format == "native" {
				imgExt = getNativeExtension(img)
			}
			filename := fmt.Sprintf("%s-%03d.%s", imageRoot, i, imgExt)

			// Create directory if needed
			dir := filepath.Dir(filename)
			if dir != "" && dir != "." {
				os.MkdirAll(dir, 0755)
			}

			data, err := extractor.GetImageDataWithFormat(img, format, quality)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not extract image %d: %v\n", i, err)
				continue
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
				continue
			}
		}
		fmt.Printf("Extracted %d images\n", len(images))
	}
}

func getNativeExtension(img *pdf.ImageInfo) string {
	switch strings.ToLower(img.Filter) {
	case "dctdecode":
		return "jpg"
	case "jpxdecode":
		return "jp2"
	case "ccittfaxdecode":
		return "tif"
	case "jbig2decode":
		return "jb2"
	}
	return "png"
}
