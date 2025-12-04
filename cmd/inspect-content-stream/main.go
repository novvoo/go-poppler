package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: inspect-content-stream <pdf-file> [page-num]")
		os.Exit(1)
	}

	pdfFile := os.Args[1]
	pageNum := 1
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &pageNum)
	}

	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	page, err := doc.GetPage(pageNum)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	contents, err := page.GetContents()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== 内容流前500字节 ===\n")
	if len(contents) > 500 {
		fmt.Printf("%s\n...\n", string(contents[:500]))
	} else {
		fmt.Printf("%s\n", string(contents))
	}

	// 查找cm操作（CTM变换）
	fmt.Printf("\n=== 查找CTM变换 (cm操作) ===\n")
	lines := strings.Split(string(contents), "\n")
	count := 0
	for i, line := range lines {
		if strings.Contains(line, " cm") {
			fmt.Printf("行 %d: %s\n", i+1, line)
			count++
			if count >= 10 {
				break
			}
		}
	}

	// 查找第一个Tm操作
	fmt.Printf("\n=== 查找前10个Tm操作 ===\n")
	count = 0
	for i, line := range lines {
		if strings.Contains(line, " Tm") {
			fmt.Printf("行 %d: %s\n", i+1, line)
			count++
			if count >= 10 {
				break
			}
		}
	}
}
