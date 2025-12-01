// pdfdetach - 从 PDF 中分离/提取附件
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

var (
	listOnly   = flag.Bool("list", false, "list all embedded files")
	saveAll    = flag.Bool("saveall", false, "save all embedded files")
	saveName   = flag.String("save", "", "save specified embedded file")
	saveNum    = flag.Int("savefile", 0, "save embedded file by number")
	outputDir  = flag.String("o", ".", "output directory")
	ownerPwd   = flag.String("opw", "", "owner password")
	userPwd    = flag.String("upw", "", "user password")
	printHelp  = flag.Bool("h", false, "print usage information")
	printHelp2 = flag.Bool("help", false, "print usage information")
	version    = flag.Bool("v", false, "print version info")
)

func main() {
	flag.Parse()

	if *printHelp || *printHelp2 {
		printUsage()
		os.Exit(0)
	}

	if *version {
		fmt.Println("pdfdetach version 1.0.0")
		fmt.Println("Copyright 2024 go-poppler authors")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	inputFile := args[0]

	// 打开 PDF 文件
	doc, err := pdf.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PDF: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// 获取附件列表
	attachments, err := pdf.GetAttachments(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting attachments: %v\n", err)
		os.Exit(1)
	}

	if len(attachments) == 0 {
		fmt.Println("No embedded files in this document.")
		os.Exit(0)
	}

	// 列出附件
	if *listOnly || (!*saveAll && *saveName == "" && *saveNum == 0) {
		fmt.Printf("%d embedded files\n", len(attachments))
		for i, att := range attachments {
			fmt.Printf("%d: %s\n", i+1, att.Name)
			if att.Description != "" {
				fmt.Printf("   Description: %s\n", att.Description)
			}
			fmt.Printf("   Size: %d bytes\n", att.Size)
			if !att.CreationDate.IsZero() {
				fmt.Printf("   Created: %s\n", att.CreationDate.Format("2006-01-02 15:04:05"))
			}
			if !att.ModDate.IsZero() {
				fmt.Printf("   Modified: %s\n", att.ModDate.Format("2006-01-02 15:04:05"))
			}
			if att.MimeType != "" {
				fmt.Printf("   MIME Type: %s\n", att.MimeType)
			}
		}
		os.Exit(0)
	}

	// 创建输出目录
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// 保存所有附件
	if *saveAll {
		for _, att := range attachments {
			outputPath := filepath.Join(*outputDir, att.Name)
			if err := saveAttachment(att, outputPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", att.Name, err)
			} else {
				fmt.Printf("Saved: %s\n", outputPath)
			}
		}
		os.Exit(0)
	}

	// 按名称保存
	if *saveName != "" {
		for _, att := range attachments {
			if att.Name == *saveName {
				outputPath := filepath.Join(*outputDir, att.Name)
				if err := saveAttachment(att, outputPath); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", att.Name, err)
					os.Exit(1)
				}
				fmt.Printf("Saved: %s\n", outputPath)
				os.Exit(0)
			}
		}
		fmt.Fprintf(os.Stderr, "Attachment not found: %s\n", *saveName)
		os.Exit(1)
	}

	// 按编号保存
	if *saveNum > 0 {
		if *saveNum > len(attachments) {
			fmt.Fprintf(os.Stderr, "Invalid attachment number: %d (only %d attachments)\n", *saveNum, len(attachments))
			os.Exit(1)
		}
		att := attachments[*saveNum-1]
		outputPath := filepath.Join(*outputDir, att.Name)
		if err := saveAttachment(att, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", att.Name, err)
			os.Exit(1)
		}
		fmt.Printf("Saved: %s\n", outputPath)
		os.Exit(0)
	}
}

func saveAttachment(att *pdf.Attachment, outputPath string) error {
	return att.SaveTo(filepath.Dir(outputPath))
}

func printUsage() {
	fmt.Println("pdfdetach version 1.0.0")
	fmt.Println("Copyright 2024 go-poppler authors")
	fmt.Println()
	fmt.Println("Usage: pdfdetach [options] <PDF-file>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -list         list all embedded files")
	fmt.Println("  -saveall      save all embedded files")
	fmt.Println("  -save <name>  save specified embedded file by name")
	fmt.Println("  -savefile <n> save specified embedded file by number")
	fmt.Println("  -o <dir>      output directory (default: current)")
	fmt.Println("  -opw <pwd>    owner password")
	fmt.Println("  -upw <pwd>    user password")
	fmt.Println("  -h, -help     print usage information")
	fmt.Println("  -v            print version info")
}
