// pdftops - PDF to PostScript converter
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
	level1 := flag.Bool("level1", false, "generate Level 1 PostScript")
	level1Sep := flag.Bool("level1sep", false, "generate Level 1 separable PostScript")
	level2 := flag.Bool("level2", false, "generate Level 2 PostScript")
	level2Sep := flag.Bool("level2sep", false, "generate Level 2 separable PostScript")
	level3 := flag.Bool("level3", false, "generate Level 3 PostScript")
	level3Sep := flag.Bool("level3sep", false, "generate Level 3 separable PostScript")
	eps := flag.Bool("eps", false, "generate EPS")
	form := flag.Bool("form", false, "generate a PostScript form")
	opi := flag.Bool("opi", false, "generate OPI comments")
	noembt1 := flag.Bool("noembt1", false, "don't embed Type 1 fonts")
	noembtt := flag.Bool("noembtt", false, "don't embed TrueType fonts")
	noembcidps := flag.Bool("noembcidps", false, "don't embed CID PostScript fonts")
	noembcidtt := flag.Bool("noembcidtt", false, "don't embed CID TrueType fonts")
	passfonts := flag.Bool("passfonts", false, "don't substitute missing fonts")
	preload := flag.Bool("preload", false, "preload images and forms")
	paperWidth := flag.Float64("paperw", 0, "paper width in points")
	paperHeight := flag.Float64("paperh", 0, "paper height in points")
	paper := flag.String("paper", "", "paper size (letter, legal, A4, etc.)")
	nocrop := flag.Bool("nocrop", false, "don't crop pages to CropBox")
	expand := flag.Bool("expand", false, "expand pages smaller than paper")
	noshrink := flag.Bool("noshrink", false, "don't shrink pages larger than paper")
	nocenter := flag.Bool("nocenter", false, "don't center pages smaller than paper")
	duplex := flag.Bool("duplex", false, "enable duplex printing")
	ownerPwd := flag.String("opw", "", "owner password")
	userPwd := flag.String("upw", "", "user password")
	quiet := flag.Bool("q", false, "don't print any messages")
	version := flag.Bool("v", false, "print version info")
	help := flag.Bool("h", false, "print usage information")
	flag.BoolVar(help, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdftops version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdftops [options] <PDF-file> [<PS-file>]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("pdftops version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		return
	}

	if *help || flag.NArg() < 1 {
		flag.Usage()
		return
	}

	pdfFile := flag.Arg(0)
	psFile := flag.Arg(1)
	if psFile == "" {
		psFile = strings.TrimSuffix(filepath.Base(pdfFile), ".pdf") + ".ps"
	}

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Determine PS level
	level := 2
	if *level1 || *level1Sep {
		level = 1
	} else if *level3 || *level3Sep {
		level = 3
	}

	// Determine paper size
	pWidth := *paperWidth
	pHeight := *paperHeight
	if *paper != "" {
		switch strings.ToLower(*paper) {
		case "letter":
			pWidth, pHeight = 612, 792
		case "legal":
			pWidth, pHeight = 612, 1008
		case "a4":
			pWidth, pHeight = 595, 842
		case "a3":
			pWidth, pHeight = 842, 1191
		}
	}
	if pWidth == 0 {
		pWidth = 612
	}
	if pHeight == 0 {
		pHeight = 792
	}

	// Create output file
	output, err := os.Create(psFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer output.Close()

	// Create PostScript writer
	psWriter := pdf.NewPostScriptWriter(doc, pdf.PSOptions{
		FirstPage:   *firstPage,
		LastPage:    *lastPage,
		Level:       level,
		EPS:         *eps,
		Duplex:      *duplex,
		PaperWidth:  pWidth,
		PaperHeight: pHeight,
	})

	err = psWriter.Write(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PostScript: %v\n", err)
		os.Exit(1)
	}

	if !*quiet {
		fmt.Printf("Wrote %s\n", psFile)
	}

	// Suppress unused variable warnings
	_ = level1Sep
	_ = level2
	_ = level2Sep
	_ = level3Sep
	_ = form
	_ = opi
	_ = noembt1
	_ = noembtt
	_ = noembcidps
	_ = noembcidtt
	_ = passfonts
	_ = preload
	_ = nocrop
	_ = expand
	_ = noshrink
	_ = nocenter
	_ = ownerPwd
	_ = userPwd
}
