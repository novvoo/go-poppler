package main

import (
	"fmt"
	"os"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <pdf文件路径>")
		os.Exit(1)
	}

	doc, err := pdf.Open(os.Args[1])
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	fmt.Println("===========================================")
	fmt.Println("     文本顺序分析")
	fmt.Println("===========================================\n")

	// 提取第一页文本
	extractor := pdf.NewTextExtractor(doc)
	text, err := extractor.ExtractPageText(1)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("【程序提取的文本（前800字符）】")
	fmt.Println("-------------------------------------------")
	if len(text) > 800 {
		fmt.Println(text[:800])
		fmt.Printf("\n... (总共 %d 字符)\n", len(text))
	} else {
		fmt.Println(text)
	}
	fmt.Println("-------------------------------------------")

	fmt.Println("\n【对比说明】")
	fmt.Println("请将上面的输出与 pdftotext 的输出对比：")
	fmt.Println("  ./pdftotext.exe -f 1 -l 1 -layout test/test.pdf -")
	fmt.Println("\n如果顺序不一致，说明文本位置排序算法需要进一步调整。")
}
