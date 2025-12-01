package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	var (
		listOnly  bool
		saveAll   bool
		saveFile  int
		outputDir string
		ownerPw   string
		userPw    string
		printHelp bool
		printVer  bool
	)

	flag.BoolVar(&listOnly, "list", false, "list all embedded files")
	flag.BoolVar(&saveAll, "saveall", false, "save all embedded files")
	flag.IntVar(&saveFile, "save", 0, "save the specified embedded file (1-based index)")
	flag.StringVar(&outputDir, "o", ".", "output directory")
	flag.StringVar(&ownerPw, "opw", "", "owner password")
	flag.StringVar(&userPw, "upw", "", "user password")
	flag.BoolVar(&printHelp, "h", false, "print usage information")
	flag.BoolVar(&printHelp, "help", false, "print usage information")
	flag.BoolVar(&printVer, "v", false, "print version info")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdfdetach version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <PDF-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if printVer {
		fmt.Println("pdfdetach version 1.0.0")
		os.Exit(0)
	}

	if printHelp {
		flag.Usage()
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	doc, err := pdf.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Couldn't open file '%s': %v\n", inputFile, err)
		os.Exit(1)
	}
	defer doc.Close()

	// Get embedded files
	attachments, err := pdf.GetAttachments(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting attachments: %v\n", err)
		os.Exit(1)
	}

	if len(attachments) == 0 {
		fmt.Println("No embedded files found.")
		os.Exit(0)
	}

	if listOnly {
		fmt.Printf("%d embedded files:\n", len(attachments))
		for i, att := range attachments {
			fmt.Printf("%d: %s", i+1, att.Name)
			if att.Size > 0 {
				fmt.Printf(", %d bytes", att.Size)
			}
			if att.Description != "" {
				fmt.Printf(", %s", att.Description)
			}
			fmt.Println()
		}
		os.Exit(0)
	}

	if saveFile > 0 {
		if saveFile > len(attachments) {
			fmt.Fprintf(os.Stderr, "Error: Invalid file index %d (only %d files)\n", saveFile, len(attachments))
			os.Exit(1)
		}
		att := attachments[saveFile-1]
		err := att.SaveTo(outputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Saved: %s\n", att.Name)
		os.Exit(0)
	}

	if saveAll {
		for _, att := range attachments {
			err := att.SaveTo(outputDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", att.Name, err)
				continue
			}
			fmt.Printf("Saved: %s\n", att.Name)
		}
		os.Exit(0)
	}

	// Default: list files
	fmt.Printf("%d embedded files:\n", len(attachments))
	for i, att := range attachments {
		fmt.Printf("%d: %s\n", i+1, att.Name)
	}

	// Suppress unused
	_ = ownerPw
	_ = userPw
}
