package main

import (
	"fmt"
	"image/png"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <png文件路径>")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Println("===========================================")
	fmt.Println("     PDF文字渲染诊断报告")
	fmt.Println("===========================================")
	fmt.Printf("\n文件: %s\n", os.Args[1])
	fmt.Printf("尺寸: %d x %d 像素\n\n", width, height)

	// 统计信息
	totalBlack := 0
	totalGray := 0
	totalNonWhite := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 < 10 && g8 < 10 && b8 < 10 {
				totalBlack++
			} else if r8 < 200 && g8 < 200 && b8 < 200 {
				totalGray++
			}

			if r8 < 250 || g8 < 250 || b8 < 250 {
				totalNonWhite++
			}
		}
	}

	totalPixels := width * height

	fmt.Println("【像素统计】")
	fmt.Printf("  总像素数:     %d\n", totalPixels)
	fmt.Printf("  黑色像素:     %d (%.2f%%) - 纯文字\n", totalBlack, float64(totalBlack)*100/float64(totalPixels))
	fmt.Printf("  灰色像素:     %d (%.2f%%) - 抗锯齿\n", totalGray, float64(totalGray)*100/float64(totalPixels))
	fmt.Printf("  非白色像素:   %d (%.2f%%) - 总内容\n\n", totalNonWhite, float64(totalNonWhite)*100/float64(totalPixels))

	// 分析文字行
	fmt.Println("【文字行分析】")
	textLines := 0
	emptyLines := 0

	for y := 0; y < height; y++ {
		hasText := false
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 < 10 && g8 < 10 && b8 < 10 {
				hasText = true
				break
			}
		}

		if hasText {
			textLines++
		} else {
			emptyLines++
		}
	}

	fmt.Printf("  有文字的行:   %d / %d (%.1f%%)\n", textLines, height, float64(textLines)*100/float64(height))
	fmt.Printf("  空白行:       %d / %d (%.1f%%)\n\n", emptyLines, height, float64(emptyLines)*100/float64(height))

	// 检查抗锯齿效果
	fmt.Println("【抗锯齿分析】")
	fmt.Println("  检查第126行（文字密集区）的抗锯齿效果:")

	y := 126
	if y < height {
		blackCount := 0
		grayCount := 0

		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 < 10 && g8 < 10 && b8 < 10 {
				blackCount++
			} else if r8 < 200 && g8 < 200 && b8 < 200 {
				grayCount++
			}
		}

		fmt.Printf("    黑色像素: %d\n", blackCount)
		fmt.Printf("    灰色像素: %d\n", grayCount)
		fmt.Printf("    抗锯齿比: %.2f (灰色/黑色)\n\n", float64(grayCount)/float64(blackCount))
	}

	// 诊断结论
	fmt.Println("【诊断结论】")

	if totalBlack == 0 {
		fmt.Println("  ❌ 严重问题: 没有检测到任何黑色像素")
		fmt.Println("     文字完全没有被渲染")
	} else if totalBlack < 1000 {
		fmt.Println("  ⚠️  警告: 黑色像素数量极少")
		fmt.Println("     文字可能只渲染了一小部分")
	} else {
		fmt.Println("  ✓ 文字已成功渲染")
		fmt.Printf("     检测到 %d 个黑色像素\n", totalBlack)
	}

	if totalGray > totalBlack*2 {
		fmt.Println("  ✓ 抗锯齿效果良好")
		fmt.Println("     灰色像素数量充足，文字边缘平滑")
	} else if totalGray > totalBlack {
		fmt.Println("  ✓ 有抗锯齿效果")
		fmt.Println("     文字边缘有一定平滑处理")
	} else {
		fmt.Println("  ⚠️  抗锯齿效果较弱")
		fmt.Println("     文字边缘可能比较锐利")
	}

	coverage := float64(textLines) / float64(height) * 100
	if coverage > 50 {
		fmt.Printf("  ✓ 文字覆盖率高 (%.1f%%)\n", coverage)
		fmt.Println("     页面包含大量文字内容")
	} else if coverage > 20 {
		fmt.Printf("  ✓ 文字覆盖率正常 (%.1f%%)\n", coverage)
	} else {
		fmt.Printf("  ⚠️  文字覆盖率较低 (%.1f%%)\n", coverage)
		fmt.Println("     页面可能主要是空白或图片")
	}

	fmt.Println("\n===========================================")
}
