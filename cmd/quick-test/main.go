package main

import (
	"fmt"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	doc, err := pdf.Open("test/test.pdf")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer doc.Close()

	// 使用高级文本提取 API
	page, err := doc.GetPage(1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 测试不同的提取模式
	fmt.Println("=== 标准模式 ===")
	opts := pdf.TextExtractionOptions{Layout: false, Raw: false}
	text, _ := pdf.ExtractTextFromPage(page, opts)
	if len(text) > 500 {
		fmt.Println(text[:500])
	} else {
		fmt.Println(text)
	}

	fmt.Println("\n=== 布局模式 ===")
	layoutText, _ := pdf.ExtractPageTextWithLayout(page)
	if len(layoutText) > 500 {
		fmt.Println(layoutText[:500])
	} else {
		fmt.Println(layoutText)
	}
}
