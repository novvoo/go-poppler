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

	for y := 125; y <= 150; y++ {
		fmt.Printf("\n第 %d 行:\n", y)

		// 找出这一行的黑色像素位置
		var blackPositions []int
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 < 10 && g8 < 10 && b8 < 10 {
				blackPositions = append(blackPositions, x)
			}
		}

		if len(blackPositions) > 0 {
			fmt.Printf("  黑色像素数: %d\n", len(blackPositions))
			fmt.Printf("  前10个位置: ")
			for i := 0; i < 10 && i < len(blackPositions); i++ {
				fmt.Printf("%d ", blackPositions[i])
			}
			fmt.Println()

			// 显示前5个黑色像素周围的颜色值
			fmt.Println("  前5个黑色像素周围的RGB值:")
			for i := 0; i < 5 && i < len(blackPositions); i++ {
				x := blackPositions[i]
				fmt.Printf("    位置 %d: ", x)

				for dx := -2; dx <= 2; dx++ {
					px := x + dx
					if px >= 0 && px < width {
						c := img.At(px, y)
						r, g, b, _ := c.RGBA()
						r8 := uint8(r >> 8)
						g8 := uint8(g >> 8)
						b8 := uint8(b >> 8)
						fmt.Printf("(%d,%d,%d) ", r8, g8, b8)
					}
				}
				fmt.Println()
			}
		}

		if y > 130 {
			break
		}
	}
}
