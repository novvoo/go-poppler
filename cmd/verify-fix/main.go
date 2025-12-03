package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <pdf文件路径>")
		os.Exit(1)
	}

	pdfPath := os.Args[1]

	fmt.Println("===========================================")
	fmt.Println("     修复效果验证")
	fmt.Println("===========================================\n")

	// 1. 使用 pdftotext 提取
	fmt.Println("【1. pdftotext 提取结果】")
	cmd := exec.Command("./pdftotext.exe", "-f", "1", "-l", "1", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	pdftotextResult := string(output)
	pdftotextLines := strings.Split(pdftotextResult, "\n")

	fmt.Printf("总行数: %d\n", len(pdftotextLines))
	fmt.Println("前10行:")
	for i := 0; i < 10 && i < len(pdftotextLines); i++ {
		fmt.Printf("  %2d: %s\n", i+1, pdftotextLines[i])
	}

	// 2. 使用我们的程序提取
	fmt.Println("\n【2. 程序提取结果】")
	doc, err := pdf.Open(pdfPath)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	extractor := pdf.NewTextExtractor(doc)
	programResult, err := extractor.ExtractPageText(1)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	programLines := strings.Split(programResult, "\n")

	fmt.Printf("总行数: %d\n", len(programLines))
	fmt.Println("前10行:")
	for i := 0; i < 10 && i < len(programLines); i++ {
		fmt.Printf("  %2d: %s\n", i+1, programLines[i])
	}

	// 3. 对比分析
	fmt.Println("\n【3. 对比分析】")

	// 比较前10行
	matchCount := 0
	for i := 0; i < 10 && i < len(pdftotextLines) && i < len(programLines); i++ {
		line1 := strings.TrimSpace(pdftotextLines[i])
		line2 := strings.TrimSpace(programLines[i])

		if line1 == line2 {
			matchCount++
		} else {
			// 检查是否大致相似（忽略空格差异）
			s1 := strings.ReplaceAll(line1, " ", "")
			s2 := strings.ReplaceAll(line2, " ", "")
			if s1 == s2 {
				matchCount++
				fmt.Printf("  行 %d: 内容相同但空格不同\n", i+1)
			} else {
				fmt.Printf("  行 %d: 不匹配\n", i+1)
				fmt.Printf("    pdftotext: %s\n", line1[:min(50, len(line1))])
				fmt.Printf("    程序:      %s\n", line2[:min(50, len(line2))])
			}
		}
	}

	fmt.Printf("\n前10行匹配度: %d/10 (%.0f%%)\n", matchCount, float64(matchCount)*10)

	// 4. 渲染测试
	fmt.Println("\n【4. 渲染测试】")
	fmt.Println("重新生成预览图...")

	renderer := pdf.NewRenderer(doc)
	renderer.SetResolution(150, 150)

	img, err := renderer.RenderPageWithText(1)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("✓ 渲染成功: %dx%d\n", img.Width, img.Height)

		// 保存
		filename := "test/test_preview/page_1_fixed.jpg"
		err = renderer.SaveImage(img, filename, "jpeg")
		if err != nil {
			fmt.Printf("保存失败: %v\n", err)
		} else {
			fmt.Printf("✓ 已保存: %s\n", filename)
			fmt.Println("\n请打开图片查看文字渲染效果是否正确")
		}
	}

	fmt.Println("\n===========================================")
	fmt.Println("验证完成")
	fmt.Println("===========================================")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
