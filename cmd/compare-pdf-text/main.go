package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <pdf文件路径>")
		fmt.Println("示例: go run main.go test/test.pdf")
		os.Exit(1)
	}

	pdfPath := os.Args[1]

	fmt.Println("===========================================")
	fmt.Println("     PDF文本内容提取")
	fmt.Println("===========================================")
	fmt.Printf("\nPDF文件: %s\n\n", pdfPath)

	// 使用pdftotext提取第一页文本
	cmd := exec.Command("./pdftotext.exe", "-f", "1", "-l", "1", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("错误: 无法提取PDF文本: %v\n", err)
		fmt.Println("\n尝试使用简单模式...")

		cmd = exec.Command("./pdftotext.exe", "-f", "1", "-l", "1", pdfPath, "-")
		output, err = cmd.Output()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}
	}

	text := string(output)
	lines := strings.Split(text, "\n")

	fmt.Println("【第一页文本内容】")
	fmt.Println("-------------------------------------------")

	lineCount := 0
	charCount := 0
	nonEmptyLines := 0

	for i, line := range lines {
		if i >= 50 {
			fmt.Printf("\n... 还有 %d 行\n", len(lines)-50)
			break
		}

		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmptyLines++
			charCount += len(trimmed)
		}

		lineCount++
		fmt.Printf("%3d: %s\n", i+1, line)
	}

	fmt.Println("-------------------------------------------")
	fmt.Printf("\n【统计信息】\n")
	fmt.Printf("  总行数: %d\n", len(lines))
	fmt.Printf("  非空行: %d\n", nonEmptyLines)
	fmt.Printf("  字符数: %d\n", charCount)

	fmt.Println("\n===========================================")
}
