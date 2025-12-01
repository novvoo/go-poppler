// pdfthumbnail - 生成 PDF 缩略图
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

func main() {
	// 命令行参数
	firstPage := flag.Int("f", 1, "起始页码")
	lastPage := flag.Int("l", 0, "结束页码 (0 表示最后一页)")
	size := flag.Int("size", 128, "缩略图最大尺寸")
	format := flag.String("format", "png", "输出格式 (png, jpeg)")
	quality := flag.Int("quality", 85, "JPEG 质量 (1-100)")
	ownerPassword := flag.String("opw", "", "所有者密码")
	userPassword := flag.String("upw", "", "用户密码")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [选项] <PDF文件> <输出前缀>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n生成 PDF 页面缩略图\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	pdfFile := flag.Arg(0)
	outputPrefix := flag.Arg(1)

	// 打开 PDF 文件
	doc, err := pdf.Open(pdfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法打开 PDF 文件: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// 处理密码
	if *ownerPassword != "" || *userPassword != "" {
		password := *ownerPassword
		if password == "" {
			password = *userPassword
		}
		if err := doc.Decrypt(password); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 密码错误或无法解密: %v\n", err)
			os.Exit(1)
		}
	}

	// 确定页面范围
	numPages := doc.NumPages()
	if *lastPage == 0 || *lastPage > numPages {
		*lastPage = numPages
	}
	if *firstPage < 1 {
		*firstPage = 1
	}

	// 创建渲染器
	renderer := pdf.NewRenderer(doc)

	// 计算缩略图分辨率 (假设页面为 A4，72 DPI 基准)
	// 缩略图大小 / 页面大小 * 72 = 目标 DPI
	thumbnailDPI := float64(*size) / 11.0 * 72.0 / 72.0 // 约 11 英寸高的页面
	if thumbnailDPI < 10 {
		thumbnailDPI = 10
	}
	if thumbnailDPI > 72 {
		thumbnailDPI = 72
	}

	renderer.SetResolution(thumbnailDPI, thumbnailDPI)

	// 生成缩略图
	for pageNum := *firstPage; pageNum <= *lastPage; pageNum++ {
		img, err := renderer.RenderPage(pageNum)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 无法渲染第 %d 页: %v\n", pageNum, err)
			continue
		}

		// 缩放到目标大小
		img = resizeImage(img, *size)

		// 生成输出文件名
		var ext string
		switch *format {
		case "jpeg", "jpg":
			ext = "jpg"
		default:
			ext = "png"
		}

		outputFile := fmt.Sprintf("%s-%d.%s", outputPrefix, pageNum, ext)

		// 确保输出目录存在
		outputDir := filepath.Dir(outputFile)
		if outputDir != "" && outputDir != "." {
			os.MkdirAll(outputDir, 0755)
		}

		// 保存缩略图
		opts := &pdf.ImageSaveOptions{
			Format:  *format,
			Quality: *quality,
		}
		if err := renderer.SaveImageWithOptions(img, outputFile, opts); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 无法保存缩略图 %s: %v\n", outputFile, err)
			continue
		}

		fmt.Printf("已生成: %s\n", outputFile)
	}
}

// resizeImage 缩放图像到指定最大尺寸
func resizeImage(img *pdf.RenderedImage, maxSize int) *pdf.RenderedImage {
	if img == nil || img.Width == 0 || img.Height == 0 {
		return img
	}

	// 计算缩放比例
	var scale float64
	if img.Width > img.Height {
		scale = float64(maxSize) / float64(img.Width)
	} else {
		scale = float64(maxSize) / float64(img.Height)
	}

	if scale >= 1.0 {
		return img // 不需要缩放
	}

	newWidth := int(float64(img.Width) * scale)
	newHeight := int(float64(img.Height) * scale)

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	// 简单的最近邻缩放
	newData := make([]byte, newWidth*newHeight*3)
	for y := 0; y < newHeight; y++ {
		srcY := int(float64(y) / scale)
		if srcY >= img.Height {
			srcY = img.Height - 1
		}
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			if srcX >= img.Width {
				srcX = img.Width - 1
			}

			srcIdx := (srcY*img.Width + srcX) * 3
			dstIdx := (y*newWidth + x) * 3

			if srcIdx+2 < len(img.Data) && dstIdx+2 < len(newData) {
				newData[dstIdx] = img.Data[srcIdx]
				newData[dstIdx+1] = img.Data[srcIdx+1]
				newData[dstIdx+2] = img.Data[srcIdx+2]
			}
		}
	}

	return &pdf.RenderedImage{
		Width:  newWidth,
		Height: newHeight,
		Data:   newData,
	}
}
