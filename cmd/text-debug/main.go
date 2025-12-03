package main

import (
	"fmt"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	fmt.Println("调试程序启动: 提取 test/test.pdf 的文本")

	doc, err := pdf.Open("test/test.pdf")
	if err != nil {
		fmt.Printf("打开PDF失败: %v\n", err)
		return
	}
	defer doc.Close()

	fmt.Printf("PDF页面数: %d\n", doc.NumPages())

	extractor := pdf.NewTextExtractor(doc)
	// 设置选项用于调试
	extractor.Layout = true // 保持布局
	extractor.Raw = false   // 正常顺序
	extractor.Options = pdf.TextExtractionOptions{
		Layout:     true,
		Raw:        false,
		NoDiagonal: false,
		FirstPage:  1,
		LastPage:   1, // 只第一页，避免超时
	}

	text, err := extractor.ExtractText()
	if err != nil {
		fmt.Printf("提取文本失败: %v\n", err)
		return
	}

	fmt.Println("\n=== 提取的文本 ===")
	fmt.Println(text)
	fmt.Println("\n=== 调试结束 ===")
}
