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

	// Create signature validator with advanced features
	validator := pdf.NewSignatureValidator(doc)

	// Verify all signatures
	results := validator.VerifyAllSignatures()

	if len(results) == 0 {
		fmt.Println("File is not signed.")
		return
	}

	fmt.Printf("Digital Signatures: %d\n", len(results))
	for i, result := range results {
		fmt.Printf("\nSignature #%d:\n", i+1)
		fmt.Printf("  - Signer: %s\n", result.SignerName)

		// Validation status
		if result.Valid {
			fmt.Printf("  - Status: Valid ✓\n")
		} else {
			fmt.Printf("  - Status: Invalid ✗\n")
		}

		// Signing time
		if !result.SigningTime.IsZero() {
			fmt.Printf("  - Signing Time: %s\n", result.SigningTime.Format("2006-01-02 15:04:05"))
		}

		// Additional info
		if result.Reason != "" {
			fmt.Printf("  - Reason: %s\n", result.Reason)
		}
		if result.Location != "" {
			fmt.Printf("  - Location: %s\n", result.Location)
		}

		// Signature type and algorithm
		if *dump {
			fmt.Printf("  - Signature Type: %s\n", result.SignatureType)
			fmt.Printf("  - Hash Algorithm: %s\n", result.HashAlgorithm)
			fmt.Printf("  - Coverage Status: %s\n", result.CoverageStatus)

			// Certificate info
			if result.Certificate != nil {
				fmt.Printf("  - Certificate Subject: %s\n", result.Certificate.Subject)
				fmt.Printf("  - Certificate Issuer: %s\n", result.Certificate.Issuer)
				fmt.Printf("  - Certificate Valid From: %s\n", result.Certificate.NotBefore.Format("2006-01-02"))
				fmt.Printf("  - Certificate Valid To: %s\n", result.Certificate.NotAfter.Format("2006-01-02"))
			}

			// Validation errors
			if len(result.ValidationErrors) > 0 {
				fmt.Printf("  - Validation Errors:\n")
				for _, err := range result.ValidationErrors {
					fmt.Printf("    * %s\n", err)
				}
			}
		}
	}

	// Check document integrity
	if !*nocert {
		fmt.Println("\n=== Document Integrity Check ===")
		intact, issues := validator.VerifyDocumentIntegrity()
		if intact {
			fmt.Println("Document integrity: OK ✓")
		} else {
			fmt.Println("Document integrity: Issues found ✗")
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
		}
	}

	// Suppress unused variable warnings
	_ = nocert
	_ = noCRL
	_ = noOCSP
	_ = ownerPwd
	_ = userPwd
}
