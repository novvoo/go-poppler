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
	pdfFile := flag.String("pdf", "", "PDF 文件路径")
	outputDir := flag.String("output", "", "输出目录（默认为 PDF 文件所在目录的 preview 子目录）")
	maxPages := flag.Int("pages", 3, "预览的最大页数（0 表示全部）")
	dpi := flag.Float64("dpi", 150, "渲染分辨率 DPI")
	thumbDpi := flag.Float64("thumb-dpi", 72, "缩略图分辨率 DPI")
	format := flag.String("format", "jpeg", "输出图片格式（jpeg, png, ppm）")
	noSvg := flag.Bool("no-svg", false, "不生成 SVG 预览")
	noText := flag.Bool("no-text", false, "不显示文本预览")
	noThumb := flag.Bool("no-thumb", false, "不生成缩略图")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "PDF 预览工具 - 将 PDF 页面渲染为图像\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s -pdf test/test.pdf\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -pdf document.pdf -pages 5 -dpi 300\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -pdf file.pdf -output ./preview -no-svg\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s test/test.pdf\n", os.Args[0])
	}

	flag.Parse()

	// 检查 PDF 文件参数
	var pdfPath string
	if *pdfFile != "" {
		pdfPath = *pdfFile
	} else if flag.NArg() > 0 {
		pdfPath = flag.Arg(0)
	} else {
		// 默认使用 test/test.pdf
		pdfPath = filepath.Join("test", "test.pdf")
		fmt.Printf("未指定 PDF 文件，使用默认文件: %s\n", pdfPath)
	}

	// 检查文件是否存在
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		fmt.Printf("错误: PDF 文件不存在: %s\n", pdfPath)
		fmt.Printf("\n使用 -h 查看帮助信息\n")
		os.Exit(1)
	}

	fmt.Printf("正在打开 PDF 文件: %s\n", pdfPath)

	doc, err := pdf.Open(pdfPath)
	if err != nil {
		fmt.Printf("错误: 无法打开 PDF 文件: %v\n", err)
		os.Exit(1)
	}
	defer doc.Close()

	// 显示 PDF 信息
	fmt.Printf("\n=== PDF 文档信息 ===\n")
	fmt.Printf("版本: %s\n", doc.GetVersion())
	fmt.Printf("页数: %d\n", doc.NumPages())

	info := doc.GetInfo()
	if info.Title != "" {
		fmt.Printf("标题: %s\n", info.Title)
	}
	if info.Author != "" {
		fmt.Printf("作者: %s\n", info.Author)
	}
	if !info.CreationDate.IsZero() {
		fmt.Printf("创建日期: %s\n", info.CreationDate.Format("2006-01-02 15:04:05"))
	}

	// 创建输出目录
	var outDir string
	if *outputDir != "" {
		outDir = *outputDir
	} else {
		// 默认在 PDF 文件所在目录创建 preview 子目录
		pdfDir := filepath.Dir(pdfPath)
		pdfName := filepath.Base(pdfPath)
		pdfNameNoExt := pdfName[:len(pdfName)-len(filepath.Ext(pdfName))]
		outDir = filepath.Join(pdfDir, pdfNameNoExt+"_preview")
	}

	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		fmt.Printf("错误: 无法创建输出目录: %v\n", err)
		os.Exit(1)
	}

	// 计算要渲染的页数
	totalPages := doc.NumPages()
	pagesToRender := totalPages
	if *maxPages > 0 && *maxPages < totalPages {
		pagesToRender = *maxPages
	}

	// 渲染页面为 PNG 图像（包含文字）
	fmt.Printf("\n=== 渲染页面预览（包含文字）===\n")

	renderer := pdf.NewRenderer(doc)
	renderer.SetResolution(*dpi, *dpi)

	for i := 1; i <= pagesToRender; i++ {
		fmt.Printf("正在渲染第 %d/%d 页...\n", i, totalPages)

		// 使用新的文本渲染方法
		img, err := renderer.RenderPageWithText(i)
		if err != nil {
			fmt.Printf("  错误: %v\n", err)
			continue
		}

		// 保存图片
		ext := *format
		if ext == "jpeg" {
			ext = "jpg"
		}
		filename := filepath.Join(outDir, fmt.Sprintf("page_%d.%s", i, ext))
		err = renderer.SaveImage(img, filename, *format)
		if err != nil {
			fmt.Printf("  保存失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ 已保存: %s (%dx%d)\n", filename, img.Width, img.Height)
	}

	// 生成缩略图（包含文字）
	if !*noThumb {
		fmt.Printf("\n=== 生成缩略图（包含文字）===\n")

		thumbRenderer := pdf.NewRenderer(doc)
		thumbRenderer.SetResolution(*thumbDpi, *thumbDpi)

		for i := 1; i <= pagesToRender; i++ {
			fmt.Printf("正在生成第 %d/%d 页缩略图...\n", i, totalPages)

			// 使用新的文本渲染方法
			img, err := thumbRenderer.RenderPageWithText(i)
			if err != nil {
				fmt.Printf("  错误: %v\n", err)
				continue
			}

			// 保存缩略图
			ext := *format
			if ext == "jpeg" {
				ext = "jpg"
			}
			filename := filepath.Join(outDir, fmt.Sprintf("thumb_%d.%s", i, ext))
			err = thumbRenderer.SaveImage(img, filename, *format)
			if err != nil {
				fmt.Printf("  保存失败: %v\n", err)
				continue
			}

			fmt.Printf("  ✓ 已保存: %s (%dx%d)\n", filename, img.Width, img.Height)
		}
	}

	// 提取第一页文本预览
	if !*noText {
		fmt.Printf("\n=== 文本预览（第1页）===\n")

		extractor := pdf.NewTextExtractor(doc)
		text, err := extractor.ExtractPageText(1)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			// 显示前 500 个字符
			if len(text) > 500 {
				fmt.Printf("%s...\n", text[:500])
			} else {
				fmt.Printf("%s\n", text)
			}
		}
	}

	// 生成 SVG 预览（第一页）
	if !*noSvg {
		fmt.Printf("\n=== 生成 SVG 预览 ===\n")

		svgFile := filepath.Join(outDir, "page_1.svg")
		f, err := os.Create(svgFile)
		if err != nil {
			fmt.Printf("错误: 无法创建 SVG 文件: %v\n", err)
		} else {
			defer f.Close()

			cairoRenderer := pdf.NewCairoRenderer(doc, pdf.CairoOptions{
				Format:    "svg",
				FirstPage: 1,
			})

			err = cairoRenderer.Render(f)
			if err != nil {
				fmt.Printf("错误: SVG 渲染失败: %v\n", err)
			} else {
				fmt.Printf("✓ SVG 已保存: %s\n", svgFile)
				fmt.Printf("  可以在浏览器中打开查看\n")
			}
		}
	}

	fmt.Printf("\n=== 完成 ===\n")
	fmt.Printf("预览文件已保存到: %s\n", outDir)
	fmt.Printf("\n可以使用以下方式查看:\n")
	ext := *format
	if ext == "jpeg" {
		ext = "jpg"
	}
	fmt.Printf("  - 图像: 使用图片查看器打开 page_*.%s\n", ext)
	if !*noThumb {
		fmt.Printf("  - 缩略图: 使用图片查看器打开 thumb_*.%s\n", ext)
	}
	if !*noSvg {
		fmt.Printf("  - SVG: 使用浏览器打开 page_1.svg\n")
	}
}
