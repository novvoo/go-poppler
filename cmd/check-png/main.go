package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <png文件路径>")
		fmt.Println("示例: go run main.go test/test_preview/page_1.png")
		os.Exit(1)
	}

	pngPath := os.Args[1]

	// 打开PNG文件
	file, err := os.Open(pngPath)
	if err != nil {
		fmt.Printf("无法打开文件: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 解码PNG
	img, err := png.Decode(file)
	if err != nil {
		fmt.Printf("无法解码PNG: %v\n", err)
		os.Exit(1)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Printf("=== PNG 文件分析 ===\n")
	fmt.Printf("文件路径: %s\n", pngPath)
	fmt.Printf("图片尺寸: %d x %d\n", width, height)
	fmt.Printf("图片类型: %T\n\n", img)

	// 分析像素颜色分布
	colorMap := make(map[color.Color]int)
	var nonWhitePixels int
	var blackPixels int

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			colorMap[c]++

			r, g, b, _ := c.RGBA()
			// 转换为8位值
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			// 统计非白色像素
			if r8 < 250 || g8 < 250 || b8 < 250 {
				nonWhitePixels++
			}

			// 统计黑色像素
			if r8 < 10 && g8 < 10 && b8 < 10 {
				blackPixels++
			}
		}
	}

	totalPixels := width * height
	fmt.Printf("=== 像素统计 ===\n")
	fmt.Printf("总像素数: %d\n", totalPixels)
	fmt.Printf("非白色像素: %d (%.2f%%)\n", nonWhitePixels, float64(nonWhitePixels)*100/float64(totalPixels))
	fmt.Printf("黑色像素: %d (%.2f%%)\n", blackPixels, float64(blackPixels)*100/float64(totalPixels))
	fmt.Printf("不同颜色数: %d\n\n", len(colorMap))

	// 显示前10种最常见的颜色
	fmt.Printf("=== 颜色分布 (前10种) ===\n")
	type colorCount struct {
		color color.Color
		count int
	}
	var colors []colorCount
	for c, count := range colorMap {
		colors = append(colors, colorCount{c, count})
	}

	// 简单排序
	for i := 0; i < len(colors)-1; i++ {
		for j := i + 1; j < len(colors); j++ {
			if colors[j].count > colors[i].count {
				colors[i], colors[j] = colors[j], colors[i]
			}
		}
	}

	for i := 0; i < 10 && i < len(colors); i++ {
		r, g, b, a := colors[i].color.RGBA()
		fmt.Printf("%d. RGBA(%d, %d, %d, %d) - %d 像素 (%.2f%%)\n",
			i+1, r>>8, g>>8, b>>8, a>>8, colors[i].count,
			float64(colors[i].count)*100/float64(totalPixels))
	}

	// 采样检查：显示图片中心区域的像素
	fmt.Printf("\n=== 中心区域采样 (50x50) ===\n")
	centerX := width / 2
	centerY := height / 2
	sampleSize := 25

	hasContent := false
	for y := centerY - sampleSize; y < centerY+sampleSize && y < height; y++ {
		if y < 0 {
			continue
		}
		for x := centerX - sampleSize; x < centerX+sampleSize && x < width; x++ {
			if x < 0 {
				continue
			}
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			if r8 < 250 || g8 < 250 || b8 < 250 {
				hasContent = true
				fmt.Printf("位置 (%d, %d): RGB(%d, %d, %d)\n", x, y, r8, g8, b8)
			}
		}
	}

	if !hasContent {
		fmt.Println("中心区域全是白色或接近白色")
	}

	// 扫描每一行，找出有内容的行
	fmt.Printf("\n=== 有内容的行分布 ===\n")
	rowsWithContent := 0
	firstContentRow := -1
	lastContentRow := -1

	for y := 0; y < height; y++ {
		hasRowContent := false
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			if r8 < 250 || g8 < 250 || b8 < 250 {
				hasRowContent = true
				break
			}
		}

		if hasRowContent {
			rowsWithContent++
			if firstContentRow == -1 {
				firstContentRow = y
			}
			lastContentRow = y
		}
	}

	fmt.Printf("有内容的行数: %d / %d\n", rowsWithContent, height)
	if firstContentRow != -1 {
		fmt.Printf("第一行内容: 第 %d 行\n", firstContentRow)
		fmt.Printf("最后一行内容: 第 %d 行\n", lastContentRow)
		fmt.Printf("内容高度: %d 像素\n", lastContentRow-firstContentRow+1)
	}

	// 扫描每一列，找出有内容的列
	fmt.Printf("\n=== 有内容的列分布 ===\n")
	colsWithContent := 0
	firstContentCol := -1
	lastContentCol := -1

	for x := 0; x < width; x++ {
		hasColContent := false
		for y := 0; y < height; y++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			if r8 < 250 || g8 < 250 || b8 < 250 {
				hasColContent = true
				break
			}
		}

		if hasColContent {
			colsWithContent++
			if firstContentCol == -1 {
				firstContentCol = x
			}
			lastContentCol = x
		}
	}

	fmt.Printf("有内容的列数: %d / %d\n", colsWithContent, width)
	if firstContentCol != -1 {
		fmt.Printf("第一列内容: 第 %d 列\n", firstContentCol)
		fmt.Printf("最后一列内容: 第 %d 列\n", lastContentCol)
		fmt.Printf("内容宽度: %d 像素\n", lastContentCol-firstContentCol+1)
	}

	// 结论
	fmt.Printf("\n=== 诊断结论 ===\n")
	if nonWhitePixels == 0 {
		fmt.Println("❌ 图片完全是白色的，没有任何内容被渲染")
	} else if nonWhitePixels < totalPixels/1000 {
		fmt.Println("⚠️  图片几乎是空白的，只有极少量内容")
	} else {
		fmt.Println("✓ 图片包含可见内容")
	}

	if blackPixels > 0 {
		fmt.Printf("✓ 检测到 %d 个黑色像素（可能是文字）\n", blackPixels)
	} else {
		fmt.Println("⚠️  没有检测到黑色像素（文字可能没有正确渲染）")
	}
}
