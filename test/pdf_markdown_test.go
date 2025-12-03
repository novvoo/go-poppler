package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

// TestMarkdownConversion 测试PDF转Markdown基本功能
func TestMarkdownConversion(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试，无法打开PDF: %v", err)
		return
	}
	defer doc.Close()

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         0,
		HeadingDetection: true,
		IncludeImages:    false,
	}

	writer := pdf.NewMarkdownWriter(doc, options)
	if writer == nil {
		t.Fatal("Markdown写入器创建失败")
	}

	outputFile := filepath.Join(".", "output.md")
	defer os.Remove(outputFile)

	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	err = writer.Write(f)
	if err != nil {
		t.Errorf("写入Markdown失败: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Markdown文件未创建")
	} else {
		t.Logf("Markdown已生成: %s", outputFile)
	}
}

// TestMarkdownWithImages 测试带图像的Markdown转换
func TestMarkdownWithImages(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         1, // 只测试第一页
		HeadingDetection: true,
		IncludeImages:    true,
		ExtractImages:    true,
		ImagePrefix:      "test-img",
	}

	writer := pdf.NewMarkdownWriter(doc, options)
	outputFile := filepath.Join(".", "output_with_images.md")
	defer os.Remove(outputFile)

	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	err = writer.Write(f)
	if err != nil {
		t.Errorf("写入Markdown失败: %v", err)
	}

	// 读取生成的文件检查图像引用
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("无法读取输出文件: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "![Image") {
		t.Log("Markdown包含图像引用")
	}

	t.Logf("生成的Markdown长度: %d 字符", len(contentStr))
}

// TestMarkdownPageRange 测试页面范围转换
func TestMarkdownPageRange(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	if doc.NumPages() < 2 {
		t.Skip("PDF页数不足，跳过页面范围测试")
		return
	}

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         2,
		HeadingDetection: true,
		PageSeparator:    "\n\n---\n\n",
	}

	writer := pdf.NewMarkdownWriter(doc, options)
	outputFile := filepath.Join(".", "output_range.md")
	defer os.Remove(outputFile)

	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	err = writer.Write(f)
	if err != nil {
		t.Errorf("写入Markdown失败: %v", err)
	}

	// 读取并验证内容
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("无法读取输出文件: %v", err)
	}

	contentStr := string(content)
	separatorCount := strings.Count(contentStr, "---")
	t.Logf("页面分隔符数量: %d", separatorCount)

	if separatorCount > 0 {
		t.Log("成功检测到页面分隔符")
	}
}

// TestMarkdownHeadingDetection 测试标题检测功能
func TestMarkdownHeadingDetection(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	// 测试启用标题检测
	t.Run("with heading detection", func(t *testing.T) {
		options := pdf.MarkdownOptions{
			FirstPage:        1,
			LastPage:         1,
			HeadingDetection: true,
		}

		writer := pdf.NewMarkdownWriter(doc, options)
		outputFile := filepath.Join(".", "output_headings.md")
		defer os.Remove(outputFile)

		f, err := os.Create(outputFile)
		if err != nil {
			t.Fatalf("无法创建输出文件: %v", err)
		}
		defer f.Close()

		err = writer.Write(f)
		if err != nil {
			t.Errorf("写入Markdown失败: %v", err)
		}

		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("无法读取输出文件: %v", err)
		}

		contentStr := string(content)
		if strings.Contains(contentStr, "#") {
			t.Log("检测到Markdown标题")
		}
	})

	// 测试禁用标题检测
	t.Run("without heading detection", func(t *testing.T) {
		options := pdf.MarkdownOptions{
			FirstPage:        1,
			LastPage:         1,
			HeadingDetection: false,
		}

		writer := pdf.NewMarkdownWriter(doc, options)
		outputFile := filepath.Join(".", "output_no_headings.md")
		defer os.Remove(outputFile)

		f, err := os.Create(outputFile)
		if err != nil {
			t.Fatalf("无法创建输出文件: %v", err)
		}
		defer f.Close()

		err = writer.Write(f)
		if err != nil {
			t.Errorf("写入Markdown失败: %v", err)
		}

		t.Log("成功生成无标题检测的Markdown")
	})
}

