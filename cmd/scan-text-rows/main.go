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

	fmt.Printf("图片尺寸: %d x %d\n\n", width, height)
	fmt.Println("扫描前200行的黑色像素分布:")

	for y := 0; y < 200 && y < height; y++ {
		blackCount := 0
		firstX := -1
		lastX := -1

		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if r8 < 10 && g8 < 10 && b8 < 10 {
				blackCount++
				if firstX == -1 {
					firstX = x
				}
				lastX = x
			}
		}

		if blackCount > 0 {
			fmt.Printf("行 %3d: %4d 个黑色像素, X范围 [%4d, %4d]\n", y, blackCount, firstX, lastX)
		}
	}
}
