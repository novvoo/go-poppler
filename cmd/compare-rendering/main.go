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

	fmt.Println("===========================================")
	fmt.Println("     渲染方式对比")
	fmt.Println("===========================================\n")

	doc, err := pdf.Open(os.Args[1])
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	renderer := pdf.NewRenderer(doc)
	renderer.SetResolution(150, 150)

	// 方法1: 使用poppler原生渲染（不包含自定义文本）
	fmt.Println("【方法1: Poppler原生渲染】")
	fmt.Println("正在渲染...")
	img1, err := renderer.RenderPage(1)
	if err != nil {
		fmt.Printf("❌ 失败: %v\n", err)
	} else {
		filename := "test/test_preview/page_1_native.png"
		err = renderer.SaveImage(img1, filename, "png")
		if err != nil {
			fmt.Printf("❌ 保存失败: %v\n", err)
		} else {
			fmt.Printf("✓ 已保存: %s\n", filename)
			fmt.Printf("  尺寸: %dx%d\n", img1.Width, img1.Height)
		}
	}

	// 方法2: 使用自定义文本渲染
	fmt.Println("\n【方法2: 自定义文本渲染】")
	fmt.Println("正在渲染...")
	img2, err := renderer.RenderPageWithText(1)
	if err != nil {
		fmt.Printf("❌ 失败: %v\n", err)
	} else {
		filename := "test/test_preview/page_1_custom.png"
		err = renderer.SaveImage(img2, filename, "png")
		if err != nil {
			fmt.Printf("❌ 保存失败: %v\n", err)
		} else {
			fmt.Printf("✓ 已保存: %s\n", filename)
			fmt.Printf("  尺寸: %dx%d\n", img2.Width, img2.Height)
		}
	}

	fmt.Println("\n===========================================")
	fmt.Println("【对比说明】")
	fmt.Println()
	fmt.Println("page_1_native.png:")
	fmt.Println("  - 使用poppler原生渲染")
	fmt.Println("  - 应该显示正常（如果poppler支持）")
	fmt.Println("  - 但可能缺少某些文本")
	fmt.Println()
	fmt.Println("page_1_custom.png:")
	fmt.Println("  - 使用自定义文本渲染")
	fmt.Println("  - 使用系统字体")
	fmt.Println("  - 如果乱码，说明字体加载或渲染有问题")
	fmt.Println()
	fmt.Println("请对比两个图片，确定问题所在。")
	fmt.Println("===========================================")
}