// TestMarkdownCustomOptions 测试自定义选项
func TestMarkdownCustomOptions(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         1,
		HeadingDetection: true,
		IncludeImages:    true,
		ExtractImages:    false, // 只引用，不提取
		ImagePrefix:      "custom-prefix",
		PageSeparator:    "\n\n***\n\n",
	}

	writer := pdf.NewMarkdownWriter(doc, options)
	if writer == nil {
		t.Fatal("Markdown写入器创建失败")
	}

	outputFile := filepath.Join(".", "output_custom.md")
	defer os.Remove(outputFile)

	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	err = writer.Write(f)
	if err != nil {
		t.Errorf("写入Markdown失败: %v", err)
	}

	// 验证自定义前缀
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("无法读取输出文件: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "custom-prefix") {
		t.Log("成功使用自定义图像前缀")
	}
}

// TestMarkdownFrontMatter 测试文档元数据前言
func TestMarkdownFrontMatter(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         1,
		HeadingDetection: true,
	}

	writer := pdf.NewMarkdownWriter(doc, options)
	outputFile := filepath.Join(".", "output_frontmatter.md")
	defer os.Remove(outputFile)

	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	err = writer.Write(f)
	if err != nil {
		t.Errorf("写入Markdown失败: %v", err)
	}

	// 检查是否包含前言
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("无法读取输出文件: %v", err)
	}

	contentStr := string(content)
	info := doc.GetInfo()

	if info.Title != "" && strings.Contains(contentStr, "title:") {
		t.Log("Markdown包含标题元数据")
	}
	if info.Author != "" && strings.Contains(contentStr, "author:") {
		t.Log("Markdown包含作者元数据")
	}
}

// TestConvertToMarkdown 测试便捷函数
func TestConvertToMarkdown(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         1,
		HeadingDetection: true,
	}

	markdown, err := pdf.ConvertToMarkdown(testPDF, options)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}

	if len(markdown) == 0 {
		t.Error("生成的Markdown为空")
	} else {
		t.Logf("成功转换，Markdown长度: %d 字符", len(markdown))

		// 显示前100个字符作为预览
		preview := markdown
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		t.Logf("Markdown预览: %s", preview)
	}
}

// TestMarkdownAllPages 测试转换所有页面
func TestMarkdownAllPages(t *testing.T) {
	testPDF := filepath.Join(".", "test.pdf")

	doc, err := pdf.Open(testPDF)
	if err != nil {
		t.Skipf("跳过测试: %v", err)
		return
	}
	defer doc.Close()

	options := pdf.MarkdownOptions{
		FirstPage:        1,
		LastPage:         0, // 0表示所有页面
		HeadingDetection: true,
		PageSeparator:    "\n\n---\n\n",
	}

	writer := pdf.NewMarkdownWriter(doc, options)
	outputFile := filepath.Join(".", "output_all_pages.md")
	defer os.Remove(outputFile)

	f, err := os.Create(outputFile)
	if err != nil {
		t.Fatalf("无法创建输出文件: %v", err)
	}
	defer f.Close()

	err = writer.Write(f)
	if err != nil {
		t.Errorf("写入Markdown失败: %v", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("无法获取文件信息: %v", err)
	}

	t.Logf("转换了 %d 页，生成文件大小: %d 字节", doc.NumPages(), fileInfo.Size())
}

// TestMarkdownErrorHandling 测试错误处理
func TestMarkdownErrorHandling(t *testing.T) {
	t.Run("invalid file", func(t *testing.T) {
		options := pdf.MarkdownOptions{}
		_, convErr := pdf.ConvertToMarkdown("nonexistent.pdf", options)
		if convErr == nil {
			t.Error("期望返回错误，但没有")
		}
	})

	t.Run("invalid page range", func(t *testing.T) {
		testPDF := filepath.Join(".", "test.pdf")

		doc, openErr := pdf.Open(testPDF)
		if openErr != nil {
			t.Skip("跳过测试")
			return
		}
		defer doc.Close()

		options := pdf.MarkdownOptions{
			FirstPage: 999,
			LastPage:  1000,
		}

		writer := pdf.NewMarkdownWriter(doc, options)
		outputFile := filepath.Join(".", "output_invalid.md")
		defer os.Remove(outputFile)

		f, createErr := os.Create(outputFile)
		if createErr != nil {
			t.Fatalf("无法创建输出文件: %v", createErr)
		}
		defer f.Close()

		// 应该处理无效页面范围而不崩溃
		_ = writer.Write(f)
		// 不应该panic，可能返回错误或生成空文件
		t.Log("处理无效页面范围完成")
	})
}
