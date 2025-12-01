// pdfsig - PDF digital signature tool
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	// Define flags
	nocert := flag.Bool("nocert", false, "don't verify certificates")
	noCRL := flag.Bool("no-crl", false, "don't check CRL")
	noOCSP := flag.Bool("no-ocsp", false, "don't check OCSP")
	dump := flag.Bool("dump", false, "dump all signatures")
	ownerPwd := flag.String("opw", "", "owner password")
	userPwd := flag.String("upw", "", "user password")
	version := flag.Bool("v", false, "print version info")
	help := flag.Bool("h", false, "print usage information")
	flag.BoolVar(help, "help", false, "print usage information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pdfsig version 1.0.0\n")
		fmt.Fprintf(os.Stderr, "Copyright 2024 go-poppler authors\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pdfsig [options] <PDF-file>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("pdfsig version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		return
	}

	if *help || flag.NArg() < 1 {
		flag.Usage()
		return
	}

	pdfFile := flag.Arg(0)

	// Open PDF
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// Get signatures
	sigs := pdf.GetSignatures(doc)

	if len(sigs) == 0 {
		fmt.Println("File is not signed.")
		return
	}

	fmt.Printf("Digital Signatures: %d\n", len(sigs))
	for i, sig := range sigs {
		fmt.Printf("\nSignature #%d:\n", i+1)
		fmt.Printf("  - Signer: %s\n", sig.Signer)
		fmt.Printf("  - Signing Time: %s\n", sig.SigningTime)
		fmt.Printf("  - Reason: %s\n", sig.Reason)
		fmt.Printf("  - Location: %s\n", sig.Location)
		fmt.Printf("  - Contact Info: %s\n", sig.ContactInfo)
		if *dump {
			fmt.Printf("  - Filter: %s\n", sig.Filter)
			fmt.Printf("  - SubFilter: %s\n", sig.SubFilter)
		}
	}

	// Suppress unused variable warnings
	_ = nocert
	_ = noCRL
	_ = noOCSP
	_ = ownerPwd
	_ = userPwd
}
