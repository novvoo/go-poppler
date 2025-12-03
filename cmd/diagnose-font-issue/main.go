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

	page, err := doc.GetPage(1)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("===========================================")
	fmt.Println("     字体问题诊断")
	fmt.Println("===========================================\n")

	// 检查字体资源
	if page.Resources == nil {
		fmt.Println("❌ 页面没有资源字典")
		os.Exit(1)
	}

	fontsObj := page.Resources.Get("Font")
	if fontsObj == nil {
		fmt.Println("❌ 页面没有字体资源")
		os.Exit(1)
	}

	fontsDict, err := doc.ResolveObject(fontsObj)
	if err != nil {
		fmt.Printf("❌ 无法解析字体字典: %v\n", err)
		os.Exit(1)
	}

	fonts, ok := fontsDict.(pdf.Dictionary)
	if !ok {
		fmt.Println("❌ 字体资源不是字典类型")
		os.Exit(1)
	}

	fmt.Printf("✓ 找到 %d 个字体\n\n", len(fonts))

	// 分析每个字体
	fontRenderer := pdf.NewFontRenderer(150)

	for name := range fonts {
		fmt.Printf("【字体: %s】\n", name)

		fontRef := fonts.Get(string(name))
		if fontRef == nil {
			fmt.Println("  ❌ 无法获取字体引用")
			continue
		}

		fontObj, err := doc.ResolveObject(fontRef)
		if err != nil {
			fmt.Printf("  ❌ 无法解析字体对象: %v\n", err)
			continue
		}

		fontDict, ok := fontObj.(pdf.Dictionary)
		if !ok {
			fmt.Println("  ❌ 字体对象不是字典类型")
			continue
		}

		// 基本信息
		if baseFont, ok := fontDict.GetName("BaseFont"); ok {
			fmt.Printf("  BaseFont: %s\n", baseFont)
		}

		if subtype, ok := fontDict.GetName("Subtype"); ok {
			fmt.Printf("  Subtype: %s\n", subtype)
		}

		if encoding, ok := fontDict.GetName("Encoding"); ok {
			fmt.Printf("  Encoding: %s\n", encoding)
		}

		// 检查嵌入字体
		hasEmbedded := false
		if fontDict.Get("FontFile") != nil {
			fmt.Println("  ✓ 有 FontFile (Type1)")
			hasEmbedded = true
		}
		if fontDict.Get("FontFile2") != nil {
			fmt.Println("  ✓ 有 FontFile2 (TrueType)")
			hasEmbedded = true

			// 尝试提取
			fontFile2 := fontDict.Get("FontFile2")
			obj, err := doc.ResolveObject(fontFile2)
			if err != nil {
				fmt.Printf("    ❌ 无法解析 FontFile2: %v\n", err)
			} else {
				if stream, ok := obj.(pdf.Stream); ok {
					data, err := stream.Decode()
					if err != nil {
						fmt.Printf("    ❌ 无法解码字体数据: %v\n", err)
					} else {
						fmt.Printf("    ✓ 字体数据大小: %d 字节\n", len(data))
					}
				}
			}
		}
		if fontDict.Get("FontFile3") != nil {
			fmt.Println("  ✓ 有 FontFile3 (OpenType/CFF)")
			hasEmbedded = true
		}

		if !hasEmbedded {
			fmt.Println("  ⚠️  没有嵌入字体，需要使用系统字体")
		}

		// 检查 ToUnicode
		if fontDict.Get("ToUnicode") != nil {
			fmt.Println("  ✓ 有 ToUnicode 映射")
		} else {
			fmt.Println("  ⚠️  没有 ToUnicode 映射")
		}

		// 尝试加载字体
		fmt.Println("  测试加载字体...")
		ttfFont, err := fontRenderer.LoadPDFFont(fontDict, doc)
		if err != nil {
			fmt.Printf("    ❌ 加载失败: %v\n", err)
		} else if ttfFont == nil {
			fmt.Println("    ⚠️  使用回退字体")
		} else {
			fmt.Println("    ✓ 加载成功")
		}

		fmt.Println()
	}

	fmt.Println("===========================================")
	fmt.Println("【诊断结论】")
	fmt.Println()
	fmt.Println("如果看到 '❌ 加载失败' 或 '⚠️ 使用回退字体'，")
	fmt.Println("说明字体加载有问题，可能导致渲染乱码。")
	fmt.Println()
	fmt.Println("常见原因:")
	fmt.Println("1. 嵌入字体格式不支持（CFF/Type1）")
	fmt.Println("2. 字体数据解码失败")
	fmt.Println("3. 系统字体路径不正确")
	fmt.Println("4. 回退字体不支持中文")
}
