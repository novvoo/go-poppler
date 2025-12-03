package main

import (
"flag"
"fmt"
"os"

"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
firstPage := flag.Int("f", 1, "first page to convert")
lastPage := flag.Int("l", 0, "last page to convert")
includeImages := flag.Bool("images", false, "include image references")
extractImages := flag.Bool("extract-images", false, "extract images to files")
imagePrefix := flag.String("image-prefix", "image", "prefix for image files")
pageSeparator := flag.String("separator", "\n\n---\n\n", "page separator")
headingDetection := flag.Bool("headings", true, "enable heading detection")
ownerPassword := flag.String("opw", "", "owner password")
userPassword := flag.String("upw", "", "user password")
help := flag.Bool("h", false, "display help")

flag.Usage = func() {
fmt.Fprintf(os.Stderr, "Usage: %s [options] <PDF-file> [<output-file>]\n\n", os.Args[0])
fmt.Fprintf(os.Stderr, "Convert PDF to Markdown format.\n\n")
fmt.Fprintf(os.Stderr, "Options:\n")
flag.PrintDefaults()
fmt.Fprintf(os.Stderr, "\nIf output-file is not specified, output goes to stdout.\n")
}

flag.Parse()

if *help {
flag.Usage()
os.Exit(0)
}

args := flag.Args()
if len(args) < 1 {
fmt.Fprintf(os.Stderr, "Error: PDF file not specified\n\n")
flag.Usage()
os.Exit(1)
}

pdfFile := args[0]
var outputFile string
if len(args) >= 2 {
outputFile = args[1]
}

doc, err := pdf.Open(pdfFile)
if err != nil {
fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
os.Exit(1)
}
defer doc.Close()

info := doc.GetInfo()
if info.Encrypted {
password := *userPassword
if password == "" {
password = *ownerPassword
}
if password != "" {
err = doc.Decrypt(password)
if err != nil {
fmt.Fprintf(os.Stderr, "Error: incorrect password\n")
os.Exit(1)
}
} else {
fmt.Fprintf(os.Stderr, "Error: PDF is encrypted\n")
os.Exit(1)
}
}

numPages := doc.NumPages()
if *firstPage < 1 {
*firstPage = 1
}
if *lastPage == 0 || *lastPage > numPages {
*lastPage = numPages
}
if *firstPage > *lastPage {
fmt.Fprintf(os.Stderr, "Error: invalid page range\n")
os.Exit(1)
}

options := pdf.MarkdownOptions{
FirstPage:        *firstPage,
LastPage:         *lastPage,
IncludeImages:    *includeImages,
ImagePrefix:      *imagePrefix,
ExtractImages:    *extractImages,
PageSeparator:    *pageSeparator,
HeadingDetection: *headingDetection,
}

writer := pdf.NewMarkdownWriter(doc, options)

var output *os.File
if outputFile != "" {
output, err = os.Create(outputFile)
if err != nil {
fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
os.Exit(1)
}
defer output.Close()
} else {
output = os.Stdout
}

err = writer.Write(output)
if err != nil {
fmt.Fprintf(os.Stderr, "Error converting: %v\n", err)
os.Exit(1)
}

if outputFile != "" {
fmt.Fprintf(os.Stderr, "Successfully converted %s to %s\n", pdfFile, outputFile)
}
}
